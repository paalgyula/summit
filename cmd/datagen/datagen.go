package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/paalgyula/summit/pkg/wow/dbc"
)

// const exportPath = "export"
const githubBase = "https://raw.githubusercontent.com/Torrer/TrinityCore-3.3.5-data/master/dbc/"

// var files = []string{"ChrClasses.dbc"}

const racesFormat = `uxxxxxxuxxxxuxlxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

type ChrRacesRec struct {
	RaceID              uint32
	Flags               uint32
	FactionID           uint32
	MaleDisplayId       uint32
	FemaleDisplayId     uint32
	ClientPrefix        uint32
	MountScale          float32
	BaseLanguage        uint32
	CreatureType        uint32
	LoginEffectSpellID  uint32
	CombatStunSpellID   uint32
	ResSicknessSpellID  uint32
	SplashSoundID       uint32
	StartingTaxiNodes   uint32
	ClientFileString    uint32
	CinematicSequenceID uint32
	NameLang            uint32
}

func downloadFile(fileName string) error {
	fullPath := path.Join("dbc", fileName)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil
	}

	out, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(fmt.Sprintf("%s/%s", githubBase, fileName))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	file := "ChrRaces.dbc"
	downloadFile(file)

	f, err := os.Open(path.Join("dbc", file))
	if err != nil {
		panic(err)
	}

	r := dbc.NewReader[ChrRacesRec](f, "nxixiixxixxxxissssssssssssssssxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxi")

	for r.HasNext() {
		rec := r.Next()

		fmt.Printf("%+v\n\n", rec) //  len(value)-int(pos)
	}

	fmt.Println(string(r.Strings()))
}
