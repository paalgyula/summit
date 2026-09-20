package displayinfo_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/displayinfo"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

func TestResolverRaceModels(t *testing.T) {
	r := displayinfo.NewResolver()
	assert.NotNil(t, r)

	// Test Orc Male lookup
	info, ok := r.GetRaceModelInfo(wow.RaceOrc, wow.GenderMale)
	assert.True(t, ok)
	assert.Equal(t, "Orc", info.RaceName)
	assert.Equal(t, "Male", info.GenderName)
	assert.Equal(t, uint32(51), info.DisplayID)
	assert.Equal(t, "Character/Orc/Male/OrcMale.glb", info.ModelPath)

	// Test Human Female lookup
	infoHF, okHF := r.GetRaceModelInfo(wow.RaceHuman, wow.GenderFemale)
	assert.True(t, okHF)
	assert.Equal(t, uint32(50), infoHF.DisplayID)
	assert.Equal(t, "Character/Human/Female/HumanFemale.glb", infoHF.ModelPath)

	// Test GetAllRaces
	races := r.GetAllRaces()
	assert.GreaterOrEqual(t, len(races), 20) // 10 races * 2 genders
}

func TestResolveCharacterVisuals(t *testing.T) {
	r := displayinfo.NewResolver()

	p := &player.Player{
		Name:       "Tand",
		Race:       wow.RaceOrc,
		Class:      wow.ClassWarlock,
		Gender:     wow.GenderMale,
		Level:      1,
		Skin:       3,
		Face:       2,
		HairStyle:  6,
		HairColor:  3,
		FacialHair: 8,
	}
	p.InitInventory(nil)

	manifest := r.ResolveCharacterVisuals(p)
	assert.Equal(t, "Tand", manifest.Name)
	assert.Equal(t, "Orc", manifest.RaceName)
	assert.Equal(t, "Warlock", manifest.ClassName)
	assert.Equal(t, "Male", manifest.GenderName)
	assert.Equal(t, uint32(51), manifest.DisplayID)
	assert.Equal(t, "Character/Orc/Male/OrcMale.glb", manifest.ModelURL)
	assert.Equal(t, "Character/Orc/Male/OrcMaleSkin00_03.webp", manifest.SkinTextureURL)
	assert.Equal(t, "Character/Orc/Male/OrcMaleFace00_02.webp", manifest.FaceTextureURL)
	assert.Equal(t, "Character/Orc/Hair/OrcHair00_03.webp", manifest.HairTextureURL)
}
