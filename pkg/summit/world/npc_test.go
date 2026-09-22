package world

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/wow"
)

// The grid visibility path builds NPC create blocks with the per-type field
// visibility table, so the NPC's object must identify itself as a unit or the
// client only receives the object header (no display id, health or npc flags).
func TestNPCCreateMaskContainsUnitFields(t *testing.T) {
	npc := NewNPC(3098, "Mottled Boar", 903, 14, 1, 42, 0, 0, 0, 0, 1, quest.NpcFlagQuestGiver)

	if got := npc.Object.ObjectTypeID(); got != wow.TypeIDUnit {
		t.Fatalf("object type id = %v, want %v", got, wow.TypeIDUnit)
	}

	viewer := object.NewObject()
	viewer.InitValues(int(object.PlayerEnd))
	viewer.SetObjectTypeID(wow.TypeIDPlayer)

	mask := npc.Object.BuildFilteredUpdateMask(viewer, false)

	for _, f := range []object.UpdateField{
		object.ObjectFieldEntry,
		object.UnitFieldDisplayid,
		object.UnitFieldHealth,
		object.UnitFieldMaxhealth,
		object.UnitFieldLevel,
		object.UnitFieldFactiontemplate,
		object.UnitFieldFlags,
		object.UnitNpcFlags,
	} {
		if !mask.GetBit(uint32(f)) {
			t.Errorf("field %d missing from NPC create mask", f)
		}
	}
}
