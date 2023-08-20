//nolint:testpackage
package dbc

import (
	"fmt"
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	t.Skip("used for local development only")

	f, err := os.Open("../../../../dbc/ChrClasses.dbc")
	assert.NoError(t, err)

	dr, err := NewReader[wotlk.ChrClassesEntry](f)
	assert.NoError(t, err)
	assert.EqualValues(t, 0xa, dr.Header.RecordCount)

	dr.ReadAll()

	fmt.Println(dr.Records[1].Name.Value())

	t.Run("CharacterLoad", func(t *testing.T) {
		base := "../../../../dbc"

		t.Run("CharStartOutfit.dbc", func(t *testing.T) {
			data, err := load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", base)
			assert.NoError(t, err)
			assert.Lenf(t, data, 126, "Expected 126 records in CharStartOutfit.dbc")
		})
	})

	// fmt.Printf("%+v\n\n", dr.Header)
}
