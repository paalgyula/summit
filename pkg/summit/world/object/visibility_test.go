package object_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/stretchr/testify/assert"
)

func TestFieldFlags_Public(t *testing.T) {
	// Object fields should be PUBLIC
	flags := object.UnitUpdateFieldFlags
	assert.Equal(t, uint32(object.UFFlagPublic), flags[object.UnitFieldHealth-object.ObjectEnd])
	assert.Equal(t, uint32(object.UFFlagPublic), flags[object.UnitFieldLevel-object.ObjectEnd])
	assert.Equal(t, uint32(object.UFFlagPublic), flags[object.UnitFieldFactiontemplate-object.ObjectEnd])
}

func TestFieldFlags_Private(t *testing.T) {
	// Some fields are PRIVATE (only visible to self)
	flags := object.UnitUpdateFieldFlags
	assert.Equal(t, uint32(object.UFFlagPrivate), flags[object.UnitFieldCritter-object.ObjectEnd])
}

func TestGetUpdateFieldData_Self(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(int(object.PlayerEnd))

	// When target is self, visibleFlag should include PRIVATE
	visibleFlag := object.GetUpdateFieldDataForTesting(obj, obj)
	assert.True(t, visibleFlag&object.UFFlagPrivate != 0, "self should see PRIVATE fields")
	assert.True(t, visibleFlag&object.UFFlagPublic != 0, "self should see PUBLIC fields")
}

func TestGetUpdateFieldData_OtherPlayer(t *testing.T) {
	self := object.NewObject()
	self.InitValues(int(object.PlayerEnd))

	other := object.NewObject()
	other.InitValues(int(object.PlayerEnd))

	// When target is another player, visibleFlag should NOT include PRIVATE
	visibleFlag := object.GetUpdateFieldDataForTesting(self, other)
	assert.False(t, visibleFlag&object.UFFlagPrivate != 0, "other player should NOT see PRIVATE fields")
	assert.True(t, visibleFlag&object.UFFlagPublic != 0, "other player should see PUBLIC fields")
}

func TestFieldFlags_HealthIsPublic(t *testing.T) {
	// Health should be visible to everyone (PUBLIC) — not PRIVATE
	flags := object.UnitUpdateFieldFlags
	healthIdx := object.UnitFieldHealth - object.ObjectEnd
	assert.Equal(t, uint32(object.UFFlagPublic), flags[healthIdx], "health should be PUBLIC")
}

func TestFieldFlags_CritterIsPrivate(t *testing.T) {
	// Critter (pet GUID) is PRIVATE — only visible to owner
	flags := object.UnitUpdateFieldFlags
	critterIdx := object.UnitFieldCritter - object.ObjectEnd
	assert.Equal(t, uint32(object.UFFlagPrivate), flags[critterIdx], "critter should be PRIVATE")
}
