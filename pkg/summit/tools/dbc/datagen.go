package dbc

import (
	"fmt"
	"os"
	"path"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
)

func LoadAll(dbcDirectoryPath string) {
	_, _ = load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", dbcDirectoryPath)
	_, _ = load[wotlk.MapEntry]("Map.dbc", dbcDirectoryPath)
}

func load[C any](fileName string, baseDir ...string) ([]C, error) {
	f, err := os.Open(path.Join(append(baseDir, fileName)...))
	if err != nil {
		panic(err)
	}

	r, err := NewReader[C](f)
	if err != nil {
		return nil, err
	}

	if err := r.ReadAll(); err != nil {
		return nil, err
	}

	fmt.Printf("DBC loader: Loaded %d records from %s\n", r.Header.RecordCount, fileName)

	return r.Records, nil
}
