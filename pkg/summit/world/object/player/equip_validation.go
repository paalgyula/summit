package player

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
)

// EquipResult represents the result of an equip validation check.
type EquipResult int

const (
	EquipResultOK              EquipResult = 0
	EquipResultCantEquipLevel  EquipResult = 1
	EquipResultCantEquipSkill  EquipResult = 2
	EquipResultCantEquipClass  EquipResult = 3
	EquipResultCantEquipRace   EquipResult = 4
	EquipResultCantEquipSlot   EquipResult = 5
	EquipResultInventoryFull   EquipResult = 6
	EquipResultAlreadyEquipped EquipResult = 7
	EquipResultWrongSlot       EquipResult = 8
	EquipResultTwoHandRequired EquipResult = 9
	EquipResultCantDualWield   EquipResult = 10
	EquipResultLocked          EquipResult = 11

	// AzerothCore InventoryResult codes used by item-use validation
	// (SMSG_INVENTORY_CHANGE_FAILURE). Values match AC Item.h InventoryResult.
	EquipResultYouCanNeverUse EquipResult = 10
	EquipResultNoRequiredProf EquipResult = 8
	EquipResultItemNotFound   EquipResult = 23
	EquipResultYouAreDead     EquipResult = 38
	EquipResultCantDoRightNow EquipResult = 39
	EquipResultNotInCombat    EquipResult = 60
)

// getValidEquipSlots returns the valid equipment slots for an item template.
func getValidEquipSlots(tpl *basedata.ItemTemplate) [4]int {
	var slots [4]int

	for i := range slots {
		slots[i] = -1
	}

	if tpl == nil {
		return slots
	}

	inventoryType := tpl.InventoryType

	switch inventoryType {
	case wow.InventoryTypeHead:
		slots[0] = EquipmentSlotHead
	case wow.InventoryTypeNeck:
		slots[0] = EquipmentSlotNeck
	case wow.InventoryTypeShoulders:
		slots[0] = EquipmentSlotShoulder
	case wow.InventoryTypeBody:
		slots[0] = EquipmentSlotBody
	case wow.InventoryTypeChest, wow.InventoryTypeRobe:
		slots[0] = EquipmentSlotChest
	case wow.InventoryTypeWaist:
		slots[0] = EquipmentSlotWaist
	case wow.InventoryTypeLegs:
		slots[0] = EquipmentSlotLegs
	case wow.InventoryTypeFeet:
		slots[0] = EquipmentSlotFeet
	case wow.InventoryTypeWrists:
		slots[0] = EquipmentSlotWrists
	case wow.InventoryTypeHands:
		slots[0] = EquipmentSlotHands
	case wow.InventoryTypeFinger:
		slots[0] = EquipmentSlotFinger1
		slots[1] = EquipmentSlotFinger2
	case wow.InventoryTypeTrinket:
		slots[0] = EquipmentSlotTrinket1
		slots[1] = EquipmentSlotTrinket2
	case wow.InventoryTypeCloak:
		slots[0] = EquipmentSlotBack
	case wow.InventoryTypeWeapon:
		slots[0] = EquipmentSlotMainHand
	case wow.InventoryTypeShield, wow.InventoryTypeWeaponOffHand, wow.InventoryTypeHoldable:
		slots[0] = EquipmentSlotOffHand
	case wow.InventoryTypeRanged, wow.InventoryTypeRangedRight, wow.InventoryTypeThrown, wow.InventoryTypeRelic:
		slots[0] = EquipmentSlotRanged
	case wow.InventoryTypeTabard:
		slots[0] = EquipmentSlotTabard
	case wow.InventoryType2hweapon:
		slots[0] = EquipmentSlotMainHand
	case wow.InventoryTypeWeaponMainHand:
		slots[0] = EquipmentSlotMainHand
	}

	return slots
}

// FindEquipSlotForItem returns the best equipment slot for an item, preferring
// empty slots. If the requested slot is set and valid, it tries that first.
// Returns -1 if no valid slot exists.
func FindEquipSlotForItem(p *Player, tpl *basedata.ItemTemplate, requestedSlot int) int {
	if tpl == nil {
		return -1
	}

	slots := getValidEquipSlots(tpl)

	// Check if any valid slot exists
	hasValidSlot := false

	for _, s := range slots {
		if s >= 0 {
			hasValidSlot = true

			break
		}
	}

	if !hasValidSlot {
		return -1
	}

	// If a specific slot was requested, check if it's valid for this item
	if requestedSlot >= 0 && requestedSlot < EquipmentSlotEnd {
		for _, s := range slots {
			if s == requestedSlot {
				return s
			}
		}

		// Requested slot is not valid for this item type
		return -1
	}

	// No specific slot requested - find first free slot
	for _, s := range slots {
		if s >= 0 && p.Inventory.GetEquipment(s) == nil {
			return s
		}
	}

	// All occupied, return first matching (for swap)
	for _, s := range slots {
		if s >= 0 {
			return s
		}
	}

	return -1
}

// CanEquipItem checks if a player can equip an item at the given slot.
// The tpl parameter is the item template (from basedata).
func CanEquipItem(p *Player, item *Item, tpl *basedata.ItemTemplate, slot int) EquipResult {
	if item == nil {
		return EquipResultWrongSlot
	}

	if tpl == nil {
		return EquipResultWrongSlot
	}

	// Level check
	if tpl.RequiredLevel > 0 && p.Level < uint8(tpl.RequiredLevel) {
		return EquipResultCantEquipLevel
	}

	// Class check
	if tpl.AllowableClass != -1 {
		classBit := int32(1) << (p.Class - 1)
		if tpl.AllowableClass&classBit == 0 {
			return EquipResultCantEquipClass
		}
	}

	// Race check
	if tpl.AllowableRace != -1 {
		raceBit := int32(1) << (p.Race - 1)
		if tpl.AllowableRace&raceBit == 0 {
			return EquipResultCantEquipRace
		}
	}

	// Combat check - some items can't be equipped in combat
	if p.InCombat && !tpl.CanChangeEquipStateInCombat() {
		return EquipResultCantEquipSlot
	}

	// Validate slot is correct for this item type
	if slot >= 0 && slot < EquipmentSlotEnd {
		correctSlot := FindEquipSlotForItem(p, tpl, slot)
		if correctSlot != slot {
			return EquipResultWrongSlot
		}
	}

	return EquipResultOK
}

// CanUseItem checks whether the player may use an item with the given template
// (AzerothCore Player::CanUseItem template overload): alive, class/race, level,
// required skill/spell. slot is unused and kept for API symmetry with equip.
func CanUseItem(p *Player, _ *Item, tpl *basedata.ItemTemplate) EquipResult {
	if p == nil || tpl == nil {
		return EquipResultItemNotFound
	}

	if !p.IsAlive() {
		return EquipResultYouAreDead
	}

	if tpl.AllowableClass != -1 {
		classBit := int32(1) << (p.Class - 1)
		if tpl.AllowableClass&classBit == 0 {
			return EquipResultYouCanNeverUse
		}
	}

	if tpl.AllowableRace != -1 {
		raceBit := int32(1) << (p.Race - 1)
		if tpl.AllowableRace&raceBit == 0 {
			return EquipResultYouCanNeverUse
		}
	}

	if tpl.RequiredLevel > 0 && p.Level < uint8(tpl.RequiredLevel) {
		return EquipResultCantEquipLevel
	}

	// Required skill/proficiency is not modelled yet; skip RequiredSkillRank.

	return EquipResultOK
}

// CanEquipItemInSlot checks if a player can equip an item in a specific slot.
func CanEquipItemInSlot(p *Player, item *Item, tpl *basedata.ItemTemplate, slot int) EquipResult {
	if item == nil || tpl == nil {
		return EquipResultWrongSlot
	}

	// Validate slot is correct for this item type
	correctSlot := FindEquipSlotForItem(p, tpl, slot)
	if correctSlot != slot {
		return EquipResultWrongSlot
	}

	return CanEquipItem(p, item, tpl, slot)
}

// EquipItemWithValidation validates and equips an item.
// tpl is the item template (from basedata). Returns the previously equipped
// item (for swap) or nil, and the result code.
func (p *Player) EquipItemWithValidation(item *Item, tpl *basedata.ItemTemplate, slot int) (*Item, EquipResult) {
	if item == nil {
		return nil, EquipResultWrongSlot
	}

	if tpl == nil {
		return nil, EquipResultWrongSlot
	}

	// Find the best slot for this item
	equipSlot := FindEquipSlotForItem(p, tpl, slot)
	if equipSlot < 0 {
		return nil, EquipResultWrongSlot
	}

	// Validate
	result := CanEquipItemInSlot(p, item, tpl, equipSlot)
	if result != EquipResultOK {
		return nil, result
	}

	// Perform the equip
	existing := p.Inventory.GetEquipment(equipSlot)

	// Remember the source slot before SetEquipment changes item.SlotIndex
	srcSlot := item.SlotIndex

	// Place item in equipment slot
	p.Inventory.SetEquipment(equipSlot, item)

	// Apply stat modifications
	p.ApplyItemMods(item, equipSlot, true)

	// If there was an item already there, move it to the source slot
	if existing != nil {
		p.ApplyItemMods(existing, equipSlot, false)
		p.Inventory.SetItem(srcSlot, existing)
		existing.UpdateFields()
	} else {
		// Clear the source slot (where the item came from)
		if srcSlot >= 0 {
			p.Inventory.RemoveItem(srcSlot)
		}
	}

	item.UpdateFields()
	p.UpdateInventoryFields()

	return existing, EquipResultOK
}

// GetFreeInventorySlot returns the first empty backpack slot, or -1 if full.
func (p *Player) GetFreeInventorySlot() int {
	for i := InventorySlotItemStart; i < InventorySlotItemEnd; i++ {
		if p.Inventory.GetItem(i) == nil {
			return i
		}
	}

	return -1
}

// HasItemCount returns how many of an item entry the player has (equipped + inventory).
func (p *Player) HasItemCount(entry uint32) int {
	count := 0

	for i := 0; i < InventorySlotTotal; i++ {
		item := p.Inventory.GetItem(i)
		if item != nil && item.ItemEntry == entry {
			count += int(item.StackCount)
		}
	}

	return count
}

// String returns a human-readable error message for the equip result.
func (r EquipResult) String() string {
	switch r {
	case EquipResultOK:
		return "OK"
	case EquipResultCantEquipLevel:
		return "You must reach a higher level to use this item."
	case EquipResultCantEquipSkill:
		return "You don't have the required skill to use this item."
	case EquipResultCantEquipClass:
		return "Your class cannot use this item."
	case EquipResultCantEquipRace:
		return "Your race cannot use this item."
	case EquipResultCantEquipSlot:
		return "There is no equipment slot open for that item."
	case EquipResultInventoryFull:
		return "Your inventory is full."
	case EquipResultAlreadyEquipped:
		return "You already have that item equipped."
	case EquipResultWrongSlot:
		return "You can't equip that in that slot."
	case EquipResultTwoHandRequired:
		return "You need to equip a two-handed weapon."
	case EquipResultCantDualWield:
		return "You cannot dual wield."
	case EquipResultLocked:
		return "That item is locked."
	default:
		return fmt.Sprintf("Unknown equip error %d", int(r))
	}
}
