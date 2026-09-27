package vendor

import (
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/rs/zerolog/log"
)

// VendorItemInstance represents an item in an active vendor's inventory,
// including dynamic stock tracking and restock timers.
type VendorItemInstance struct {
	Item         store.VendorItem
	CurrentCount uint32
	LastRestock  time.Time
}

// Manager maintains vendor inventories, stock counts, and restock timers.
type Manager struct {
	mu      sync.RWMutex
	vendors map[uint32][]*VendorItemInstance // creature entry -> items
}

// NewManager creates a new vendor manager.
func NewManager(worldStore store.WorldRepo) *Manager {
	mgr := &Manager{
		vendors: make(map[uint32][]*VendorItemInstance),
	}

	mgr.Load(worldStore)

	return mgr
}

// Load loads vendor items from the world store into memory.
func (m *Manager) Load(worldStore store.WorldRepo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if worldStore == nil {
		m.loadFallbackVendors()
		return
	}

	itemsMap, err := worldStore.GetAllVendorItems()
	if err != nil {
		log.Warn().Err(err).Msg("vendor: failed to load vendor items from world store, using fallback vendors")
		m.loadFallbackVendors()

		return
	}

	if len(itemsMap) == 0 {
		log.Info().Msg("vendor: no vendor items found in store, loading default starter vendors")
		m.loadFallbackVendors()

		return
	}

	now := time.Now()

	for entry, items := range itemsMap {
		var instances []*VendorItemInstance

		for _, item := range items {
			instances = append(instances, &VendorItemInstance{
				Item:         item,
				CurrentCount: item.MaxCount,
				LastRestock:  now,
			})
		}

		m.vendors[entry] = instances
	}

	log.Info().Int("vendors", len(m.vendors)).Msg("vendor: loaded vendor data from world store")
}

// HasVendorItems returns true if the creature entry sells any items.
func (m *Manager) HasVendorItems(entry uint32) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items, exists := m.vendors[entry]
	return exists && len(items) > 0
}

// GetVendorItems returns the current list of items sold by the creature entry,
// updating any expired restock timers for limited stock items.
func (m *Manager) GetVendorItems(entry uint32) []*VendorItemInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	items, exists := m.vendors[entry]
	if !exists {
		return nil
	}

	now := time.Now()

	for _, it := range items {
		if it.Item.MaxCount > 0 && it.CurrentCount < it.Item.MaxCount && it.Item.IncrTime > 0 {
			elapsed := now.Sub(it.LastRestock)
			interval := time.Duration(it.Item.IncrTime) * time.Second

			if elapsed >= interval {
				ticks := uint32(elapsed / interval)
				it.CurrentCount += ticks

				if it.CurrentCount > it.Item.MaxCount {
					it.CurrentCount = it.Item.MaxCount
				}

				it.LastRestock = now
			}
		}
	}

	// Return a copy of the slice
	res := make([]*VendorItemInstance, len(items))
	copy(res, items)

	return res
}

// GetVendorItem returns a specific vendor item at vendorslot (0-indexed).
func (m *Manager) GetVendorItem(entry uint32, slot uint32) *VendorItemInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items, exists := m.vendors[entry]
	if !exists || int(slot) >= len(items) {
		return nil
	}

	return items[slot]
}

// UpdateStock decrements the stock of an item if limited.
// Returns the new remaining count and whether the purchase was allowed.
func (m *Manager) UpdateStock(entry uint32, slot uint32, count uint32) (uint32, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	items, exists := m.vendors[entry]
	if !exists || int(slot) >= len(items) {
		return 0, false
	}

	it := items[slot]
	if it.Item.MaxCount == 0 {
		// Unlimited stock
		return 0xFFFFFFFF, true
	}

	if it.CurrentCount < count {
		return it.CurrentCount, false
	}

	it.CurrentCount -= count
	if it.CurrentCount == 0 {
		it.LastRestock = time.Now()
	}

	return it.CurrentCount, true
}

// AddVendorItem adds or appends an item to a vendor.
func (m *Manager) AddVendorItem(entry uint32, item store.VendorItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	it := &VendorItemInstance{
		Item:         item,
		CurrentCount: item.MaxCount,
		LastRestock:  time.Now(),
	}

	m.vendors[entry] = append(m.vendors[entry], it)
}

// loadFallbackVendors initializes baseline vendors (food/drink/reagents/starter gear)
// for testing and standalone mode when no database is connected.
func (m *Manager) loadFallbackVendors() {
	now := time.Now()

	// Default vendor items: Tough Jerky (117), Refreshing Spring Water (159),
	// Hearthstone (6948), Heavy Silk Bandage (8544), Rough Arrow (2512), Rough Shot (2515)
	starterItems := []store.VendorItem{
		{Entry: 0, Slot: 1, Item: 117, MaxCount: 0, IncrTime: 0},  // Tough Jerky
		{Entry: 0, Slot: 2, Item: 159, MaxCount: 0, IncrTime: 0},  // Refreshing Spring Water
		{Entry: 0, Slot: 3, Item: 2512, MaxCount: 0, IncrTime: 0}, // Rough Arrow
		{Entry: 0, Slot: 4, Item: 2515, MaxCount: 0, IncrTime: 0}, // Rough Shot
	}

	// Common starter area vendor creature entries:
	// 4265 (Innkeeper Farley), 6740 (Innkeeper Allison), 1247 (Innkeeper Boomer), etc.
	for _, entry := range []uint32{4265, 6740, 1247, 54, 56, 124, 151, 152} {
		var instances []*VendorItemInstance

		for _, it := range starterItems {
			itCopy := it
			itCopy.Entry = entry
			instances = append(instances, &VendorItemInstance{
				Item:         itCopy,
				CurrentCount: itCopy.MaxCount,
				LastRestock:  now,
			})
		}

		m.vendors[entry] = instances
	}
}
