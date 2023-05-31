package dbc

import (
	"fmt"
	"os"
	"path"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
)

func LoadMaps(dbcDirectoryPath string) ([]wotlk.MapEntry, error) {
	filename := "Map.dbc"
	f, err := os.Open(path.Join(filename))
	if err != nil {
		panic(err)
	}

	r, err := NewReader[wotlk.MapEntry](f)
	if err != nil {
		return nil, err
	}
	if err := r.ReadAll(); err != nil {
		return nil, err
	}

	fmt.Println(r.Records[0].MapName.Value())

	// fmt.Println(string(r.Strings()))
	return r.Records, nil
}

func LoadCharacter(dbcDirectoryPath string) ([]wotlk.CharStartOutfitEntry, error) {
	filename := "CharStartOutfit.dbc"
	f, err := os.Open(path.Join(dbcDirectoryPath, filename))
	if err != nil {
		return nil, fmt.Errorf("cannot load CharStartOutfit.dbc: %w", err)
	}

	r, err := NewReader[wotlk.CharStartOutfitEntry](f)
	if err != nil {
		return nil, err
	}
	if err := r.ReadAll(); err != nil {
		return nil, err
	}

	fmt.Println("Loaded", len(r.Records), "records")

	return r.Records, nil
}
