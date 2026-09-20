package blp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/KarpelesLab/gowebp"
)

const (
	MagicBLP1 = 0x31504C42 // "BLP1"
	MagicBLP2 = 0x32504C42 // "BLP2"

	TypeJPEG   = 0
	TypeDirect = 1

	EncodingUncompressed = 1
	EncodingDXT          = 2
	EncodingARGB         = 3

	AlphaTypeDXT1 = 0
	AlphaTypeDXT3 = 1
	AlphaTypeDXT5 = 7
)

var (
	ErrNotBLP            = errors.New("not a valid BLP texture")
	ErrUnsupportedFormat = errors.New("unsupported BLP format or compression")
)

type Header struct {
	Magic         uint32
	Type          uint32
	Encoding      uint8
	AlphaDepth    uint8
	AlphaType     uint8
	HasMips       uint8
	Width         uint32
	Height        uint32
	MipmapOffsets [16]uint32
	MipmapSizes   [16]uint32
}

// Decode reads a BLP texture and returns the base (mip 0) image.
func Decode(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read blp data: %w", err)
	}

	if len(data) < 148 {
		return nil, ErrNotBLP
	}

	var h Header
	h.Magic = binary.LittleEndian.Uint32(data[0:4])
	if h.Magic != MagicBLP1 && h.Magic != MagicBLP2 {
		return nil, ErrNotBLP
	}

	h.Type = binary.LittleEndian.Uint32(data[4:8])
	h.Encoding = data[8]
	h.AlphaDepth = data[9]
	h.AlphaType = data[10]
	h.HasMips = data[11]
	h.Width = binary.LittleEndian.Uint32(data[12:16])
	h.Height = binary.LittleEndian.Uint32(data[16:20])

	for i := 0; i < 16; i++ {
		h.MipmapOffsets[i] = binary.LittleEndian.Uint32(data[20+i*4 : 24+i*4])
		h.MipmapSizes[i] = binary.LittleEndian.Uint32(data[84+i*4 : 88+i*4])
	}

	if h.Width == 0 || h.Height == 0 {
		return nil, errors.New("invalid BLP dimensions")
	}

	mipOffset := h.MipmapOffsets[0]
	mipSize := h.MipmapSizes[0]

	if uint32(len(data)) < mipOffset+mipSize {
		return nil, errors.New("BLP data truncated")
	}

	mipData := data[mipOffset : mipOffset+mipSize]

	switch h.Encoding {
	case EncodingDXT:
		return decodeDXT(mipData, int(h.Width), int(h.Height), h.AlphaType)

	case EncodingUncompressed:
		palette := make([]color.NRGBA, 256)
		paletteData := data[148 : 148+1024]
		for i := 0; i < 256; i++ {
			b := paletteData[i*4+0]
			g := paletteData[i*4+1]
			r := paletteData[i*4+2]
			a := paletteData[i*4+3]
			palette[i] = color.NRGBA{R: r, G: g, B: b, A: a}
		}
		return decodePaletted(mipData, int(h.Width), int(h.Height), int(h.AlphaDepth), palette)

	case EncodingARGB:
		return decodeARGB(mipData, int(h.Width), int(h.Height))

	default:
		return nil, fmt.Errorf("%w: encoding %d", ErrUnsupportedFormat, h.Encoding)
	}
}

// DecodeFile opens and decodes a BLP file from disk.
func DecodeFile(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Decode(f)
}

// ToPNG encodes an image as PNG into the given writer.
func ToPNG(img image.Image, w io.Writer) error {
	return png.Encode(w, img)
}

// SavePNG writes an image to disk as a PNG file.
func SavePNG(img image.Image, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ToWebP encodes an image as WebP into the given writer.
// If lossy is true, quality is 0..100 (e.g. 85.0).
// If lossy is false, it uses lossless VP8L encoding with alpha.
func ToWebP(img image.Image, w io.Writer, quality float32, lossy bool) error {
	opts := &gowebp.Options{
		Lossy:   lossy,
		Quality: quality,
	}
	return gowebp.Encode(w, img, opts)
}

// SaveWebP writes an image to disk as a WebP file.
func SaveWebP(img image.Image, destPath string, quality float32, lossy bool) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return ToWebP(img, f, quality, lossy)
}

// Save writes an image to disk using the file extension (.webp or .png) to choose the format.
func Save(img image.Image, destPath string) error {
	ext := strings.ToLower(filepath.Ext(destPath))
	if ext == ".webp" {
		return SaveWebP(img, destPath, 85.0, true)
	}
	return SavePNG(img, destPath)
}

func decodeARGB(data []byte, width, height int) (image.Image, error) {
	expected := width * height * 4
	if len(data) < expected {
		return nil, errors.New("insufficient ARGB data")
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < width*height; i++ {
		b := data[i*4+0]
		g := data[i*4+1]
		r := data[i*4+2]
		a := data[i*4+3]
		off := i * 4
		img.Pix[off+0] = r
		img.Pix[off+1] = g
		img.Pix[off+2] = b
		img.Pix[off+3] = a
	}
	return img, nil
}

func decodePaletted(data []byte, width, height, alphaDepth int, palette []color.NRGBA) (image.Image, error) {
	pixelCount := width * height
	if len(data) < pixelCount {
		return nil, errors.New("insufficient paletted index data")
	}

	indices := data[:pixelCount]
	alphaData := data[pixelCount:]

	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for i := 0; i < pixelCount; i++ {
		c := palette[indices[i]]
		alpha := c.A

		if alphaDepth == 8 && len(alphaData) > i {
			alpha = alphaData[i]
		} else if alphaDepth == 1 && len(alphaData) > (i/8) {
			bit := (alphaData[i/8] >> (i % 8)) & 1
			if bit == 0 {
				alpha = 0
			} else {
				alpha = 255
			}
		} else if alphaDepth == 4 && len(alphaData) > (i/2) {
			nibble := (alphaData[i/2] >> ((i % 2) * 4)) & 0x0F
			alpha = (nibble * 255) / 15
		}

		off := i * 4
		img.Pix[off+0] = c.R
		img.Pix[off+1] = c.G
		img.Pix[off+2] = c.B
		img.Pix[off+3] = alpha
	}

	return img, nil
}

func decodeDXT(data []byte, width, height int, alphaType uint8) (image.Image, error) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	dataOffset := 0

	bw := (width + 3) / 4
	bh := (height + 3) / 4

	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			var colors [16]color.NRGBA

			switch alphaType {
			case AlphaTypeDXT1:
				if dataOffset+8 > len(data) {
					return img, nil
				}
				decodeDXT1Block(data[dataOffset:dataOffset+8], &colors)
				dataOffset += 8

			case AlphaTypeDXT3:
				if dataOffset+16 > len(data) {
					return img, nil
				}
				decodeDXT3Block(data[dataOffset:dataOffset+16], &colors)
				dataOffset += 16

			case AlphaTypeDXT5:
				if dataOffset+16 > len(data) {
					return img, nil
				}
				decodeDXT5Block(data[dataOffset:dataOffset+16], &colors)
				dataOffset += 16

			default:
				// Fallback to DXT1
				if dataOffset+8 > len(data) {
					return img, nil
				}
				decodeDXT1Block(data[dataOffset:dataOffset+8], &colors)
				dataOffset += 8
			}

			// Stamp 4x4 block into image
			for py := 0; py < 4; py++ {
				y := by*4 + py
				if y >= height {
					continue
				}
				for px := 0; px < 4; px++ {
					x := bx*4 + px
					if x >= width {
						continue
					}
					c := colors[py*4+px]
					off := (y*width + x) * 4
					img.Pix[off+0] = c.R
					img.Pix[off+1] = c.G
					img.Pix[off+2] = c.B
					img.Pix[off+3] = c.A
				}
			}
		}
	}

	return img, nil
}

func rgb565ToNRGBA(c uint16, a uint8) color.NRGBA {
	r := uint8(((c >> 11) & 0x1F) * 255 / 31)
	g := uint8(((c >> 5) & 0x3F) * 255 / 63)
	b := uint8((c & 0x1F) * 255 / 31)
	return color.NRGBA{R: r, G: g, B: b, A: a}
}

func decodeDXT1Block(block []byte, out *[16]color.NRGBA) {
	c0 := binary.LittleEndian.Uint16(block[0:2])
	c1 := binary.LittleEndian.Uint16(block[2:4])
	lookup := binary.LittleEndian.Uint32(block[4:8])

	rgb0 := rgb565ToNRGBA(c0, 255)
	rgb1 := rgb565ToNRGBA(c1, 255)

	var palette [4]color.NRGBA
	palette[0] = rgb0
	palette[1] = rgb1

	if c0 > c1 {
		palette[2] = color.NRGBA{
			R: uint8((2*uint32(rgb0.R) + uint32(rgb1.R)) / 3),
			G: uint8((2*uint32(rgb0.G) + uint32(rgb1.G)) / 3),
			B: uint8((2*uint32(rgb0.B) + uint32(rgb1.B)) / 3),
			A: 255,
		}
		palette[3] = color.NRGBA{
			R: uint8((uint32(rgb0.R) + 2*uint32(rgb1.R)) / 3),
			G: uint8((uint32(rgb0.G) + 2*uint32(rgb1.G)) / 3),
			B: uint8((uint32(rgb0.B) + 2*uint32(rgb1.B)) / 3),
			A: 255,
		}
	} else {
		palette[2] = color.NRGBA{
			R: uint8((uint32(rgb0.R) + uint32(rgb1.R)) / 2),
			G: uint8((uint32(rgb0.G) + uint32(rgb1.G)) / 2),
			B: uint8((uint32(rgb0.B) + uint32(rgb1.B)) / 2),
			A: 255,
		}
		palette[3] = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	}

	for i := 0; i < 16; i++ {
		idx := (lookup >> (i * 2)) & 0x03
		out[i] = palette[idx]
	}
}

func decodeDXT3Block(block []byte, out *[16]color.NRGBA) {
	alphaBits := binary.LittleEndian.Uint64(block[0:8])
	decodeDXT1Block(block[8:16], out)

	for i := 0; i < 16; i++ {
		nibble := uint8((alphaBits >> (i * 4)) & 0x0F)
		out[i].A = (nibble * 255) / 15
	}
}

func decodeDXT5Block(block []byte, out *[16]color.NRGBA) {
	a0 := block[0]
	a1 := block[1]

	var alphas [8]uint8
	alphas[0] = a0
	alphas[1] = a1

	if a0 > a1 {
		alphas[2] = uint8((6*uint32(a0) + 1*uint32(a1)) / 7)
		alphas[3] = uint8((5*uint32(a0) + 2*uint32(a1)) / 7)
		alphas[4] = uint8((4*uint32(a0) + 3*uint32(a1)) / 7)
		alphas[5] = uint8((3*uint32(a0) + 4*uint32(a1)) / 7)
		alphas[6] = uint8((2*uint32(a0) + 5*uint32(a1)) / 7)
		alphas[7] = uint8((1*uint32(a0) + 6*uint32(a1)) / 7)
	} else {
		alphas[2] = uint8((4*uint32(a0) + 1*uint32(a1)) / 5)
		alphas[3] = uint8((3*uint32(a0) + 2*uint32(a1)) / 5)
		alphas[4] = uint8((2*uint32(a0) + 3*uint32(a1)) / 5)
		alphas[5] = uint8((1*uint32(a0) + 4*uint32(a1)) / 5)
		alphas[6] = 0
		alphas[7] = 255
	}

	// 48-bit index array (6 bytes)
	var aIndices uint64
	for i := 0; i < 6; i++ {
		aIndices |= uint64(block[2+i]) << (i * 8)
	}

	decodeDXT1Block(block[8:16], out)

	for i := 0; i < 16; i++ {
		idx := (aIndices >> (i * 3)) & 0x07
		out[i].A = alphas[idx]
	}
}
