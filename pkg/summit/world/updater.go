package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

type Updater struct {
	updateFlags wow.ObjectUpdateFlags
}

// BuildCreateObject builds a visibility-aware SMSG_UPDATE_OBJECT with a
// CreateObject or CreateObject2 block for the given player to the target.
func (upd *Updater) BuildCreateObject(source *player.Player, target *player.Player) *wow.Packet {
	isSelf := source.GUID() == target.GUID()

	ud := object.NewUpdateData()
	upd.buildCreateObjectBlock(source, target, isSelf, ud)

	return ud.BuildPacket()
}

// buildCreateObjectBlock writes a single UPDATETYPE_CREATE_OBJECT(2) block
// using visibility-filtered fields. Determines CREATE_OBJECT vs CREATE_OBJECT2
// based on AzerothCore's logic: self, pets, dynamic objects, corpses,
// specific game object types, and owned game objects use CREATE_OBJECT2.
func (upd *Updater) buildCreateObjectBlock(source *player.Player, target *player.Player, isSelf bool, ud *object.UpdateData) {
	updateType := wow.UpdateTypeCreateObject

	flags := source.Object.UpdateFlags()
	if isSelf {
		flags |= wow.UpdateFlagSelf
		updateType = wow.UpdateTypeCreateObject2
	}

	// Determine CREATE_OBJECT2 based on object type and relationship
	// (mirrors AzerothCore's BuildCreateUpdateBlockForPlayer logic)
	if flags&wow.UpdateFlagStationaryPosition != 0 {
		typeID := source.Object.ObjectTypeID()

		// DynamicObject, Corpse → CREATE_OBJECT2
		if typeID == wow.TypeIDDynamicoObject || typeID == wow.TypeIDCorpse {
			updateType = wow.UpdateTypeCreateObject2
		}

		// TODO: Pet detection — when target's pet GUID matches source GUID
		// if target.GetPetGUID() == source.GUID() {
		//     updateType = wow.UpdateTypeCreateObject2
		// }

		// GameObject type-based decisions
		if typeID == wow.TypeIDGameObject {
			goType := source.Object.GameObjectType()
			switch goType {
			case wow.GameObjectTypeTrap, wow.GameObjectTypeDuelArbiter,
				wow.GameObjectTypeFlagStand, wow.GameObjectTypeFlagDrop:
				updateType = wow.UpdateTypeCreateObject2
			default:
				// TODO: if go has an owner, use CREATE_OBJECT2
				// if source.GetOwner() != nil {
				//     updateType = wow.UpdateTypeCreateObject2
				// }
			}
		}

		// Units with an attack target get UPDATEFLAG_HAS_TARGET
		if source.Object.ObjectTypeID() == wow.TypeIDUnit || source.Object.ObjectTypeID() == wow.TypeIDPlayer {
			if source.AttackTarget != 0 {
				flags |= wow.UpdateFlagHasTarget
			}
		}
	}

	// Build the block in a temporary buffer
	buf := object.NewUpdateBlockBuffer()

	// Update type
	_ = buf.WriteOne(updateType)

	// GUID
	_ = buf.Write(source.GUID())

	// Object type ID
	_ = buf.WriteOne(int(source.Object.ObjectTypeID()))

	// Update flags
	_ = buf.Write(flags)

	// Movement update (flag, position, speeds, etc.)
	upd.buildMovementUpdate(source, flags, buf)

	// Values update — visibility-filtered
	upd.buildValuesUpdateFiltered(source, target, isSelf, buf)

	ud.AddUpdateBlock(buf.Bytes())
}

// buildValuesUpdateFiltered writes a values block using BuildFilteredUpdateMask
// so only fields visible to the target are included.
func (upd *Updater) buildValuesUpdateFiltered(source *player.Player, target *player.Player, isSelf bool, buf *object.UpdateBlockBuffer) {
	mask := source.Object.BuildFilteredUpdateMask(target.Object, isSelf)
	block := source.Object.BuildValuesUpdateBlock(mask, target.Object)
	buf.WriteBytes(block)
}

// BuildValuesUpdateObject builds an incremental SMSG_UPDATE_OBJECT with a
// Values-only block containing only changed fields visible to the target.
func (upd *Updater) BuildValuesUpdateObject(source *player.Player, target *player.Player) *wow.Packet {
	ud := object.NewUpdateData()

	buf := object.NewUpdateBlockBuffer()

	// Update type: Values
	_ = buf.WriteOne(wow.UpdateTypeValues)

	// GUID
	_ = buf.Write(source.GUID())

	// Values update — incremental, visibility-filtered
	mask := source.Object.BuildIncrementalUpdateMask(target.Object)
	block := source.Object.BuildValuesUpdateBlock(mask, target.Object)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}

// BuildDestroyObject builds an SMSG_DESTROY_OBJECT packet.
func BuildDestroyObject(obj *player.Player, onDeath bool) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerDestroyObject)

	_ = pkt.Write(obj.GUID())

	// If true, client calls CGUnit_C::OnDeath() (death animation)
	if onDeath {
		_ = pkt.WriteOne(1)
	} else {
		_ = pkt.WriteOne(0)
	}

	return pkt
}

// buildMovementUpdate writes the movement part of a player's create block.
func (upd *Updater) buildMovementUpdate(p *player.Player, flags wow.ObjectUpdateFlags, buf *object.UpdateBlockBuffer) {
	mv := &object.MovementBlock{
		Flags: p.MoveFlags &^ wow.MovementFlagOnTransport,
		Time:  uint32(time.Now().UnixMilli()),
		X:     p.Location.X,
		Y:     p.Location.Y,
		Z:     p.Location.Z,
		O:     p.Location.O,
	}

	if p.Unit != nil {
		mv.Speeds = p.Unit.Speed
	} else {
		mv.Speeds = object.DefaultUnitSpeeds()
	}

	if flags&wow.UpdateFlagSelf != 0 {
		mv.LowGUID = 0x15
	} else {
		mv.LowGUID = 0x08
	}

	object.WriteMovementBlock(buf, flags, mv)
}

// BuildItemCreateObject builds an SMSG_UPDATE_OBJECT with a CreateObject block for an item.
func (upd *Updater) BuildItemCreateObject(item *player.Item, target *player.Player) *wow.Packet {
	if item == nil || item.Object == nil {
		return nil
	}

	ud := object.NewUpdateData()
	upd.buildItemCreateBlock(item, target, ud)

	return ud.BuildPacket()
}

// BuildItemValuesUpdate builds an SMSG_UPDATE_OBJECT with a Values-only block
// carrying the item's fields for its owner. Used when an existing stack grows
// (e.g. a loot item merging into it) so the client learns the new stack count.
func (upd *Updater) BuildItemValuesUpdate(item *player.Item, target *player.Player) *wow.Packet {
	if item == nil || item.Object == nil || target == nil {
		return nil
	}

	// The owner sees the ITEM_FIELD_* owner-only fields (stack count, durability,
	// charges); BuildFilteredUpdateMask includes them when isOwner is true.
	isOwner := item.Owner == target.GUID()
	mask := item.Object.BuildFilteredUpdateMask(target.Object, isOwner)

	ud := object.NewUpdateData()
	buf := object.NewUpdateBlockBuffer()

	_ = buf.WriteOne(wow.UpdateTypeValues)
	_ = buf.Write(item.GUID())
	buf.WriteBytes(item.Object.BuildValuesUpdateBlock(mask, target.Object))

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}

// buildItemCreateBlock appends a CreateObject block for an item; the owner
// also gets the OWNER-only fields (stack count, durability, charges).
func (upd *Updater) buildItemCreateBlock(item *player.Item, target *player.Player, ud *object.UpdateData) {
	buf := object.NewUpdateBlockBuffer()

	// Update type
	_ = buf.WriteOne(wow.UpdateTypeCreateObject)

	// GUID
	_ = buf.Write(item.GUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDItem))

	// Items carry only a low GUID (Item::Item sets m_updateFlag = UPDATEFLAG_LOWGUID)
	flags := wow.ObjectUpdateFlags(wow.UpdateFlagLowGUID)
	_ = buf.Write(flags)

	object.WriteMovementBlock(buf, flags, &object.MovementBlock{LowGUID: item.GUID().Counter()})

	// Values update — visibility-filtered
	isOwner := item.Owner == target.GUID()
	mask := item.Object.BuildFilteredUpdateMask(target.Object, isOwner)
	block := item.Object.BuildValuesUpdateBlock(mask, target.Object)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())
}

// BuildSelfCreateObject builds the login SMSG_UPDATE_OBJECT for the player
// itself: like Player::BuildCreateUpdateBlockForPlayer it carries the
// inventory items first, then the player block that references them.
func (upd *Updater) BuildSelfCreateObject(p *player.Player) *wow.Packet {
	ud := object.NewUpdateData()

	p.UpdateInventoryFields()

	if p.Inventory != nil {
		for i := 0; i < player.InventorySlotTotal; i++ {
			if item := p.Inventory.GetItem(i); item != nil {
				upd.buildItemCreateBlock(item, p, ud)
			}
		}
	}

	upd.buildCreateObjectBlock(p, p, true, ud)

	return ud.BuildPacket()
}

// BuildInventoryUpdate builds an SMSG_UPDATE_OBJECT with a values update
// for the player's inventory fields. Uses the changesMask to only send
// fields that actually changed.
func (upd *Updater) BuildInventoryUpdate(p *player.Player) *wow.Packet {
	if p == nil || p.Object == nil {
		return nil
	}

	ud := object.NewUpdateData()
	buf := object.NewUpdateBlockBuffer()

	// Update type
	_ = buf.WriteOne(wow.UpdateTypeValues)

	// GUID
	_ = buf.Write(p.GUID())

	// Build mask for inventory-related fields
	mask := &object.UpdateMask{}
	mask.SetCount(uint32(p.Object.ValuesCount()))

	// Mark PlayerFieldInvSlotHead through + 45 (23 slots * 2)
	for i := 0; i < 46; i++ {
		field := int(object.PlayerFieldInvSlotHead) + i
		if field < p.Object.ValuesCount() {
			mask.SetBit(uint32(field))
		}
	}

	// Mark PlayerFieldPackSlot_1 through + 31 (16 slots * 2)
	for i := 0; i < 32; i++ {
		field := int(object.PlayerFieldPackSlot_1) + i
		if field < p.Object.ValuesCount() {
			mask.SetBit(uint32(field))
		}
	}

	// Worn gear (PLAYER_VISIBLE_ITEM_n entry + enchant, 19 slots * 2)
	for i := 0; i < player.EquipmentSlotEnd*2; i++ {
		field := int(object.PlayerVisibleItem1Entryid) + i
		if field < p.Object.ValuesCount() {
			mask.SetBit(uint32(field))
		}
	}

	block := p.Object.BuildValuesUpdateBlock(mask, p.Object)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}

// BuildNPCCreateObject builds an SMSG_UPDATE_OBJECT with a create block for an NPC.
func BuildNPCCreateObject(npc *NPC) *wow.Packet {
	ud := object.NewUpdateData()
	buf := object.NewUpdateBlockBuffer()

	_ = buf.WriteOne(wow.UpdateTypeCreateObject)
	_ = buf.Write(npc.GetGUID())
	_ = buf.WriteOne(int(wow.TypeIDUnit))

	flags := wow.ObjectUpdateFlags(wow.UpdateFlagLowGUID | wow.UpdateFlagLiving | wow.UpdateFlagStationaryPosition)
	_ = buf.Write(flags)

	speeds := object.DefaultUnitSpeeds()
	if npc.Unit != nil {
		for mt, v := range npc.Unit.Speed {
			if v > 0 {
				speeds[mt] = v
			}
		}
	}

	object.WriteMovementBlock(buf, flags, &object.MovementBlock{
		X: npc.X, Y: npc.Y, Z: npc.Z, O: npc.O,
		Speeds:  speeds,
		LowGUID: 0x0B,
	})

	// Values update — full mask (create block)
	mask := npc.Object.BuildFullUpdateMask()
	block := npc.Object.BuildValuesUpdateBlock(mask, nil)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}

// BuildGameObjectCreateObject builds an SMSG_UPDATE_OBJECT with a create block for a game object.
func BuildGameObjectCreateObject(gobj *GameObject) *wow.Packet {
	ud := object.NewUpdateData()
	buf := object.NewUpdateBlockBuffer()

	_ = buf.WriteOne(int(gobj.CreateUpdateType()))
	_ = buf.Write(gobj.GetGUID())
	_ = buf.WriteOne(int(wow.TypeIDGameObject))

	// Game objects carry a low GUID, a position (m_updateFlag in
	// GameObject::GameObject), and a packed rotation.
	flags := wow.ObjectUpdateFlags(
		wow.UpdateFlagLowGUID |
			wow.UpdateFlagStationaryPosition |
			wow.UpdateFlagPosition |
			wow.UpdateFlagRotation,
	)
	_ = buf.Write(flags)

	object.WriteMovementBlock(buf, flags, &object.MovementBlock{
		X: gobj.X, Y: gobj.Y, Z: gobj.Z, O: gobj.O,
		LowGUID:  uint32(gobj.ID),
		Rotation: gobj.PackedRotation,
	})

	// Values update — full mask (create block)
	mask := gobj.Object.BuildFullUpdateMask()
	block := gobj.Object.BuildValuesUpdateBlock(mask, nil)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}
