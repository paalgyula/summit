package blizzard

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	Name   string

	chunked bool

	r io.ReadSeeker
}

const magicSize = 4

var (
	ErrInvalidM2Header         = errors.New("invalid M2 header")
	ErrChunkedM2NotImplemented = errors.New("chunked M2 file is not implemented")
)

func NewM2Reader(r io.ReadSeeker) (*M2Reader, error) {
	reader := &M2Reader{
		header:  new(M2Header),
		chunked: false,
		Name:    "",
		r:       r,
	}

	magic := make([]byte, magicSize)
	if _, err := r.Read(magic); err != nil {
		return nil, fmt.Errorf("error while reading magic: %w", err)
	}

	switch fh := string(magic); fh {
	case "MD20":
	case "MD21":
		reader.chunked = true

		return nil, ErrChunkedM2NotImplemented
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidM2Header, fh)
	}

	if err := reader.readHeader(); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	if err := reader.readData(); err != nil {
		return nil, fmt.Errorf("reading data: %w", err)
	}

	return reader, nil
}

func (mr *M2Reader) readValue(out any) error {
	if err := binary.Read(mr.r, binary.LittleEndian, out); err != nil {
		return fmt.Errorf("reading %T: %w", out, err)
	}

	return nil
}

func (mr *M2Reader) readData() error {
	pos, err := mr.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("saving current position: %w", err)
	}

	if _, err := mr.r.Seek(int64(mr.header.NameOffset), io.SeekStart); err != nil {
		return fmt.Errorf("cannot seek to name offset: %w", err)
	}

	name := make([]byte, mr.header.NameLength)
	if _, err := io.ReadFull(mr.r, name); err != nil {
		return fmt.Errorf("reading model name: %w", err)
	}

	mr.Name = string(name)

	if _, err := mr.r.Seek(pos, io.SeekStart); err != nil {
		return fmt.Errorf("restoring position: %w", err)
	}

	return nil
}

//nolint:funlen,gocognit,cyclop
func (mr *M2Reader) readHeader() error {
	fields := []any{
		&mr.header.Version,
		&mr.header.NameLength,
		&mr.header.NameOffset,
		&mr.header.GlobalFlags,
		&mr.header.GlobalLoopsLength,
		&mr.header.GlobalLoopsOffset,
		&mr.header.SequencesLength,
		&mr.header.SequencesOffset,
		&mr.header.SequenceIdxHashByIDLength,
		&mr.header.SequenceIdxHashByOffset,
	}

	for _, f := range fields {
		if err := mr.readValue(f); err != nil {
			return err
		}
	}

	if mr.header.Version <= VersionBC {
		for _, f := range []any{
			&mr.header.PlayableAnimationLookupLength,
			&mr.header.PlayableAnimationLookupOffset,
		} {
			if err := mr.readValue(f); err != nil {
				return err
			}
		}
	}

	for _, f := range []any{
		&mr.header.BonesLength,
		&mr.header.BonesOffset,
		&mr.header.BoneIndicesByIDLength,
		&mr.header.BoneIndicesByIDOffset,
		&mr.header.VerticesLength,
		&mr.header.VerticesOffset,
	} {
		if err := mr.readValue(f); err != nil {
			return err
		}
	}

	if mr.header.Version <= VersionBC {
		for _, f := range []any{
			&mr.header.SkinProfilesLength,
			&mr.header.SkinProfilesOffset,
		} {
			if err := mr.readValue(f); err != nil {
				return err
			}
		}
	} else {
		if err := mr.readValue(&mr.header.NumSkinProfiles); err != nil {
			return err
		}
	}

	for _, f := range []any{
		&mr.header.ColorsLength,
		&mr.header.ColorsOffset,
		&mr.header.TexturesLength,
		&mr.header.TexturesOffset,
		&mr.header.TextureWeightsLength,
		&mr.header.TextureWeightsOffset,
	} {
		if err := mr.readValue(f); err != nil {
			return err
		}
	}

	if mr.header.Version <= VersionBC {
		for _, f := range []any{
			&mr.header.TextureFlipbooksLength,
			&mr.header.TextureFlipbooksOffset,
		} {
			if err := mr.readValue(f); err != nil {
				return err
			}
		}
	}

	for _, f := range []any{
		&mr.header.TextureTransformsLength,
		&mr.header.TextureTransformsOffset,
		&mr.header.TextureIndicesByIDLength,
		&mr.header.TextureIndicesByIDOffset,
		&mr.header.MaterialsLength,
		&mr.header.MaterialsOffset,
		&mr.header.BoneLookupTableLength,
		&mr.header.BoneLookupTableOffset,
		&mr.header.TextureLookupTableLength,
		&mr.header.TextureLookupTableOffset,
		&mr.header.TextureUnitLookupTableLength,
		&mr.header.TextureUnitLookupTableOffset,
		&mr.header.TransparencyLookupTableLength,
		&mr.header.TransparencyLookupTableOffset,
		&mr.header.TextureTransformsLookupTableLength,
		&mr.header.TextureTransformsLookupTableOffset,
	} {
		if err := mr.readValue(f); err != nil {
			return err
		}
	}

	return nil
}
