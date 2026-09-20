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
