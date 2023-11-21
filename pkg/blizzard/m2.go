package blizzard

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

const (
	VersionClassic     = 256
	VersionBC          = 263
	VersionWOTLK       = 264
	VersionCata        = 272
	VersionPanda       = 272
	VersionWOD         = 272
	VersionLegion      = 274
	VersionBFA         = 274
	VersionShadowlands = 274
)

type M2 struct {
	Name string
}

type M2Header struct {
	Version                            uint32
	NameLength                         uint32
	NameOffset                         uint32
	GlobalFlags                        uint32
	GlobalLoopsLength                  uint32
	GlobalLoopsOffset                  uint32
	SequencesLength                    uint32
	SequencesOffset                    uint32
	SequenceIdxHashByIDLength          uint32
	SequenceIdxHashByOffset            uint32
	PlayableAnimationLookupLength      uint32 // only if header.Version <= M2_VERSION_THE_BURNING_CRUSADE
	PlayableAnimationLookupOffset      uint32 // only if header.Version <= M2_VERSION_THE_BURNING_CRUSADE
	BonesLength                        uint32
	BonesOffset                        uint32
	BoneIndicesByIDLength              uint32
	BoneIndicesByIDOffset              uint32
	VerticesLength                     uint32
	VerticesOffset                     uint32
	SkinProfilesLength                 uint32 // only if header.Version <= M2_VERSION_THE_BURNING_CRUSADE
	SkinProfilesOffset                 uint32 // only if header.Version <= M2_VERSION_THE_BURNING_CRUSADE
	NumSkinProfiles                    uint32 // only if header.Version > M2_VERSION_THE_BURNING_CRUSADE
	ColorsLength                       uint32
	ColorsOffset                       uint32
	TexturesLength                     uint32
	TexturesOffset                     uint32
	TextureWeightsLength               uint32
	TextureWeightsOffset               uint32
	TextureFlipbooksLength             uint32 // only if header.Version <= M2_VERSION_THE_BURNING_CRUSADE
	TextureFlipbooksOffset             uint32 // only if header.Version <= M2_VERSION_THE_BURNING_CRUSADE
	TextureTransformsLength            uint32
	TextureTransformsOffset            uint32
	TextureIndicesByIDLength           uint32
	TextureIndicesByIDOffset           uint32
	MaterialsLength                    uint32
	MaterialsOffset                    uint32
	BoneLookupTableLength              uint32
	BoneLookupTableOffset              uint32
	TextureLookupTableLength           uint32
	TextureLookupTableOffset           uint32
	TextureUnitLookupTableLength       uint32
	TextureUnitLookupTableOffset       uint32
	TransparencyLookupTableLength      uint32
	TransparencyLookupTableOffset      uint32
	TextureTransformsLookupTableLength uint32
	TextureTransformsLookupTableOffset uint32
}

type M2Reader struct {
	header *M2Header

	// Is the M2 file chunked
	chunked bool

	r io.ReadSeeker

	readerPosition int64
}

var ErrInvalidM2Header = errors.New("invalid M2 header")

func NewM2Reader(r *os.File) (*M2Reader, error) {
	reader := &M2Reader{
		header:  new(M2Header),
		chunked: false,
		r:       r,

		readerPosition: 0,
	}

	magic := make([]byte, 4)
	if _, err := r.Read(magic); err != nil {
		return nil, fmt.Errorf("error while reading magic: %w", err)
	}

	fh := string(magic)

	switch fh {
	case "MD20":
		// Set some variables
		reader.chunked = false
	case "MD21":
		reader.chunked = true

		panic("chunked M2 file is not implemented")
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidM2Header, fh)
	}

	reader.readHeader()

	reader.readData()

	return reader, nil
}

// Refactored function to take a parameter to readValue value into.
func (mr *M2Reader) readValue(out any) {
	if err := binary.Read(mr.r, binary.LittleEndian, out); err != nil {
		log.Fatal().Err(err).Caller().Msgf("cannot read %T from reader", out)
	}
}

func (mr *M2Reader) readData() error {
	// This should not result an error.
	mr.readerPosition, _ = mr.r.Seek(0, io.SeekCurrent)

	_, err := mr.r.Seek(int64(mr.header.NameOffset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("cannot seek to name offset: %w", err)
	}

	model := new(M2)

	name := make([]byte, mr.header.NameLength)
	_, _ = mr.r.Read(name)

	model.Name = string(name)

	return nil
}

// readHeader reads the header from the M2 file.
//
//nolint:funlen
func (mr *M2Reader) readHeader() {
	mr.readValue(&mr.header.Version)
	mr.readValue(&mr.header.NameLength)
	mr.readValue(&mr.header.NameOffset)
	mr.readValue(&mr.header.GlobalFlags)
	mr.readValue(&mr.header.GlobalLoopsLength)
	mr.readValue(&mr.header.GlobalLoopsOffset)
	mr.readValue(&mr.header.SequencesLength)
	mr.readValue(&mr.header.SequencesOffset)
	mr.readValue(&mr.header.SequenceIdxHashByIDLength)
	mr.readValue(&mr.header.SequenceIdxHashByOffset)

	if mr.header.Version <= VersionBC {
		mr.readValue(&mr.header.PlayableAnimationLookupLength)
		mr.readValue(&mr.header.PlayableAnimationLookupOffset)
	}

	mr.readValue(&mr.header.BonesLength)
	mr.readValue(&mr.header.BonesOffset)
	mr.readValue(&mr.header.BoneIndicesByIDLength)
	mr.readValue(&mr.header.BoneIndicesByIDOffset)
	mr.readValue(&mr.header.VerticesLength)
	mr.readValue(&mr.header.VerticesOffset)

	if mr.header.Version <= VersionBC {
		mr.readValue(&mr.header.SkinProfilesLength)
		mr.readValue(&mr.header.SkinProfilesOffset)
	} else {
		mr.readValue(&mr.header.NumSkinProfiles)
	}

	mr.readValue(&mr.header.ColorsLength)
	mr.readValue(&mr.header.ColorsOffset)
	mr.readValue(&mr.header.TexturesLength)
	mr.readValue(&mr.header.TexturesOffset)
	mr.readValue(&mr.header.TextureWeightsLength)
	mr.readValue(&mr.header.TextureWeightsOffset)

	if mr.header.Version <= VersionBC {
		mr.readValue(&mr.header.TextureFlipbooksLength)
		mr.readValue(&mr.header.TextureFlipbooksOffset)
	}

	mr.readValue(&mr.header.TextureTransformsLength)
	mr.readValue(&mr.header.TextureTransformsOffset)
	mr.readValue(&mr.header.TextureIndicesByIDLength)
	mr.readValue(&mr.header.TextureIndicesByIDOffset)
	mr.readValue(&mr.header.MaterialsLength)
	mr.readValue(&mr.header.MaterialsOffset)
	mr.readValue(&mr.header.BoneLookupTableLength)
	mr.readValue(&mr.header.BoneLookupTableOffset)
	mr.readValue(&mr.header.TextureLookupTableLength)
	mr.readValue(&mr.header.TextureLookupTableOffset)
	mr.readValue(&mr.header.TextureUnitLookupTableLength)
	mr.readValue(&mr.header.TextureUnitLookupTableOffset)
	mr.readValue(&mr.header.TransparencyLookupTableLength)
	mr.readValue(&mr.header.TransparencyLookupTableOffset)
	mr.readValue(&mr.header.TextureTransformsLookupTableLength)
	mr.readValue(&mr.header.TextureTransformsLookupTableOffset)

	fmt.Println(mr.header.ToString())
}

// ToString generates a string representation of the M2Header with all fields.
func (h *M2Header) ToString() string {
	return fmt.Sprintf("%+v", h)
}
