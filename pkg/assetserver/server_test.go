package assetserver

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/paalgyula/summit/pkg/converter/blp"
)

func TestAssetServerHealthAndCORS(t *testing.T) {
	tmpDir := t.TempDir()
	srv, err := NewServer(Config{
		AssetDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Test /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	srv.Echo().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing or invalid CORS origin header: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Test OPTIONS preflight
	optReq := httptest.NewRequest(http.MethodOptions, "/assets/model.glb", nil)
	optReq.Header.Set("Origin", "http://localhost:3000")
	optReq.Header.Set("Access-Control-Request-Method", "GET")
	optW := httptest.NewRecorder()
	srv.Echo().ServeHTTP(optW, optReq)

	if optW.Code != http.StatusNoContent && optW.Code != http.StatusOK {
		t.Fatalf("expected 200 or 204 for OPTIONS, got %d", optW.Code)
	}
}

func TestAssetServerJITWebP(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a 4x4 DXT1 BLP file into AssetDir
	blpData := make([]byte, 148+8)
	binary.LittleEndian.PutUint32(blpData[0:4], blp.MagicBLP2)
	binary.LittleEndian.PutUint32(blpData[4:8], blp.TypeDirect)
	blpData[8] = blp.EncodingDXT
	blpData[9] = 0
	blpData[10] = blp.AlphaTypeDXT1
	binary.LittleEndian.PutUint32(blpData[12:16], 4)
	binary.LittleEndian.PutUint32(blpData[16:20], 4)
	binary.LittleEndian.PutUint32(blpData[20:24], 148)
	binary.LittleEndian.PutUint32(blpData[84:88], 8)
	// Red color block
	copy(blpData[148:], []byte{0x00, 0xF8, 0x00, 0x00, 0, 0, 0, 0})

	texDir := filepath.Join(tmpDir, "Textures")
	_ = os.MkdirAll(texDir, 0755)
	if err := os.WriteFile(filepath.Join(texDir, "Grass.blp"), blpData, 0644); err != nil {
		t.Fatalf("failed to write test BLP: %v", err)
	}

	srv, err := NewServer(Config{
		AssetDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Request Grass.webp (which does NOT exist yet on disk)
	req := httptest.NewRequest(http.MethodGet, "/Textures/Grass.webp", nil)
	w := httptest.NewRecorder()
	srv.Echo().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from JIT WebP conversion, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("expected image/webp, got %s", w.Header().Get("Content-Type"))
	}

	// Verify that the cached WebP was created
	cachedWebP := filepath.Join(tmpDir, ".cache", "Textures", "Grass.webp")
	if _, err := os.Stat(cachedWebP); err != nil {
		t.Fatalf("expected cached webp at %s, error: %v", cachedWebP, err)
	}

	// Remove source BLP to prove that second request is served entirely from cache (NO JIT!)
	_ = os.Remove(filepath.Join(texDir, "Grass.blp"))

	req2 := httptest.NewRequest(http.MethodGet, "/Textures/Grass.webp", nil)
	w2 := httptest.NewRecorder()
	srv.Echo().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from cache hit, got %d", w2.Code)
	}
	if w2.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("expected image/webp, got %s", w2.Header().Get("Content-Type"))
	}
}

func TestMPQLoadOrder(t *testing.T) {
	want := []int{0, 0, 0, 1, 2, 2, 2, 3, 3}
	got := []string{"Data/common.MPQ", "Data/common-2.MPQ", "Data/lichking.MPQ", "Data/enUS/locale-enUS.MPQ",
		"Data/patch.MPQ", "Data/patch-2.MPQ", "Data/patch-3.MPQ", "Data/enUS/patch-enUS.MPQ", "Data/enUS/patch-enUS-2.MPQ"}
	for i, p := range got {
		if mpqRank(p) != want[i] {
			t.Errorf("rank(%s) = %d, want %d", p, mpqRank(p), want[i])
		}
	}
	if mpqNumber("Data/patch-3.MPQ") != 3 || mpqNumber("Data/patch.MPQ") != 1 || mpqNumber("Data/enUS/patch-enUS-2.MPQ") != 2 {
		t.Errorf("mpqNumber")
	}
	if !isLocaleDir("enUS") || isLocaleDir("Data") || isLocaleDir("Interface") {
		t.Errorf("isLocaleDir")
	}
}
