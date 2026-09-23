package loot

// StoreType identifies one of the *_loot_template tables the manager holds.
type StoreType uint8

const (
	// StoreCreature is creature_loot_template (keyed by creature_template.lootid).
	StoreCreature StoreType = iota
	// StoreGameObject is gameobject_loot_template.
	StoreGameObject
	// StoreItem is item_loot_template.
	StoreItem
	// StoreReference is reference_loot_template, resolved for `-ref` rows.
	StoreReference
)

// String returns the store's table base name.
func (t StoreType) String() string {
	switch t {
	case StoreCreature:
		return "creature"
	case StoreGameObject:
		return "gameobject"
	case StoreItem:
		return "item"
	case StoreReference:
		return "reference"
	default:
		return "unknown"
	}
}

// Manager owns every loot template table and resolves loot for creatures, game
// objects and items. Reference (`-ref`) entries are followed into the
// reference_loot_template store.
type Manager struct {
	stores map[StoreType]*LootStore
}

// NewManager creates an empty manager with one store per loot table.
func NewManager() *Manager {
	return &Manager{
		stores: map[StoreType]*LootStore{
			StoreCreature:   NewLootStore(StoreCreature.String()),
			StoreGameObject: NewLootStore(StoreGameObject.String()),
			StoreItem:       NewLootStore(StoreItem.String()),
			StoreReference:  NewLootStore(StoreReference.String()),
		},
	}
}

// Store returns the LootStore backing a table (never nil for a valid type).
func (m *Manager) Store(t StoreType) *LootStore {
	return m.stores[t]
}

// Add registers a single loot template row.
func (m *Manager) Add(t StoreType, e LootEntry) {
	if s := m.stores[t]; s != nil {
		s.AddEntry(e)
	}
}

// GetLootFor returns the template for a loot id, or nil.
func (m *Manager) GetLootFor(t StoreType, id uint32) *LootTemplate {
	s := m.stores[t]
	if s == nil {
		return nil
	}

	return s.GetLootFor(id)
}

// HasLoot reports whether the given table has a template for the loot id.
func (m *Manager) HasLoot(t StoreType, id uint32) bool {
	s := m.stores[t]
	return s != nil && s.HasLootFor(id)
}

// Count returns the number of loot ids in a table (used for logging).
func (m *Manager) Count(t StoreType) int {
	s := m.stores[t]
	if s == nil {
		return 0
	}

	return len(s.templates)
}

// FillLoot rolls the template of lootID into l, resolving references against
// the reference store. It reports false when the table has no such loot id.
func (m *Manager) FillLoot(l *Loot, t StoreType, lootID uint32, mode LootMode) bool {
	tpl := m.GetLootFor(t, lootID)
	if tpl == nil {
		return false
	}

	var refs *LootStore

	if t != StoreReference {
		refs = m.stores[StoreReference]
	}

	tpl.ProcessWithRefs(l, mode, refs)

	return true
}
