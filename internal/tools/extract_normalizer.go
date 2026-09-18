//go:build !windows

//nolint:all
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RenameToLowerCase(path string) error {
	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip renaming the root directory
		if file == path {
			return nil
		}

		dir, filename := filepath.Split(file)

		dir = strings.ToLower(dir)
		_ = os.MkdirAll(dir, 0o755)

		newFilename := strings.ToLower(filename)
		newPath := filepath.Join(dir, newFilename)

		err = os.Rename(file, newPath)
		if err != nil {
			return err
		}

		fmt.Printf("Renamed: %s to %s\n", file, newPath)

		return nil
	})
	if err != nil {
		RenameToLowerCase(path)
	}

	return err
}

func main() {
	extractDir := flag.String("dir", ".", "extracted WoW assets directory")

	flag.Parse()

	if extractDir == nil {
		fmt.Println("dir flag is required.")

		os.Exit(1)
	}

	err := RenameToLowerCase(*extractDir)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
}
