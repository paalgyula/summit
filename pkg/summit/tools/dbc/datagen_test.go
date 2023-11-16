//nolint:testpackage
package dbc_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	t.Skip("used for local development only")

	f, err := os.Open("../../../../dbc/ChrClasses.dbc")
	assert.NoError(t, err)

	dr, err := dbc.NewReader[wotlk.ChrClassesEntry](f)
	assert.NoError(t, err)
	assert.EqualValues(t, 0xa, dr.Header.RecordCount)

	dr.ReadAll()

	fmt.Println(dr.Records[1].Name.Value())

	t.Run("CharacterLoad", func(t *testing.T) {
		base := "../../../../dbc"

		t.Run("CharStartOutfit.dbc", func(t *testing.T) {
			data, err := dbc.Load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", base)
			assert.NoError(t, err)
			assert.Lenf(t, data, 126, "Expected 126 records in CharStartOutfit.dbc")
		})
	})

	// fmt.Printf("%+v\n\n", dr.Header)
}

func TestReadOutfitData(t *testing.T) {
	base := "../../../../dbc"

	data, err := dbc.Load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", base)
	assert.NoError(t, err)
	assert.Lenf(t, data, 126, "Expected 126 records in CharStartOutfit.dbc")

	// for i, csoe := range data {
	// 	fmt.Printf("%04d. %+v\n", i, csoe)
	// }
}
