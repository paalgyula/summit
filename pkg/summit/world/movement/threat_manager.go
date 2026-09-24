package movement

import (
	"sync"
)

// ThreatTarget is the interface for units that can be on a threat list.
type ThreatTarget interface {
	GetGUID() interface{}
	IsAlive() bool
	GetPositionX() float32
	GetPositionY() float32
	GetPositionZ() float32
}

// ThreatManager manages the threat table for a creature.
// It handles threat addition, victim selection, and threat clearing.
type ThreatManager struct {
	mu       sync.RWMutex
	threats  map[interface{}]float64 // GUID -> threat value
	owner    interface{}
	victim   interface{} // current victim (highest threat)
}

// NewThreatManager creates a ThreatManager for the given owner.
func NewThreatManager(owner interface{}) *ThreatManager {
	return &ThreatManager{
		threats: make(map[interface{}]float64),
		owner:   owner,
	}
}

// AddThreat adds threat from an attacker. If the attacker is not on the list,
// it is added. If the new threat exceeds the current victim's threat by >10%,
// the victim is switched.
func (tm *ThreatManager) AddThreat(guid interface{}, amount float64) {
	if guid == nil || amount <= 0 {
		return
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.threats[guid] += amount
}

// ModifyThreat modifies existing threat for a GUID.
func (tm *ThreatManager) ModifyThreat(guid interface{}, amount float64) {
	if guid == nil {
		return
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.threats[guid]; exists {
		tm.threats[guid] += amount
		if tm.threats[guid] < 0 {
			tm.threats[guid] = 0
		}
	}
}

// GetThreat returns the threat value for a GUID.
func (tm *ThreatManager) GetThreat(guid interface{}) float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.threats[guid]
}

// RemoveThreat removes a GUID from the threat list.
func (tm *ThreatManager) RemoveThreat(guid interface{}) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.threats, guid)
}

// ClearThreat empties the threat table.
func (tm *ThreatManager) ClearThreat() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.threats = make(map[interface{}]float64)
	tm.victim = nil
}

// SelectVictim returns the GUID with the highest threat that is alive.
// It accepts a predicate function to check if a target is valid.
func (tm *ThreatManager) SelectVictim(isValid func(guid interface{}) bool) interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var bestGuid interface{}
	var bestThreat float64

	for guid, threat := range tm.threats {
		if threat > bestThreat && isValid(guid) {
			bestGuid = guid
			bestThreat = threat
		}
	}

	return bestGuid
}

// GetVictim returns the current victim.
func (tm *ThreatManager) GetVictim() interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.victim
}

// SetVictim sets the current victim.
func (tm *ThreatManager) SetVictim(victim interface{}) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.victim = victim
}

// IsOnThreatList returns true if the GUID is on the threat list.
func (tm *ThreatManager) IsOnThreatList(guid interface{}) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.threats[guid]
	return exists
}

// GetThreatList returns a copy of the threat list.
func (tm *ThreatManager) GetThreatList() map[interface{}]float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[interface{}]float64, len(tm.threats))
	for k, v := range tm.threats {
		result[k] = v
	}
	return result
}

// Size returns the number of entries on the threat list.
func (tm *ThreatManager) Size() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.threats)
}
