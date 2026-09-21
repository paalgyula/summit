package assetserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paalgyula/summit/pkg/converter/blp"
	"github.com/paalgyula/summit/pkg/converter/dbc"
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
	_ = os.MkdirAll(texDir, 0o755)
	if err := os.WriteFile(filepath.Join(texDir, "Grass.blp"), blpData, 0o644); err != nil {
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
	got := []string{
		"Data/common.MPQ", "Data/common-2.MPQ", "Data/lichking.MPQ", "Data/enUS/locale-enUS.MPQ",
		"Data/patch.MPQ", "Data/patch-2.MPQ", "Data/patch-3.MPQ", "Data/enUS/patch-enUS.MPQ", "Data/enUS/patch-enUS-2.MPQ",
	}
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

func TestAssetServerCharacterData(t *testing.T) {
	tmpDir := t.TempDir()

	// A one-race ChrRaces.dbc: 69 columns, the client prefix and name as strings
	strs := []byte("\x00Hu\x00Human\x00")
	row := make([]byte, 69*4)
	binary.LittleEndian.PutUint32(row[0:], 1)
	binary.LittleEndian.PutUint32(row[6*4:], 1)
	binary.LittleEndian.PutUint32(row[14*4:], 4)
	file := []byte("WDBC")
	for _, v := range []uint32{1, 69, 69 * 4, uint32(len(strs))} {
		file = binary.LittleEndian.AppendUint32(file, v)
	}
	file = append(append(file, row...), strs...)
	if err := os.MkdirAll(filepath.Join(tmpDir, "DBFilesClient"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "DBFilesClient", "ChrRaces.dbc"), file, 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := NewServer(Config{AssetDir: tmpDir})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/"+CharacterDataPath, nil)
	w := httptest.NewRecorder()
	srv.Echo().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data struct {
		Races map[string]struct{ Name, Prefix string } `json:"races"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if data.Races["1"].Name != "Human" || data.Races["1"].Prefix != "Hu" {
		t.Fatalf("unexpected races: %+v", data.Races)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".cache", "dbc", "character.json")); err != nil {
		t.Fatalf("character data not cached: %v", err)
	}
}

func TestDebugCreatureDBC(t *testing.T) {
	srv, err := NewServer(Config{
		AssetDir: "../../client/assets",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("creature displays count: %d", len(srv.creatures.displays))
	// Trigger load
	_, _ = srv.CreatureDisplay(1)
	t.Logf("load err: %v, displays count: %d, models: %d, extras: %d", srv.creatures.err, len(srv.creatures.displays), len(srv.creatures.models), len(srv.creatures.extras))
	count := 0
	for id, row := range srv.creatures.displays {
		t.Logf("display sample ID: %d, row: %+v", id, row)
		count++
		if count > 5 {
			break
		}
	}
	count = 0
	for id, row := range srv.creatures.models {
		t.Logf("model sample ID: %d, row: %+v", id, row)
		count++
		if count > 5 {
			break
		}
	}
	var zeroModelDisplays []uint32
	var extraWithZeroModel []uint32
	for id, row := range srv.creatures.displays {
		if row.modelID == 0 {
			zeroModelDisplays = append(zeroModelDisplays, id)
			if row.extraID != 0 {
				extraWithZeroModel = append(extraWithZeroModel, id)
			}
		}
	}
	t.Logf("displays with modelID == 0: %d, with extra: %d", len(zeroModelDisplays), len(extraWithZeroModel))
	// Print raw columns for display 612 and some other displays
	cdiData, _ := srv.readSource("DBFilesClient/CreatureDisplayInfo.dbc")
	cdiFile, _ := dbc.Read(bytes.NewReader(cdiData))
	t.Logf("CDI fields: %d, records: %d, recordSize: %d", cdiFile.Fields, cdiFile.Records, cdiFile.RecordSize)
	for r := 0; r < 3; r++ {
		var cols []string
		for c := 0; c < cdiFile.Fields; c++ {
			u := cdiFile.Uint32(r, c)
			s := cdiFile.String(r, c)
			if s != "" {
				cols = append(cols, fmt.Sprintf("c%d:str(%s)", c, s))
			} else {
				cols = append(cols, fmt.Sprintf("c%d:%d", c, u))
			}
		}
		t.Logf("CDI row %d: %s", r, strings.Join(cols, ", "))
	}

	// Check texture formats across all displays
	var examplesWithSlash []string
	var examplesWithBackslash []string
	var examplesWithExt []string
	var examplesEmptyModel []uint32
	for id, row := range srv.creatures.displays {
		if _, ok := srv.creatures.models[row.modelID]; !ok {
			examplesEmptyModel = append(examplesEmptyModel, id)
		}
		for _, tex := range row.textures {
			if tex == "" {
				continue
			}
			if strings.Contains(tex, "/") {
				examplesWithSlash = append(examplesWithSlash, tex)
			}
			if strings.Contains(tex, "\\") {
				examplesWithBackslash = append(examplesWithBackslash, tex)
			}
			if strings.Contains(tex, ".") {
				examplesWithExt = append(examplesWithExt, tex)
			}
		}
	}
	t.Logf("textures with /: %d (e.g. %v)", len(examplesWithSlash), head(examplesWithSlash, 3))
	t.Logf("textures with \\: %d (e.g. %v)", len(examplesWithBackslash), head(examplesWithBackslash, 3))
	t.Logf("textures with .: %d (e.g. %v)", len(examplesWithExt), head(examplesWithExt, 3))
	t.Logf("displays missing model: %d", len(examplesEmptyModel))
}

func head(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
