package adt

import (
	"encoding/binary"
)

const (
	// LayerFlagCompressed marks an MCAL alpha map stored with RLE compression.
	LayerFlagCompressed = 0x200
	// ChunkFlagDoNotFixAlpha means the 64th alpha row/column is valid data.
	ChunkFlagDoNotFixAlpha = 0x8000

	AlphaMapSize = 64
	MaxLayers    = 4
)

// parseLayers reads the MCLY entries and decodes each layer's MCAL alpha map
// into c.Layers / c.AlphaMaps. The first layer has no alpha map (it is the base).
func parseLayers(c *Chunk, data []byte) {
	nLayers := int(c.Header.NLayers)
	if nLayers <= 0 || c.Header.OfsMCLY == 0 {
		return
	}
	if nLayers > MaxLayers {
		nLayers = MaxLayers
	}

	ofs := subChunkOffset(c.Header.OfsMCLY)
	if ofs+4 <= len(data) {
		tag := string(data[ofs : ofs+4])
		if tag == "MCLY" || tag == "YLCM" {
			ofs += 8
		}
	}
	if ofs+nLayers*16 > len(data) {
		return
	}

	c.Layers = make([]Layer, nLayers)
	for i := range c.Layers {
		o := ofs + i*16
		c.Layers[i] = Layer{
			TextureID:  binary.LittleEndian.Uint32(data[o : o+4]),
			Flags:      binary.LittleEndian.Uint32(data[o+4 : o+8]),
			OffsetMCAL: binary.LittleEndian.Uint32(data[o+8 : o+12]),
			EffectID:   binary.LittleEndian.Uint32(data[o+12 : o+16]),
		}
	}

	if nLayers < 2 || c.Header.OfsMCAL == 0 {
		return
	}

	// MCAL sub-chunk: trust its own size header over sizeAlpha
	aofs := subChunkOffset(c.Header.OfsMCAL)
	if aofs+8 > len(data) {
		return
	}
	alphaLen := int(c.Header.SizeMCAL)
	if tag := string(data[aofs : aofs+4]); tag == "MCAL" || tag == "LACM" {
		alphaLen = int(binary.LittleEndian.Uint32(data[aofs+4 : aofs+8]))
		aofs += 8
	}
	if aofs+alphaLen > len(data) {
		alphaLen = len(data) - aofs
	}
	mcal := data[aofs : aofs+alphaLen]

	c.AlphaMaps = make([][AlphaMapSize * AlphaMapSize]uint8, nLayers-1)
	for i := 1; i < nLayers; i++ {
		start := int(c.Layers[i].OffsetMCAL)
		if start >= len(mcal) {
			continue
		}
		// Byte length available to this layer: up to the next layer's data
		end := len(mcal)
		if i+1 < nLayers && int(c.Layers[i+1].OffsetMCAL) > start && int(c.Layers[i+1].OffsetMCAL) <= len(mcal) {
			end = int(c.Layers[i+1].OffsetMCAL)
		}
		decodeAlphaMap(&c.AlphaMaps[i-1], mcal[start:end], c.Layers[i].Flags&LayerFlagCompressed != 0)
		if c.Header.Flags&ChunkFlagDoNotFixAlpha == 0 {
			fixAlphaMapEdges(&c.AlphaMaps[i-1])
		}
	}
}

// decodeAlphaMap expands an MCAL layer into 64x64 8-bit alpha values.
// Uncompressed maps are 4-bit (2048 bytes) or 8-bit (4096 bytes).
func decodeAlphaMap(dst *[AlphaMapSize * AlphaMapSize]uint8, src []byte, compressed bool) {
	switch {
	case compressed:
		out := 0
		for i := 0; i < len(src) && out < len(dst); {
			fill := src[i]&0x80 != 0
			count := int(src[i] & 0x7F)
			i++
			if fill {
				if i >= len(src) {
					return
				}
				for k := 0; k < count && out < len(dst); k++ {
					dst[out] = src[i]
					out++
				}
				i++
			} else {
				for k := 0; k < count && out < len(dst) && i < len(src); k++ {
					dst[out] = src[i]
					out++
					i++
				}
			}
		}
	case len(src) >= AlphaMapSize*AlphaMapSize:
		copy(dst[:], src[:AlphaMapSize*AlphaMapSize])
	default:
		// 4-bit: low nibble first
		for i := 0; i < len(src) && i*2+1 < len(dst); i++ {
			dst[i*2] = (src[i] & 0x0F) * 17
			dst[i*2+1] = (src[i] >> 4) * 17
		}
	}
}

// fixAlphaMapEdges duplicates row/column 62 into 63, as the client does for
// maps without the "do not fix" flag.
func fixAlphaMapEdges(m *[AlphaMapSize * AlphaMapSize]uint8) {
	for i := 0; i < AlphaMapSize; i++ {
		m[i*AlphaMapSize+63] = m[i*AlphaMapSize+62]
		m[63*AlphaMapSize+i] = m[62*AlphaMapSize+i]
	}
}
