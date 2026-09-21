package dbc_test

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/stretchr/testify/assert"
)

const dbcBasePath = "../../../../dbc"

func TestGenerate(t *testing.T) {
	t.Skip("used for local development only")

	f, err := os.Open(path.Join(dbcBasePath, "ChrClasses.dbc"))
	assert.NoError(t, err)

	dr, err := dbc.NewReader[wotlk.ChrClassesEntry](f)
	assert.NoError(t, err)
	assert.EqualValues(t, 0xa, dr.Header.RecordCount)

	dr.ReadAll()

	fmt.Println(dr.Records[1].Name.Value())

	t.Run("CharacterLoad", func(t *testing.T) {
		t.Run("CharStartOutfit.dbc", func(t *testing.T) {
			data, err := dbc.Load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", dbcBasePath)
			assert.NoError(t, err)
			assert.Lenf(t, data, 126, "Expected 126 records in CharStartOutfit.dbc")
		})
	})

	// fmt.Printf("%+v\n\n", dr.Header)
}

func TestReadOutfitData(t *testing.T) {
	data, err := dbc.Load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", dbcBasePath)
	assert.NoError(t, err)
	assert.Lenf(t, data, 126, "Expected 126 records in CharStartOutfit.dbc")

	hw := data[0] // Human warrior
	t.Logf("HW: Race=%d Class=%d Gender=%d", hw.RaceID, hw.ClassID, hw.Gender)
	for i := 0; i < len(hw.ItemID); i++ {
		if hw.ItemID[i] > 0 || hw.DisplayItemID[i] > 0 {
			t.Logf("  Slot %d: ItemID=%d, DisplayItemID=%d, InventoryType=%d",
				i, hw.ItemID[i], hw.DisplayItemID[i], hw.InventoryType[i])
		}
	}
}

func TestExportAllStartOutfits(t *testing.T) {
	data, err := dbc.Load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", dbcBasePath)
	assert.NoError(t, err)
	assert.Lenf(t, data, 126, "Expected 126 records in CharStartOutfit.dbc")

	type ItemSlot struct {
		Slot          int   `json:"slot"`
		DisplayID     int32 `json:"displayId"`
		InventoryType int32 `json:"inventoryType"`
	}

	slotForType := func(invType int32, hasMainHand bool) int {
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

	outfits := map[string][]ItemSlot{}
	for _, entry := range data {
		key := fmt.Sprintf("%d_%d_%d", entry.RaceID, entry.ClassID, entry.Gender)
		hasMainHand := false
		var items []ItemSlot
		for i := 0; i < len(entry.DisplayItemID); i++ {
			dispID := entry.DisplayItemID[i]
			invType := entry.InventoryType[i]
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
			items = append(items, ItemSlot{
				Slot:          slot,
				DisplayID:     dispID,
				InventoryType: invType,
			})
		}
		outfits[key] = items
	}

	t.Logf("Total outfits mapped: %d", len(outfits))

	// Generate client/src/entities/CharStartOutfitData.ts
	var sb strings.Builder
	sb.WriteString("/** Generated from CharStartOutfit.dbc - do not edit directly. */\n\n")
	sb.WriteString("export interface StartOutfitItem {\n")
	sb.WriteString("  slot: number;\n")
	sb.WriteString("  displayId: number;\n")
	sb.WriteString("  inventoryType: number;\n")
	sb.WriteString("}\n\n")
	sb.WriteString("export const CHAR_START_OUTFITS: Record<string, StartOutfitItem[]> = {\n")

	keys := make([]string, 0, len(outfits))
	for k := range outfits {
		keys = append(keys, k)
	}
	for _, k := range keys {
		items := outfits[k]
		sb.WriteString(fmt.Sprintf("  %q: [\n", k))
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("    { slot: %d, displayId: %d, inventoryType: %d },\n",
				item.Slot, item.DisplayID, item.InventoryType))
		}
		sb.WriteString("  ],\n")
	}
	sb.WriteString("};\n")

	outPath := "../../../../client/src/entities/CharStartOutfitData.ts"
	err = os.WriteFile(outPath, []byte(sb.String()), 0644)
	assert.NoError(t, err)
	t.Logf("Wrote %d bytes to %s", sb.Len(), outPath)
}
