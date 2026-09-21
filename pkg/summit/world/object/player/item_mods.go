package player

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
)

// ApplyItemMods applies or removes an item's stat modifications on the player.
// When apply is true, stats are added; when false, they are removed.
func (p *Player) ApplyItemMods(item *Item, slot int, apply bool) {
	if item == nil || slot < 0 || slot >= EquipmentSlotEnd {
		return
	}

	tpl := basedata.GetInstance().LookupItem(item.ItemEntry)
	if tpl == nil {
		return
	}

	// Apply/remove stat bonuses
	p.applyItemStats(tpl, apply)

	// Apply/remove armor
	p.applyItemArmor(tpl, apply)

	// Apply/remove resistances
	p.applyItemResistances(tpl, apply)

	// Apply/remove damage (weapons)
	p.applyItemDamage(tpl, slot, apply)
}

// statToUnitField maps item stat types to UnitFieldStat indices.
// Stat types: 0=mana, 1=health, 3=agi, 4=str, 5=int, 6=spi, 7=sta
func statToUnitField(statType uint32) (object.UpdateField, bool) {
	switch statType {
	case 3: // Agility
		return object.UnitFieldStat1, true
	case 4: // Strength
		return object.UnitFieldStat0, true
	case 5: // Intellect
		return object.UnitFieldStat3, true
	case 6: // Spirit
		return object.UnitFieldStat4, true
	case 7: // Stamina
		return object.UnitFieldStat2, true
	default:
		return 0, false
	}
}

// applyItemStats applies or removes stat bonuses from an item.
func (p *Player) applyItemStats(tpl *basedata.ItemTemplate, apply bool) {
	for _, stat := range tpl.Stats {
		if stat.Type == 0 {
			continue
		}

		field, ok := statToUnitField(stat.Type)
		if !ok {
			continue
		}

		value := int32(stat.Value)
		if !apply {
			value = -value
		}

		p.ModifyStat(field, value)
	}
}

// applyItemArmor applies or removes armor value from an item.
func (p *Player) applyItemArmor(tpl *basedata.ItemTemplate, apply bool) {
	if tpl.Armor == 0 {
		return
	}

	value := int32(tpl.Armor)
	if !apply {
		value = -value
	}

	p.ModifyStat(object.UnitFieldResistances, value) // Armor is at index 0 of resistances
}

// applyItemResistances applies or removes resistance values from an item.
func (p *Player) applyItemResistances(tpl *basedata.ItemTemplate, apply bool) {
	resistances := []struct {
		offset int
		value  int32
	}{
		{1, tpl.HolyRes},   // Index 1 = Holy
		{2, tpl.FireRes},   // Index 2 = Fire
		{3, tpl.NatureRes}, // Index 3 = Nature
		{4, tpl.FrostRes},  // Index 4 = Frost
		{5, tpl.ShadowRes}, // Index 5 = Shadow
		{6, tpl.ArcaneRes}, // Index 6 = Arcane
	}

	for _, r := range resistances {
		if r.value == 0 {
			continue
		}

		value := r.value
		if !apply {
			value = -value
		}

		p.ModifyStat(object.UnitFieldResistances+object.UpdateField(r.offset), value)
	}
}

// applyItemDamage applies or removes weapon damage values.
func (p *Player) applyItemDamage(tpl *basedata.ItemTemplate, slot int, apply bool) {
	if !tpl.IsWeapon() {
		return
	}

	for _, dmg := range tpl.Damage {
		if dmg.Min == 0 && dmg.Max == 0 {
			continue
		}

		if apply {
			p.Object.SetFloatValue(object.UnitFieldMindamage, dmg.Min)
			p.Object.SetFloatValue(object.UnitFieldMaxdamage, dmg.Max)
		} else {
			p.Object.SetFloatValue(object.UnitFieldMindamage, 0)
			p.Object.SetFloatValue(object.UnitFieldMaxdamage, 0)
		}

		// Only apply first damage entry
		break
	}
}

// ModifyStat modifies a stat field on the player and updates the update fields.
func (p *Player) ModifyStat(field object.UpdateField, value int32) {
	current := int32(p.Object.GetUInt32Value(field))
	newVal := uint32(current + value)
	p.Object.SetUInt32Value(field, newVal)
}
