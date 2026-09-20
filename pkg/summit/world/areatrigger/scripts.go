package areatrigger

// AreaTriggerScript is the interface for areatrigger scripts.
type AreaTriggerScript interface {
	// OnTrigger is called when a player enters the areatrigger.
	// Returns true if the trigger should be blocked (prevents default action).
	OnTrigger(playerID uint32, trigger *AreaTrigger) bool
}

// OnlyOnceAreaTriggerScript is a script that only triggers once per instance.
type OnlyOnceAreaTriggerScript interface {
	AreaTriggerScript

	// OnFirstTrigger is called only the first time a player enters the trigger.
	// Returns true if the trigger should be blocked.
	OnFirstTrigger(playerID uint32, trigger *AreaTrigger) bool
}

// SimpleAreaTriggerScript is a helper for simple areatrigger scripts.
type SimpleAreaTriggerScript struct {
	OnTriggerFunc func(playerID uint32, trigger *AreaTrigger) bool
}

// OnTriger implements AreaTriggerScript.
func (s *SimpleAreaTriggerScript) OnTrigger(playerID uint32, trigger *AreaTrigger) bool {
	if s.OnTriggerFunc != nil {
		return s.OnTriggerFunc(playerID, trigger)
	}

	return false
}

// OnlyOnceScript wraps a script to only trigger once per instance.
type OnlyOnceScript struct {
	Script      OnlyOnceAreaTriggerScript
	Manager     *Manager
	InstanceKey string
}

// OnTrigger implements AreaTriggerScript.
func (o *OnlyOnceScript) OnTrigger(playerID uint32, trigger *AreaTrigger) bool {
	if o.Manager.IsTriggerDone(o.InstanceKey, trigger.Entry) {
		return true
	}

	o.Manager.MarkTriggerDone(o.InstanceKey, trigger.Entry)

	if o.Script != nil {
		return o.Script.OnFirstTrigger(playerID, trigger)
	}

	return false
}
