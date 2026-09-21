package world

import (
	"fmt"
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

//nolint:funlen,wsl,cyclop,errcheck
func (upd *Updater) buildMovementUpdate(unit any, flags wow.ObjectUpdateFlags, buf *object.UpdateBlockBuffer) {
	var o *object.Object

	var u *object.Unit

	var p *player.Player

	switch t := unit.(type) {
	case *player.Player:
		o = t.Object
		u = t.Unit
		p = t
	case *object.Object:
		o = t
	default:
		panic(fmt.Sprintf("unknown type: %T", unit))
	}

	_ = flags

	if flags&wow.UpdateFlagLiving != 0 {
		moveFlags := o.MovementFlags()

		//nolint:exhaustive
		switch o.GUID().TypeID() {
		case wow.TypeIDUnit:
			moveFlags &= ^wow.MovementFlagOnTransport
		case wow.TypeIDPlayer:
			if p != nil && p.Transport() != nil {
				moveFlags |= wow.MovementFlagOnTransport
			} else {
				moveFlags &= ^wow.MovementFlagOnTransport
			}
		}

		_ = buf.Write(moveFlags)                      // movement flags
		_ = buf.WriteOne(0)                           // extra movement flags
		_ = buf.Write(uint32(time.Now().UnixMilli())) // time (in milliseconds)

		// Transport data (when MOVEMENTFLAG_ONTRANSPORT is set)
		if moveFlags&wow.MovementFlagOnTransport != 0 {
			if p != nil && p.Transport() != nil {
				// TODO: write transport packed GUID + offsets (X/Y/Z/O)
				// For now write zeros as placeholder
				_ = buf.WriteOne(0) // transport GUID (packed, empty)
				_ = buf.Write(float32(0)) // offset X
				_ = buf.Write(float32(0)) // offset Y
				_ = buf.Write(float32(0)) // offset Z
				_ = buf.Write(float32(0)) // offset O
				_ = buf.Write(uint32(0))  // transport seat
			}
		}

		// Swimming / flying pitch (when SWIMMING or FLYING2)
		if moveFlags&(wow.MovementFlagSwimming|wow.MovementFlagFlying2) != 0 {
			// TODO: read actual pitch from movement state
			_ = buf.Write(float32(0)) // pitch
		}

		// Fall time — always written for players
		if o.GUID().TypeID() == wow.TypeIDPlayer {
			// TODO: read actual fall time from movement state
			_ = buf.Write(uint32(0)) // fallTime

			// Falling data (velocity, sin/cos angle, xyspeed)
			if moveFlags&(wow.MovementFlagFalling|wow.MovementFlagFallingFar) != 0 {
				// TODO: read actual fall data from movement state
				_ = buf.Write(float32(0)) // velocity Z
				_ = buf.Write(float32(0)) // sinAngle
				_ = buf.Write(float32(0)) // cosAngle
				_ = buf.Write(float32(0)) // xyspeed
			}

			// Spline elevation
			if moveFlags&wow.MovementFlagSplineElevation != 0 {
				// TODO: read actual spline elevation from movement state
				_ = buf.Write(float32(0)) // u_unk1
			}
		}

		// Unit speeds
		if u != nil {
			_ = buf.Write(u.GetSpeed(wow.MoveTypeWalk))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeRun))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeRunBack))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeSwim))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeSwimBack))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeFlight))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeFlightBack))
			_ = buf.Write(u.GetSpeed(wow.MoveTypeTurnRate))
			_ = buf.Write(float32(0)) // pitch rate (not implemented)
		}

		// Spline data
		if moveFlags&wow.MovementFlagSplineEnabled != 0 {
			// TODO: write spline data from MoveSpline
			// PacketBuilder::WriteCreate(*unit->movespline, *data)
			_ = buf.Write(uint32(0)) // spline facing
			_ = buf.Write(float32(0)) // spline x
			_ = buf.Write(float32(0)) // spline y
			_ = buf.Write(float32(0)) // spline z
			_ = buf.Write(uint32(0)) // spline time
			_ = buf.Write(uint32(0)) // spline id
		}
	}

	// UPDATEFLAG_POSITION (0x100) — transport-attached position
	if flags&wow.UpdateFlagPosition != 0 {
		// Transport GUID
		if p != nil && p.Transport() != nil {
			// TODO: write transport packed GUID
			_ = buf.WriteOne(0)
		} else {
			_ = buf.WriteOne(0)
		}

		// Position
		if p != nil {
			_ = buf.Write(p.Location.X)
			_ = buf.Write(p.Location.Y)
			_ = buf.Write(p.Location.Z)
		} else {
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
		}

		// Transport offsets or position
		if p != nil && p.Transport() != nil {
			// TODO: write transport offsets
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
		} else if p != nil {
			_ = buf.Write(p.Location.X)
			_ = buf.Write(p.Location.Y)
			_ = buf.Write(p.Location.Z)
		} else {
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
		}

		// Orientation
		if p != nil {
			_ = buf.Write(p.Location.O)
		} else {
			_ = buf.Write(float32(0))
		}

		// Corpse orientation (written for corpses, 0 for others)
		if o.ObjectTypeID() == wow.TypeIDCorpse {
			if p != nil {
				_ = buf.Write(p.Location.O)
			} else {
				_ = buf.Write(float32(0))
			}
		} else {
			_ = buf.Write(float32(0))
		}
	} else if flags&wow.UpdateFlagStationaryPosition != 0 {
		// Write stationary position from the object's stored position.
		if p != nil {
			_ = buf.Write(p.Location.X)
			_ = buf.Write(p.Location.Y)
			_ = buf.Write(p.Location.Z)
			_ = buf.Write(p.Location.O)
		} else {
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
			_ = buf.Write(float32(0))
		}
	}

	// UPDATEFLAG_UNKNOWN (0x8)
	if flags&wow.UpdateFlagUnknown != 0 {
		_ = buf.WriteUint32(0)
	}

	// Low GUID
	if flags&wow.UpdateFlagLowGUID != 0 {
		//nolint:exhaustive
		switch o.GUID().TypeID() {
		case wow.TypeIDObject, wow.TypeIDItem, wow.TypeIDContainer,
			wow.TypeIDGameObject, wow.TypeIDDynamicoObject, wow.TypeIDCorpse:
			_ = buf.Write(o.GUID().Entry())
		case wow.TypeIDUnit:
			_ = buf.WriteUint32(0x0B)
		case wow.TypeIDPlayer:
			if flags&wow.UpdateFlagSelf != 0 {
				_ = buf.WriteUint32(0x15)
			} else {
				_ = buf.WriteUint32(0x08)
			}
		default:
			_ = buf.WriteUint32(0x00)
		}
	}

	// UPDATEFLAG_HAS_TARGET (0x4) — packed GUID of attack target
	if flags&wow.UpdateFlagHasTarget != 0 {
		// TODO: write victim's packed GUID
		// For now write empty GUID (1 byte zero)
		_ = buf.WriteOne(0)
	}

	// UPDATEFLAG_TRANSPORT (0x2) — transport path progress
	if flags&wow.UpdateFlagTransport != 0 {
		// TODO: write transport path progress (uint32 ms)
		_ = buf.WriteUint32(0)
	}

	// UPDATEFLAG_VEHICLE (0x80) — vehicle ID + orientation
	if flags&wow.UpdateFlagVehicle != 0 {
		// TODO: write vehicle ID from VehicleKit
		_ = buf.WriteUint32(0)
		// Write orientation (or transport offset O if on transport)
		if moveFlags := o.MovementFlags(); moveFlags&wow.MovementFlagOnTransport != 0 {
			_ = buf.Write(float32(0)) // transport offset O
		} else {
			if p != nil {
				_ = buf.Write(p.Location.O)
			} else {
				_ = buf.Write(float32(0))
			}
		}
	}

	// UPDATEFLAG_ROTATION (0x200) — packed world rotation for game objects
	if flags&wow.UpdateFlagRotation != 0 {
		// TODO: write packed world rotation (int64)
		_ = buf.Write(int64(0))
	}
}

// BuildItemCreateObject builds an SMSG_UPDATE_OBJECT with a CreateObject block for an item.
func (upd *Updater) BuildItemCreateObject(item *player.Item, target *player.Player) *wow.Packet {
	if item == nil || item.Object == nil {
		return nil
	}

	ud := object.NewUpdateData()
	buf := object.NewUpdateBlockBuffer()

	// Update type
	_ = buf.WriteOne(wow.UpdateTypeCreateObject)

	// GUID
	_ = buf.Write(item.GUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDItem))

	// Update flags for items: LowGUID | StationaryPosition (items don't move)
	flags := wow.ObjectUpdateFlags(wow.UpdateFlagLowGUID | wow.UpdateFlagStationaryPosition)
	_ = buf.Write(flags)

	// Stationary position (items don't move)
	_ = buf.Write(float32(0)) // X
	_ = buf.Write(float32(0)) // Y
	_ = buf.Write(float32(0)) // Z
	_ = buf.Write(float32(0)) // O

	// Low GUID
	_ = buf.Write(item.GUID().Entry())

	// High GUID
	_ = buf.WriteUint32(0)

	// Values update — visibility-filtered
	mask := item.Object.BuildFilteredUpdateMask(target.Object, false)
	block := item.Object.BuildValuesUpdateBlock(mask, target.Object)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

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

	block := p.Object.BuildValuesUpdateBlock(mask, p.Object)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}

// BuildNPCCreateObject builds an SMSG_UPDATE_OBJECT with a create block for an NPC.
func BuildNPCCreateObject(npc *NPC) *wow.Packet {
	ud := object.NewUpdateData()
	buf := object.NewUpdateBlockBuffer()

	// Update type
	_ = buf.WriteOne(wow.UpdateTypeCreateObject)

	// GUID
	_ = buf.Write(npc.GetGUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDUnit))

	// Update flags
	flags := wow.ObjectUpdateFlags(wow.UpdateFlagLowGUID | wow.UpdateFlagLiving | wow.UpdateFlagStationaryPosition)
	_ = buf.Write(flags)

	// Movement flags
	_ = buf.Write(wow.MovementFlagNone)
	_ = buf.WriteOne(0)                           // extra movement flags
	_ = buf.Write(uint32(0))                      // time

	// Stationary position
	_ = buf.Write(float32(npc.X))
	_ = buf.Write(float32(npc.Y))
	_ = buf.Write(float32(npc.Z))
	_ = buf.Write(float32(npc.O))

	// Unit speeds
	_ = buf.Write(float32(2.5))  // walk
	_ = buf.Write(float32(7.0))  // run
	_ = buf.Write(float32(4.5))  // run back
	_ = buf.Write(float32(4.7))  // swim
	_ = buf.Write(float32(2.5))  // swim back
	_ = buf.Write(float32(7.0))  // flight
	_ = buf.Write(float32(4.5))  // flight back
	_ = buf.Write(float32(7.0))  // turn rate

	// Low GUID
	_ = buf.WriteUint32(0x0B)

	// High GUID
	_ = buf.WriteUint32(0x00)

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

	// Update type
	_ = buf.WriteOne(wow.UpdateTypeCreateObject)

	// GUID
	_ = buf.Write(gobj.GetGUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDGameObject))

	// Update flags
	flags := wow.ObjectUpdateFlags(wow.UpdateFlagLowGUID | wow.UpdateFlagStationaryPosition | wow.UpdateFlagRotation)
	_ = buf.Write(flags)

	// Stationary position
	_ = buf.Write(float32(gobj.X))
	_ = buf.Write(float32(gobj.Y))
	_ = buf.Write(float32(gobj.Z))
	_ = buf.Write(float32(gobj.O))

	// Low GUID
	_ = buf.WriteUint32(0x0B)

	// High GUID
	_ = buf.WriteUint32(0x00)

	// Values update — full mask (create block)
	mask := gobj.Object.BuildFullUpdateMask()
	block := gobj.Object.BuildValuesUpdateBlock(mask, nil)
	buf.WriteBytes(block)

	ud.AddUpdateBlock(buf.Bytes())

	return ud.BuildPacket()
}
