package blp

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"testing"
)

func buildTestBLPHeader(width, height uint32, encoding, alphaDepth, alphaType uint8, dataSize uint32) []byte {
	buf := make([]byte, 148+dataSize)
	binary.LittleEndian.PutUint32(buf[0:4], MagicBLP2)
	binary.LittleEndian.PutUint32(buf[4:8], TypeDirect)
	buf[8] = encoding
	buf[9] = alphaDepth
	buf[10] = alphaType
	buf[11] = 0 // hasMips
	binary.LittleEndian.PutUint32(buf[12:16], width)
	binary.LittleEndian.PutUint32(buf[16:20], height)

	// Mip 0 offset and size
	binary.LittleEndian.PutUint32(buf[20:24], 148)
	binary.LittleEndian.PutUint32(buf[84:88], dataSize)
	return buf
}

func TestDecodeDXT1(t *testing.T) {
	// A 4x4 image in DXT1 is 8 bytes
	rawBLP := buildTestBLPHeader(4, 4, EncodingDXT, 0, AlphaTypeDXT1, 8)

	// Color0: Red (RGB565: 11111 000000 00000 = 0xF800)
	// Color1: Blue (RGB565: 00000 000000 11111 = 0x001F)
	// Lookup: all 0s (so all pixels are color0)
	dxtBlock := []byte{
		0x00, 0xF8, // c0
		0x1F, 0x00, // c1
		0x00, 0x00, 0x00, 0x00, // all pixel index 0
	}
	copy(rawBLP[148:], dxtBlock)

	img, err := Decode(bytes.NewReader(rawBLP))
	if err != nil {
		t.Fatalf("Decode DXT1 failed: %v", err)
	}

	if img.Bounds().Dx() != 4 || img.Bounds().Dy() != 4 {
		t.Fatalf("expected 4x4, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	c := img.At(0, 0).(color.NRGBA)
	if c.R < 240 || c.B > 10 || c.A != 255 {
		t.Fatalf("expected red pixel, got %+v", c)
	}

	// Verify PNG encoding
	var pngBuf bytes.Buffer
	if err := ToPNG(img, &pngBuf); err != nil {
		t.Fatalf("ToPNG failed: %v", err)
	}
	if pngBuf.Len() == 0 {
		t.Fatal("empty PNG output")
	}

	// Verify WebP encoding
	var webpBuf bytes.Buffer
	if err := ToWebP(img, &webpBuf, 85.0, true); err != nil {
		t.Fatalf("ToWebP failed: %v", err)
	}
	if webpBuf.Len() == 0 {
		t.Fatal("empty WebP output")
	}
}

func TestDecodeDXT5(t *testing.T) {
	// A 4x4 image in DXT5 is 16 bytes: 8 bytes alpha, 8 bytes RGB
	rawBLP := buildTestBLPHeader(4, 4, EncodingDXT, 8, AlphaTypeDXT5, 16)

	dxt5Block := []byte{
		0xFF, 0x00, // alpha0 = 255, alpha1 = 0
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // all alpha index 0 (alpha0 = 255)
		0x00, 0xF8, // c0 (red)
		0x1F, 0x00, // c1 (blue)
		0x00, 0x00, 0x00, 0x00, // index 0 (c0)
	}
	copy(rawBLP[148:], dxt5Block)

	img, err := Decode(bytes.NewReader(rawBLP))
	if err != nil {
		t.Fatalf("Decode DXT5 failed: %v", err)
	}

	c := img.At(2, 2).(color.NRGBA)
	if c.R < 240 || c.A != 255 {
		t.Fatalf("expected full opacity red pixel, got %+v", c)
	}
}

func TestDecodeDXT3AlphaExpansion(t *testing.T) {
	// A 4x4 DXT3 block: 8 bytes of 4-bit alpha, then a DXT1 color block.
	// The alpha nibbles must expand to the full 0..255 range: nibble*17
	// (15*17 == 255). The multiply used to happen in uint8, wrapping every
	// nibble above 1 down to a near-transparent value.
	rawBLP := buildTestBLPHeader(4, 4, EncodingDXT, 8, AlphaTypeDXT3, 16)

	// Alpha nibbles (pixel order): 0, 1, 8, 15, then 15 for the rest.
	alpha := uint64(0)
	for i, n := range []uint64{0, 1, 8, 15} {
		alpha |= n << (i * 4)
	}
	for i := 4; i < 16; i++ {
		alpha |= 15 << (i * 4)
	}

	dxt3Block := make([]byte, 16)
	binary.LittleEndian.PutUint64(dxt3Block[0:8], alpha)
	// Color block: c0 red > c1 blue, every pixel index 0 (opaque red).
	copy(dxt3Block[8:], []byte{0x00, 0xF8, 0x1F, 0x00, 0x00, 0x00, 0x00, 0x00})
	copy(rawBLP[148:], dxt3Block)

	img, err := Decode(bytes.NewReader(rawBLP))
	if err != nil {
		t.Fatalf("Decode DXT3 failed: %v", err)
	}

	want := []uint8{0, 17, 136, 255}
	for i, expected := range want {
		got := img.At(i, 0).(color.NRGBA).A
		if got != expected {
			t.Errorf("pixel %d alpha: got %d, want %d", i, got, expected)
		}
	}
	if got := img.At(3, 3).(color.NRGBA).A; got != 255 {
		t.Errorf("opaque pixel alpha: got %d, want 255", got)
	}
}

func TestDecodePaletted4BitAlpha(t *testing.T) {
	// Paletted texture with alphaDepth 4: the nibbles after the index data
	// expand to 0..255 the same way.
	rawBLP := buildTestBLPHeader(2, 2, EncodingUncompressed, 4, 8, 4+2)
	// Palette entry 0 is opaque white; the mip data starts after the palette.
	rawBLP = append(rawBLP[:148], append(make([]byte, 1024), 0, 0, 0, 0, 0, 0)...)
	binary.LittleEndian.PutUint32(rawBLP[20:24], 148+1024)
	rawBLP[148+0], rawBLP[148+1], rawBLP[148+2], rawBLP[148+3] = 255, 255, 255, 255
	// Index data: all pixels use entry 0. Alpha nibbles: 0, 15, 8, 1.
	rawBLP[148+1024+0] = 0
	rawBLP[148+1024+4] = 0x0F // pixel 0 low nibble 15, pixel 1 high nibble 0
	rawBLP[148+1024+5] = 0x18 // pixel 2 low nibble 8, pixel 3 high nibble 1

	img, err := Decode(bytes.NewReader(rawBLP))
	if err != nil {
		t.Fatalf("Decode paletted failed: %v", err)
	}

	want := []uint8{255, 0, 136, 17}
	for i, expected := range want {
		got := img.At(i%2, i/2).(color.NRGBA).A
		if got != expected {
			t.Errorf("pixel %d alpha: got %d, want %d", i, got, expected)
		}
	}
}

func TestDecodePalettedWithoutAlphaIsOpaque(t *testing.T) {
	// A 2x2 paletted texture with alpha depth 0: the palette's alpha bytes are
	// zero (as in the character skins), yet the pixels must come out opaque
	rawBLP := buildTestBLPHeader(2, 2, EncodingUncompressed, 0, 8, 4)
	// Palette entry 0: BGRA = (10, 20, 30, 0)
	rawBLP[148+0], rawBLP[148+1], rawBLP[148+2], rawBLP[148+3] = 10, 20, 30, 0
	// The mip data follows the palette; the header points it at 148, so
	// rebuild the offset to skip the 1024-byte palette
	rawBLP = append(rawBLP[:148], append(make([]byte, 1024), 0, 0, 0, 0)...)
	binary.LittleEndian.PutUint32(rawBLP[20:24], 148+1024)
	rawBLP[148+0], rawBLP[148+1], rawBLP[148+2], rawBLP[148+3] = 10, 20, 30, 0

	img, err := Decode(bytes.NewReader(rawBLP))
	if err != nil {
		t.Fatalf("Decode paletted failed: %v", err)
	}
	c := img.At(1, 1).(color.NRGBA)
	if c != (color.NRGBA{R: 30, G: 20, B: 10, A: 255}) {
		t.Fatalf("expected opaque (30, 20, 10), got %+v", c)
	}
}
