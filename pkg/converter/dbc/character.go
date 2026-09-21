package dbc

import (
	"fmt"
	"path"
	"strings"
)

// Section kinds of CharSections.dbc.
const (
	SectionSkin       = 0
	SectionFace       = 1
	SectionFacialHair = 2
	SectionHair       = 3
	SectionUnderwear  = 4
)

// Race is a playable race of ChrRaces.dbc.
type Race struct {
	Name string `json:"name"`
	// Prefix is the client's two-letter race code used in helmet model names
	// (Helm_..._HuM): Hu, Or, Dw, Ni, Sc, Ta, Gn, Tr, Be, Dr.
	Prefix string `json:"prefix"`
}

// Section is one CharSections.dbc row: the textures of a customization
// choice (skin colour, face, facial hair, hair style / colour, underwear).
type Section struct {
	Race    int `json:"race"`
	Gender  int `json:"gender"`
	Section int `json:"section"`
	// Textures are asset-server WebP paths ("" when unused): skin = body [,
	// extra]; face / facial hair = lower, upper; hair = hair, scalp lower,
	// scalp upper; underwear = pelvis, torso.
	Textures  [3]string `json:"textures"`
	Flags     uint32    `json:"flags"`
	Variation int       `json:"variation"`
	Color     int       `json:"color"`
}

// HairGeoset is one CharHairGeosets.dbc row: the hair mesh of a style.
type HairGeoset struct {
	Race      int  `json:"race"`
	Gender    int  `json:"gender"`
	Variation int  `json:"variation"`
	Geoset    int  `json:"geoset"`
	ShowScalp bool `json:"showScalp,omitempty"`
}

// FacialHair is one CharacterFacialHairStyles.dbc row.
type FacialHair struct {
	Race      int `json:"race"`
	Gender    int `json:"gender"`
	Variation int `json:"variation"`
	// Geosets are the variants of geoset groups 1, 2 and 3 (100, 200, 300).
	Geosets [3]int `json:"geosets"`
}

// ItemDisplay is one ItemDisplayInfo.dbc row with the data needed to draw
// the item on a character.
type ItemDisplay struct {
	// Models are the left / right model names without extension (weapons and
	// shields use the first; shoulders one per side; helmets add _<Race><Sex>).
	// Trailing empty entries are dropped from every list below.
	Models []string `json:"models,omitempty"`
	// Textures are the models' object-skin textures (and the cape's texture).
	Textures []string `json:"textures,omitempty"`
	// Geosets are the item's three geoset group variants.
	Geosets [3]int `json:"geosets"`
	Flags   uint32 `json:"flags,omitempty"`
	// HelmetGeosets index HelmetGeosetVisData (male, female).
	HelmetGeosets []int `json:"helmetGeosets,omitempty"`
	// Components are the body-texture components: arm upper, arm lower,
	// hand, torso upper, torso lower, leg upper, leg lower, foot.
	Components []string `json:"components,omitempty"`
	// Sheathe is the item's SheatheType (Item.dbc), 0 when unknown.
	Sheathe int `json:"sheathe,omitempty"`
}

// StartOutfitItem is an item equipped on a character starting outfit.
type StartOutfitItem struct {
	Slot          int   `json:"slot"`
	DisplayID     int32 `json:"displayId"`
	InventoryType int32 `json:"inventoryType"`
}

// CharacterData is everything the web client needs to dress a character:
// the customization tables and the item display table.
type CharacterData struct {
	Races         map[int]Race                 `json:"races"`
	Sections      []Section                    `json:"sections"`
	HairGeosets   []HairGeoset                 `json:"hairGeosets"`
	FacialHair    []FacialHair                 `json:"facialHair"`
	// HelmetGeosets maps a HelmetGeosetVisData id to its seven race masks:
	// hair, facial 1-3, ears and two more groups hidden for race bit (race-1).
	HelmetGeosets map[int][7]uint32            `json:"helmetGeosets"`
	ItemDisplay   map[int]ItemDisplay          `json:"itemDisplay"`
	Outfits       map[string][]StartOutfitItem `json:"outfits,omitempty"`
}

// Tables are the DBC files CharacterData is built from, by file name; any of
// them may be nil.
type Tables struct {
	ChrRaces            *File
	CharSections        *File
	CharHairGeosets     *File
	FacialHairStyles    *File // CharacterFacialHairStyles.dbc
	HelmetGeosetVisData *File
	ItemDisplayInfo     *File
	Item                *File
	CharStartOutfit     *File
}

// CharacterTableNames lists the DBFilesClient files of Tables in field order.
var CharacterTableNames = []string{
	"ChrRaces.dbc", "CharSections.dbc", "CharHairGeosets.dbc", "CharacterFacialHairStyles.dbc",
	"HelmetGeosetVisData.dbc", "ItemDisplayInfo.dbc", "Item.dbc", "CharStartOutfit.dbc",
}

// BuildCharacterData assembles the client's character data from the tables.
func BuildCharacterData(t Tables) *CharacterData {
	out := &CharacterData{
		Races:         map[int]Race{},
		HelmetGeosets: map[int][7]uint32{},
		ItemDisplay:   map[int]ItemDisplay{},
	}

	if f := t.ChrRaces; f != nil {
		// ChrRaces: 0 id, 6 client prefix, 14 name (enUS)
		for r := 0; r < f.Records; r++ {
			out.Races[int(f.Uint32(r, 0))] = Race{Name: f.String(r, 14), Prefix: f.String(r, 6)}
		}
	}

	if f := t.CharSections; f != nil {
		// CharSections: 0 id, 1 race, 2 sex, 3 section, 4-6 textures, 7 flags, 8 variation, 9 colour
		for r := 0; r < f.Records; r++ {
			s := Section{
				Race:      int(f.Uint32(r, 1)),
				Gender:    int(f.Uint32(r, 2)),
				Section:   int(f.Uint32(r, 3)),
				Flags:     f.Uint32(r, 7),
				Variation: int(f.Uint32(r, 8)),
				Color:     int(f.Uint32(r, 9)),
			}
			for i := 0; i < 3; i++ {
				s.Textures[i] = TexturePath(f.String(r, 4+i))
			}
			out.Sections = append(out.Sections, s)
		}
	}

	if f := t.CharHairGeosets; f != nil {
		// CharHairGeosets: 0 id, 1 race, 2 sex, 3 variation, 4 geoset, 5 show scalp
		for r := 0; r < f.Records; r++ {
			out.HairGeosets = append(out.HairGeosets, HairGeoset{
				Race:      int(f.Uint32(r, 1)),
				Gender:    int(f.Uint32(r, 2)),
				Variation: int(f.Uint32(r, 3)),
				Geoset:    int(f.Uint32(r, 4)),
				ShowScalp: f.Uint32(r, 5) != 0,
			})
		}
	}

	if f := t.FacialHairStyles; f != nil {
		// CharacterFacialHairStyles: 0 race, 1 sex, 2 variation, 3.. geosets
		// of groups 100, 300, 200 (then 1600, 1700 in the 8-column layout)
		for r := 0; r < f.Records; r++ {
			out.FacialHair = append(out.FacialHair, FacialHair{
				Race:      int(f.Uint32(r, 0)),
				Gender:    int(f.Uint32(r, 1)),
				Variation: int(f.Uint32(r, 2)),
				Geosets:   [3]int{int(f.Uint32(r, 3)), int(f.Uint32(r, 5)), int(f.Uint32(r, 4))},
			})
		}
	}

	if f := t.HelmetGeosetVisData; f != nil {
		for r := 0; r < f.Records; r++ {
			var masks [7]uint32
			for i := range masks {
				masks[i] = f.Uint32(r, 1+i)
			}
			out.HelmetGeosets[int(f.Uint32(r, 0))] = masks
		}
	}

	sheathe := map[int]int{}
	if f := t.Item; f != nil {
		// Item.dbc: 8 columns (id, class, subclass, sound, material, display,
		// inventory type, sheathe) or the 4-column dump (id, display, type, sheathe)
		displayCol, sheatheCol := 5, 7
		if f.Fields == 4 {
			displayCol, sheatheCol = 1, 3
		}
		for r := 0; r < f.Records; r++ {
			display, st := int(f.Uint32(r, displayCol)), int(f.Uint32(r, sheatheCol))
			if display == 0 || st == 0 {
				continue
			}
			if _, ok := sheathe[display]; !ok {
				sheathe[display] = st
			}
		}
	}

	if f := t.ItemDisplayInfo; f != nil {
		// ItemDisplayInfo: 0 id, 1-2 models, 3-4 model textures, 5 icon, 6 ground
		// model, 7-9 geoset groups, 10 flags, 11 spell visual, 12 sound, 13-14
		// helmet geoset vis, 15-22 texture components, 23 item visual, 24 particles
		for r := 0; r < f.Records; r++ {
			d := ItemDisplay{
				Models:        trimStrings([]string{ModelName(f.String(r, 1)), ModelName(f.String(r, 2))}),
				Textures:      trimStrings([]string{f.String(r, 3), f.String(r, 4)}),
				Geosets:       [3]int{int(f.Uint32(r, 7)), int(f.Uint32(r, 8)), int(f.Uint32(r, 9))},
				Flags:         f.Uint32(r, 10),
				HelmetGeosets: trimInts([]int{int(f.Uint32(r, 13)), int(f.Uint32(r, 14))}),
			}
			components := make([]string, 8)
			for i := range components {
				components[i] = f.String(r, 15+i)
			}
			d.Components = trimStrings(components)
			if len(d.Models) == 0 && len(d.Textures) == 0 && len(d.Components) == 0 && d.Geosets == [3]int{} {
				continue // icon-only items (trade goods, quest items, ...)
			}
			id := int(f.Uint32(r, 0))
			d.Sheathe = sheathe[id]
			out.ItemDisplay[id] = d
		}
	}

	if f := t.CharStartOutfit; f != nil {
		// CharStartOutfit: 0 id, 1 packed (race, class, gender, outfit),
		// 2..25 itemID (24 items), 26..49 displayItemID (24 items), 50..73 inventoryType (24 items)
		out.Outfits = make(map[string][]StartOutfitItem)
		for r := 0; r < f.Records; r++ {
			v := f.Uint32(r, 1)
			race := int(v & 0xff)
			class := int((v >> 8) & 0xff)
			gender := int((v >> 16) & 0xff)
			key := fmt.Sprintf("%d_%d_%d", race, class, gender)

			hasMainHand := false
			var items []StartOutfitItem
			for i := 0; i < 24; i++ {
				dispID := int32(f.Uint32(r, 26+i))
				invType := int32(f.Uint32(r, 50+i))
				if dispID <= 0 || invType <= 0 {
					continue
				}
				slot := slotForType(invType, hasMainHand)
				if slot < 0 {
					continue
				}
				if slot == 15 {
					hasMainHand = true
				}
				items = append(items, StartOutfitItem{
					Slot:          slot,
					DisplayID:     dispID,
					InventoryType: invType,
				})
			}
			out.Outfits[key] = items
		}
	}

	return out
}

func slotForType(invType int32, hasMainHand bool) int {
	switch invType {
	case 1: // Head
		return 0
	case 3: // Shoulders
		return 2
	case 4: // Shirt
		return 3
	case 5, 20: // Chest, Robe
		return 4
	case 6: // Waist
		return 5
	case 7: // Legs
		return 6
	case 8: // Feet
		return 7
	case 9: // Wrists
		return 8
	case 10: // Hands
		return 9
	case 16: // Cloak
		return 14
	case 13: // 1H Weapon
		if hasMainHand {
			return 16 // Off hand dual wield
		}
		return 15 // Main hand
	case 17, 21: // 2H Weapon, Main Hand
		return 15
	case 14, 22, 23: // Shield, Off Hand, Holdable
		return 16
	case 15, 25, 26: // Ranged, Thrown, Ranged Right
		return 17
	case 19: // Tabard
		return 18
	default:
		return -1
	}
}

// trimStrings drops the trailing empty entries of a list (nil when all are).
func trimStrings(list []string) []string {
	n := len(list)
	for n > 0 && list[n-1] == "" {
		n--
	}
	if n == 0 {
		return nil
	}
	return list[:n]
}

// trimInts drops the trailing zeros of a list (nil when all are).
func trimInts(list []int) []int {
	n := len(list)
	for n > 0 && list[n-1] == 0 {
		n--
	}
	if n == 0 {
		return nil
	}
	return list[:n]
}

// TexturePath maps a client BLP path to the asset server's WebP path.
func TexturePath(blp string) string {
	if blp == "" {
		return ""
	}
	p := strings.ReplaceAll(blp, "\\", "/")
	if strings.HasSuffix(strings.ToLower(p), ".blp") {
		p = p[:len(p)-4]
	}
	return p + ".webp"
}

// ModelName strips the .mdx / .m2 extension of an ItemDisplayInfo model name.
func ModelName(name string) string {
	if name == "" {
		return ""
	}
	ext := strings.ToLower(path.Ext(name))
	if ext == ".mdx" || ext == ".m2" || ext == ".mdl" {
		return name[:len(name)-len(ext)]
	}
	return name
}
