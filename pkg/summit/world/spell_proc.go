package world

// ProcEventInfo holds information about what triggered a proc.
// Maps to AzerothCore's ProcEventInfo from SpellAuras.h.
type ProcEventInfo struct {
	Actor         Unit
	ActionTarget  Unit
	Spell         *Spell
	DamageInfo    *DamageInfo
	HealInfo      *HealInfo
	TypeMaskActor uint32
	TypeMaskTarget uint32
	SpellTypeMask uint32
	SpellPhaseMask uint32
	HitMask       uint32
}

// DamageInfo holds damage information for proc events.
// Maps to AzerothCore's DamageInfo from SpellMgr.h.
type DamageInfo struct {
	Attacker   Unit
	Victim     Unit
	Spell      *SpellInfo
	Damage     uint32
	OverDamage uint32
	School     SpellSchoolMask
	Absorb     uint32
	Resist     uint32
	Periodic   bool
	Crit       bool
	HitMask    uint32
}

// HealInfo holds heal information for proc events.
// Maps to AzerothCore's HealInfo from SpellMgr.h.
type HealInfo struct {
	Healer   Unit
	Target   Unit
	Spell    *SpellInfo
	Heal     uint32
	OverHeal uint32
	Absorb   uint32
	Crit     bool
}

// AuraProcEntry pairs an aura with the proc effect mask that should fire.
type AuraProcEntry struct {
	ProcEffectMask uint32
	Aura           *Aura
}

// AuraHolder provides access to the unit's applied auras.
// Implemented by registering a getter via RegisterAuraHolder.
type AuraHolder interface {
	GetAppliedAuras() []*Aura
}

// auraHolderFunc is the registered getter function type.
type auraHolderFunc func(Unit) []*Aura

var auraHolderGetters = make(map[interface{}]auraHolderFunc)

// RegisterAuraHolder registers a getter function for a unit type.
// The key is the unit's concrete type (e.g., *player.Player).
func RegisterAuraHolder(key interface{}, fn auraHolderFunc) {
	auraHolderGetters[key] = fn
}

// getAppliedAuras retrieves auras for a unit using the registered getter.
func getAppliedAuras(unit Unit) []*Aura {
	for key, fn := range auraHolderGetters {
		if unit == key {
			return fn(unit)
		}
	}
	if holder, ok := unit.(AuraHolder); ok {
		return holder.GetAppliedAuras()
	}
	return nil
}

// ProcSkillsAndAuras is the main entry point for proc processing.
// It iterates actor and target auras, finds which ones should proc, then triggers them.
// Maps to AzerothCore's Unit::ProcSkillsAndAuras.
func ProcSkillsAndAuras(actor, target Unit, typeMaskActor, typeMaskTarget uint32,
	spell *Spell, damage *DamageInfo, heal *HealInfo) {

	if actor == nil {
		return
	}

	actorEventInfo := ProcEventInfo{
		Actor:          actor,
		ActionTarget:   target,
		Spell:          spell,
		DamageInfo:     damage,
		HealInfo:       heal,
		TypeMaskActor:  typeMaskActor,
		TypeMaskTarget: typeMaskTarget,
	}

	actorAuras := GetProcAurasTriggeredOnEvent(actor, actorEventInfo)

	var targetAuras []AuraProcEntry
	var targetEventInfo ProcEventInfo
	if target != nil && typeMaskTarget != 0 {
		targetEventInfo = ProcEventInfo{
			Actor:          actor,
			ActionTarget:   target,
			Spell:          spell,
			DamageInfo:     damage,
			HealInfo:       heal,
			TypeMaskActor:  typeMaskTarget,
			TypeMaskTarget: typeMaskActor,
		}
		targetAuras = GetProcAurasTriggeredOnEvent(target, targetEventInfo)
	}

	for _, entry := range actorAuras {
		TriggerProcOnEvent(entry.Aura, actorEventInfo)
	}

	for _, entry := range targetAuras {
		TriggerProcOnEvent(entry.Aura, targetEventInfo)
	}
}

// GetProcAurasTriggeredOnEvent finds auras that should proc for the given event.
// Returns a list of AuraProcEntry pairs (aura + proc effect mask).
// Maps to AzerothCore's Unit::GetProcAurasTriggeredOnEvent.
func GetProcAurasTriggeredOnEvent(unit Unit, eventInfo ProcEventInfo) []AuraProcEntry {
	var result []AuraProcEntry

	auras := getAppliedAuras(unit)
	for _, aur := range auras {
		if aur == nil {
			continue
		}

		procMask := checkAuraProc(aur, eventInfo)
		if procMask != 0 {
			result = append(result, AuraProcEntry{
				ProcEffectMask: procMask,
				Aura:           aur,
			})
		}
	}

	return result
}

// checkAuraProc checks if an aura should proc for the given event and returns the proc effect mask.
func checkAuraProc(aur *Aura, eventInfo ProcEventInfo) uint32 {
	if aur.SpellInfo == nil {
		return 0
	}

	if aur.ProcFlags == 0 {
		return 0
	}

	procFlags := aur.ProcFlags

	if eventInfo.TypeMaskActor != 0 && procFlags&eventInfo.TypeMaskActor != 0 {
		return procFlags & eventInfo.TypeMaskActor
	}

	if eventInfo.TypeMaskTarget != 0 && procFlags&eventInfo.TypeMaskTarget != 0 {
		return procFlags & eventInfo.TypeMaskTarget
	}

	return 0
}

// TriggerProcOnEvent executes the proc for an aura.
// Maps to AzerothCore's Aura::TriggerProcOnEvent (simplified).
func TriggerProcOnEvent(aur *Aura, eventInfo ProcEventInfo) {
	if aur == nil || aur.SpellInfo == nil {
		return
	}

	if aur.ProcCharges > 0 {
		aur.ProcCharges--
	}
}

// CanProc returns true if this aura can proc for the given event.
func (aur *Aura) CanProc(eventInfo ProcEventInfo) bool {
	if aur.ProcFlags == 0 {
		return false
	}
	return checkAuraProc(aur, eventInfo) != 0
}

// GetProcFlags returns the proc flags for this aura.
func (aur *Aura) GetProcFlags() uint32 {
	return aur.ProcFlags
}

// GetProcChance returns the proc chance percentage for this aura.
func (aur *Aura) GetProcChance() uint32 {
	return aur.ProcChance
}

// ConsumeCharge decrements proc charges. Returns true if the aura should be removed.
func (aur *Aura) ConsumeCharge() bool {
	if aur.ProcCharges == 0 {
		return false
	}
	aur.ProcCharges--
	return aur.ProcCharges == 0
}
