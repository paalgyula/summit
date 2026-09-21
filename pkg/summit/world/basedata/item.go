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
