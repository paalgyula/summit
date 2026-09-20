package displayinfo

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// RaceGenderKey uniquely identifies a playable race + gender combination.
type RaceGenderKey struct {
	Race   wow.PlayerRace
	Gender wow.PlayerGender
}

// RaceModelInfo holds the 3D model and texture information for a race/gender combo.
type RaceModelInfo struct {
	Race        wow.PlayerRace   `json:"race"`
	RaceName    string           `json:"raceName"`
	Gender      wow.PlayerGender `json:"gender"`
	GenderName  string           `json:"genderName"`
	DisplayID   uint32           `json:"displayId"`
	ModelPath   string           `json:"modelPath"` // Path to base GLB
	SkinPrefix  string           `json:"skinPrefix"`
	FacePrefix  string           `json:"facePrefix"`
	HairPrefix  string           `json:"hairPrefix"`
	Scale       float32          `json:"scale"`
}

// ItemVisualInfo describes a rendered equipped item.
type ItemVisualInfo struct {
	Slot          int    `json:"slot"`
	SlotName      string `json:"slotName"`
	DisplayInfoID uint32 `json:"displayInfoId"`
	ItemEntry     uint32 `json:"itemEntry"`
	ModelPath     string `json:"modelPath,omitempty"`
	TexturePath   string `json:"texturePath,omitempty"`
	AttachBone    string `json:"attachBone,omitempty"` // e.g. "Bone_R_Hand", "Bone_Head"
}

// CharacterVisualManifest is the complete 3D asset descriptor for web rendering.
type CharacterVisualManifest struct {
	GUID              uint64            `json:"guid"`
	Name              string            `json:"name"`
	Race              wow.PlayerRace    `json:"race"`
	RaceName          string            `json:"raceName"`
	Class             wow.PlayerClass   `json:"class"`
	ClassName         string            `json:"className"`
	Gender            wow.PlayerGender  `json:"gender"`
	GenderName        string            `json:"genderName"`
	Level             uint8             `json:"level"`
	DisplayID         uint32            `json:"displayId"`
	ModelURL          string            `json:"modelUrl"`
	FallbackModelURL  string            `json:"fallbackModelUrl,omitempty"`
	SkinTextureURL    string            `json:"skinTextureUrl"`
	FaceTextureURL    string            `json:"faceTextureUrl"`
	HairTextureURL    string            `json:"hairTextureUrl"`
	HairGeoset        uint32            `json:"hairGeoset"`
	FacialHairGeoset  uint32            `json:"facialHairGeoset"`
	Equipment         []ItemVisualInfo  `json:"equipment"`
	Customization     CustomizationData `json:"customization"`
}

// CustomizationData holds raw customization indices.
type CustomizationData struct {
	Skin       uint8 `json:"skin"`
	Face       uint8 `json:"face"`
	HairStyle  uint8 `json:"hairStyle"`
	HairColor  uint8 `json:"hairColor"`
	FacialHair uint8 `json:"facialHair"`
}

// Resolver maps WoW character models and display IDs to client asset paths.
type Resolver struct {
	raceModels map[RaceGenderKey]RaceModelInfo
}

// NewResolver creates and initializes the default WoW 3.3.5a DBC display resolver.
func NewResolver() *Resolver {
	r := &Resolver{
		raceModels: make(map[RaceGenderKey]RaceModelInfo),
	}
	r.initRaceModels()
	return r
}

func (r *Resolver) initRaceModels() {
	defs := []RaceModelInfo{
		// Human
		{Race: wow.RaceHuman, RaceName: "Human", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 49,
			ModelPath: "Character/Human/Male/HumanMale.glb", SkinPrefix: "Character/Human/Male/HumanMaleSkin",
			FacePrefix: "Character/Human/Male/HumanMaleFace", HairPrefix: "Character/Human/Hair/HumanHair", Scale: 1.0},
		{Race: wow.RaceHuman, RaceName: "Human", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 50,
			ModelPath: "Character/Human/Female/HumanFemale.glb", SkinPrefix: "Character/Human/Female/HumanFemaleSkin",
			FacePrefix: "Character/Human/Female/HumanFemaleFace", HairPrefix: "Character/Human/Hair/HumanHair", Scale: 1.0},

		// Orc
		{Race: wow.RaceOrc, RaceName: "Orc", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 51,
			ModelPath: "Character/Orc/Male/OrcMale.glb", SkinPrefix: "Character/Orc/Male/OrcMaleSkin",
			FacePrefix: "Character/Orc/Male/OrcMaleFace", HairPrefix: "Character/Orc/Hair/OrcHair", Scale: 1.0},
		{Race: wow.RaceOrc, RaceName: "Orc", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 52,
			ModelPath: "Character/Orc/Female/OrcFemale.glb", SkinPrefix: "Character/Orc/Female/OrcFemaleSkin",
			FacePrefix: "Character/Orc/Female/OrcFemaleFace", HairPrefix: "Character/Orc/Hair/OrcHair", Scale: 1.0},

		// Dwarf
		{Race: wow.RaceDwarf, RaceName: "Dwarf", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 53,
			ModelPath: "Character/Dwarf/Male/DwarfMale.glb", SkinPrefix: "Character/Dwarf/Male/DwarfMaleSkin",
			FacePrefix: "Character/Dwarf/Male/DwarfMaleFace", HairPrefix: "Character/Dwarf/Hair/DwarfHair", Scale: 1.0},
		{Race: wow.RaceDwarf, RaceName: "Dwarf", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 54,
			ModelPath: "Character/Dwarf/Female/DwarfFemale.glb", SkinPrefix: "Character/Dwarf/Female/DwarfFemaleSkin",
			FacePrefix: "Character/Dwarf/Female/DwarfFemaleFace", HairPrefix: "Character/Dwarf/Hair/DwarfHair", Scale: 1.0},

		// NightElf
		{Race: wow.RaceNightElf, RaceName: "NightElf", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 55,
			ModelPath: "Character/NightElf/Male/NightElfMale.glb", SkinPrefix: "Character/NightElf/Male/NightElfMaleSkin",
			FacePrefix: "Character/NightElf/Male/NightElfMaleFace", HairPrefix: "Character/NightElf/Hair/NightElfHair", Scale: 1.0},
		{Race: wow.RaceNightElf, RaceName: "NightElf", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 56,
			ModelPath: "Character/NightElf/Female/NightElfFemale.glb", SkinPrefix: "Character/NightElf/Female/NightElfFemaleSkin",
			FacePrefix: "Character/NightElf/Female/NightElfFemaleFace", HairPrefix: "Character/NightElf/Hair/NightElfHair", Scale: 1.0},

		// Undead (Scourge)
		{Race: wow.RaceUndead, RaceName: "Undead", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 57,
			ModelPath: "Character/Scourge/Male/ScourgeMale.glb", SkinPrefix: "Character/Scourge/Male/ScourgeMaleSkin",
			FacePrefix: "Character/Scourge/Male/ScourgeMaleFace", HairPrefix: "Character/Scourge/Hair/ScourgeHair", Scale: 1.0},
		{Race: wow.RaceUndead, RaceName: "Undead", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 58,
			ModelPath: "Character/Scourge/Female/ScourgeFemale.glb", SkinPrefix: "Character/Scourge/Female/ScourgeFemaleSkin",
			FacePrefix: "Character/Scourge/Female/ScourgeFemaleFace", HairPrefix: "Character/Scourge/Hair/ScourgeHair", Scale: 1.0},

		// Tauren
		{Race: wow.RaceTauren, RaceName: "Tauren", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 59,
			ModelPath: "Character/Tauren/Male/TaurenMale.glb", SkinPrefix: "Character/Tauren/Male/TaurenMaleSkin",
			FacePrefix: "Character/Tauren/Male/TaurenMaleFace", HairPrefix: "Character/Tauren/Hair/TaurenHair", Scale: 1.25},
		{Race: wow.RaceTauren, RaceName: "Tauren", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 60,
			ModelPath: "Character/Tauren/Female/TaurenFemale.glb", SkinPrefix: "Character/Tauren/Female/TaurenFemaleSkin",
			FacePrefix: "Character/Tauren/Female/TaurenFemaleFace", HairPrefix: "Character/Tauren/Hair/TaurenHair", Scale: 1.2},

		// Gnome
		{Race: wow.RaceGnome, RaceName: "Gnome", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 1563,
			ModelPath: "Character/Gnome/Male/GnomeMale.glb", SkinPrefix: "Character/Gnome/Male/GnomeMaleSkin",
			FacePrefix: "Character/Gnome/Male/GnomeMaleFace", HairPrefix: "Character/Gnome/Hair/GnomeHair", Scale: 0.8},
		{Race: wow.RaceGnome, RaceName: "Gnome", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 1564,
			ModelPath: "Character/Gnome/Female/GnomeFemale.glb", SkinPrefix: "Character/Gnome/Female/GnomeFemaleSkin",
			FacePrefix: "Character/Gnome/Female/GnomeFemaleFace", HairPrefix: "Character/Gnome/Hair/GnomeHair", Scale: 0.8},

		// Troll
		{Race: wow.RaceTroll, RaceName: "Troll", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 1478,
			ModelPath: "Character/Troll/Male/TrollMale.glb", SkinPrefix: "Character/Troll/Male/TrollMaleSkin",
			FacePrefix: "Character/Troll/Male/TrollMaleFace", HairPrefix: "Character/Troll/Hair/TrollHair", Scale: 1.15},
		{Race: wow.RaceTroll, RaceName: "Troll", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 1479,
			ModelPath: "Character/Troll/Female/TrollFemale.glb", SkinPrefix: "Character/Troll/Female/TrollFemaleSkin",
			FacePrefix: "Character/Troll/Female/TrollFemaleFace", HairPrefix: "Character/Troll/Hair/TrollHair", Scale: 1.1},

		// BloodElf
		{Race: wow.RaceBloodElf, RaceName: "BloodElf", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 15476,
			ModelPath: "Character/BloodElf/Male/BloodElfMale.glb", SkinPrefix: "Character/BloodElf/Male/BloodElfMaleSkin",
			FacePrefix: "Character/BloodElf/Male/BloodElfMaleFace", HairPrefix: "Character/BloodElf/Hair/BloodElfHair", Scale: 1.0},
		{Race: wow.RaceBloodElf, RaceName: "BloodElf", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 15475,
			ModelPath: "Character/BloodElf/Female/BloodElfFemale.glb", SkinPrefix: "Character/BloodElf/Female/BloodElfFemaleSkin",
			FacePrefix: "Character/BloodElf/Female/BloodElfFemaleFace", HairPrefix: "Character/BloodElf/Hair/BloodElfHair", Scale: 1.0},

		// Draenei
		{Race: wow.RaceDraenei, RaceName: "Draenei", Gender: wow.GenderMale, GenderName: "Male", DisplayID: 16125,
			ModelPath: "Character/Draenei/Male/DraeneiMale.glb", SkinPrefix: "Character/Draenei/Male/DraeneiMaleSkin",
			FacePrefix: "Character/Draenei/Male/DraeneiMaleFace", HairPrefix: "Character/Draenei/Hair/DraeneiHair", Scale: 1.15},
		{Race: wow.RaceDraenei, RaceName: "Draenei", Gender: wow.GenderFemale, GenderName: "Female", DisplayID: 16126,
			ModelPath: "Character/Draenei/Female/DraeneiFemale.glb", SkinPrefix: "Character/Draenei/Female/DraeneiFemaleSkin",
			FacePrefix: "Character/Draenei/Female/DraeneiFemaleFace", HairPrefix: "Character/Draenei/Hair/DraeneiHair", Scale: 1.1},
	}

	for _, d := range defs {
		r.raceModels[RaceGenderKey{Race: d.Race, Gender: d.Gender}] = d
	}
}

// GetRaceModelInfo returns model info for a given race and gender.
func (r *Resolver) GetRaceModelInfo(race wow.PlayerRace, gender wow.PlayerGender) (RaceModelInfo, bool) {
	info, ok := r.raceModels[RaceGenderKey{Race: race, Gender: gender}]
	return info, ok
}

// GetAllRaces returns all configured race/gender model definitions.
func (r *Resolver) GetAllRaces() []RaceModelInfo {
	res := make([]RaceModelInfo, 0, len(r.raceModels))
	for _, v := range r.raceModels {
		res = append(res, v)
	}
	return res
}

// ClassName returns the human-readable class name.
func ClassName(c wow.PlayerClass) string {
	switch c {
	case wow.ClassWarior:
		return "Warrior"
	case wow.ClassPaladin:
		return "Paladin"
	case wow.ClassHunter:
		return "Hunter"
	case wow.ClassRogue:
		return "Rogue"
	case wow.ClassPriest:
		return "Priest"
	case wow.ClassDeathKnight:
		return "Death Knight"
	case wow.ClassShaman:
		return "Shaman"
	case wow.ClassMage:
		return "Mage"
	case wow.ClassWarlock:
		return "Warlock"
	case wow.ClassDruid:
		return "Druid"
	default:
		return fmt.Sprintf("Class(%d)", c)
	}
}

// ResolveCharacterVisuals generates a full visual manifest for a player character.
func (r *Resolver) ResolveCharacterVisuals(p *player.Player) CharacterVisualManifest {
	info, ok := r.GetRaceModelInfo(p.Race, p.Gender)
	if !ok {
		// Fallback default (Human Male)
		info = r.raceModels[RaceGenderKey{Race: wow.RaceHuman, Gender: wow.GenderMale}]
	}

	skinUrl := fmt.Sprintf("%s%02d_%02d.webp", info.SkinPrefix, 0, p.Skin)
	faceUrl := fmt.Sprintf("%s%02d_%02d.webp", info.FacePrefix, 0, p.Face)
	hairUrl := fmt.Sprintf("%s%02d_%02d.webp", info.HairPrefix, 0, p.HairColor)

	manifest := CharacterVisualManifest{
		GUID:             uint64(p.GUID()),
		Name:             p.Name,
		Race:             p.Race,
		RaceName:         info.RaceName,
		Class:            p.Class,
		ClassName:        ClassName(p.Class),
		Gender:           p.Gender,
		GenderName:       info.GenderName,
		Level:            p.Level,
		DisplayID:        info.DisplayID,
		ModelURL:         info.ModelPath,
		FallbackModelURL: "models/npc/4218362.glb", // Built-in sample model for testing
		SkinTextureURL:   skinUrl,
		FaceTextureURL:   faceUrl,
		HairTextureURL:   hairUrl,
		HairGeoset:       uint32(100 + p.HairStyle),
		FacialHairGeoset: uint32(100 + p.FacialHair),
		Customization: CustomizationData{
			Skin:       p.Skin,
			Face:       p.Face,
			HairStyle:  p.HairStyle,
			HairColor:  p.HairColor,
			FacialHair: p.FacialHair,
		},
		Equipment: r.resolveEquipment(p),
	}

	return manifest
}

func (r *Resolver) resolveEquipment(p *player.Player) []ItemVisualInfo {
	var equip []ItemVisualInfo
	slotNames := map[int]string{
		player.EquipmentSlotHead:      "Head",
		player.EquipmentSlotNeck:      "Neck",
		player.EquipmentSlotShoulder:  "Shoulder",
		player.EquipmentSlotBody:      "Body",
		player.EquipmentSlotChest:     "Chest",
		player.EquipmentSlotWaist:     "Waist",
		player.EquipmentSlotLegs:      "Legs",
		player.EquipmentSlotFeet:      "Feet",
		player.EquipmentSlotWrists:    "Wrists",
		player.EquipmentSlotHands:     "Hands",
		player.EquipmentSlotFinger1:   "Finger1",
		player.EquipmentSlotFinger2:   "Finger2",
		player.EquipmentSlotTrinket1:  "Trinket1",
		player.EquipmentSlotTrinket2:  "Trinket2",
		player.EquipmentSlotBack:      "Back",
		player.EquipmentSlotMainHand:  "MainHand",
		player.EquipmentSlotOffHand:   "OffHand",
		player.EquipmentSlotRanged:    "Ranged",
		player.EquipmentSlotTabard:    "Tabard",
	}

	// Characters loaded from persistence may carry no slot data at all
	for i := 0; i < player.EquipmentSlotEnd && i < len(p.Inventory.Slots); i++ {
		item := p.Inventory.Slots[i]
		if item == nil || item.ItemEntry == 0 {
			continue
		}

		sName := slotNames[i]
		bone := ""
		if i == player.EquipmentSlotMainHand {
			bone = "Bone_R_Hand"
		} else if i == player.EquipmentSlotOffHand {
			bone = "Bone_L_Hand"
		} else if i == player.EquipmentSlotHead {
			bone = "Bone_Head"
		}

		equip = append(equip, ItemVisualInfo{
			Slot:          i,
			SlotName:      sName,
			DisplayInfoID: uint32(item.ItemEntry), // Or display info lookup
			ItemEntry:     item.ItemEntry,
			AttachBone:    bone,
		})
	}

	return equip
}
