package basedata

import "github.com/paalgyula/summit/pkg/wow"

// ItemTemplate is the part of Item.dbc the server needs to place and show an
// item: which slot it goes to and what the client draws for it.
type ItemTemplate struct {
	Entry         uint32
	DisplayID     uint32
	InventoryType wow.InventoryType
	SheatheType   uint8
}

// LookupItem returns the template of an item entry, or nil when unknown.
func (bd *Store) LookupItem(entry uint32) *ItemTemplate {
	if bd == nil {
		return nil
	}

	return bd.items[entry]
}
