package basedata

import "github.com/paalgyula/summit/pkg/wow"

// LookupItem returns the template of an item entry from DBC, or nil when unknown.
func (bd *Store) LookupItem(entry uint32) *ItemTemplate {
	if bd == nil {
		return nil
	}

	return bd.items[entry]
}

// GetItemInventoryType returns the inventory type of an item.
func GetItemInventoryType(entry uint32) wow.InventoryType {
	tpl := GetInstance().LookupItem(entry)
	if tpl == nil {
		return wow.InventoryTypeNonEquip
	}

	return tpl.InventoryType
}

// GetItemTemplate returns the full item_template for an item entry.
func GetItemTemplate(entry uint32) *ItemTemplate {
	return GetInstance().LookupItem(entry)
}

// MergeItemTemplates overlays full item templates (item_template from the
// world store) on the DBC-derived ones: the DBC only knows display id,
// class and inventory type, the database has names, stats and prices.
func (bd *Store) MergeItemTemplates(templates map[uint32]*ItemTemplate) {
	if bd == nil || len(templates) == 0 {
		return
	}

	if bd.items == nil {
		bd.items = make(map[uint32]*ItemTemplate, len(templates))
	}

	for entry, t := range templates {
		if t == nil {
			continue
		}

		// Keep the DBC display id when the database row has none
		if prev := bd.items[entry]; prev != nil && t.DisplayID == 0 {
			t.DisplayID = prev.DisplayID
		}

		bd.items[entry] = t
	}
}
