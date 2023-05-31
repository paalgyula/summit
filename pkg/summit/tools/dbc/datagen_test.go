package dbc

import (
	"fmt"
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	// LoadCharacterRaces()
	// LoadClassInfo()

	f, err := os.Open("../../../../dbc/ChrClasses.dbc")
	assert.NoError(t, err)

	dr, err := NewReader[wotlk.ChrClassesEntry](f)
	assert.NoError(t, err)
	assert.EqualValues(t, 0xa, dr.Header.RecordCount)

	dr.ReadAll()

	fmt.Println(dr.Records[1].Name.Value())

	// fmt.Printf("%+v\n\n", dr.Header)
}

func TestLoadMap(t *testing.T) {
	LoadMaps("../../../../dbc")
}

func TestLoadCharacters(t *testing.T) {
	_, err := LoadCharacter("../../../../dbc")
	assert.NoError(t, err)
}
