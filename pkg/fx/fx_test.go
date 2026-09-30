package fx

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDescriptorTrail(t *testing.T) {
	grid := Grid{Rows: 4, Cols: 4, Frames: 16}
	alphaEnd := 0.25
	doc, err := NewDescriptor("aoe2kit_fx_smoke", "trail", grid, DescriptorOverrides{AlphaEnd: &alphaEnd})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := doc["AtlasFile"], `textures\atlases\aoe2kit_fx_smoke_atlas.dds`; got != want {
		t.Fatalf("AtlasFile = %v, want %v", got, want)
	}
	if got := doc["ImageCount"]; got != 16 {
		t.Fatalf("ImageCount = %v, want 16", got)
	}
	if got := doc["AlphaEnd"]; got != alphaEnd {
		t.Fatalf("AlphaEnd = %v, want %v", got, alphaEnd)
	}
}

func TestFrameSizeInfersGrid(t *testing.T) {
	grid, err := ParseFrameSize("16x32")
	if err != nil {
		t.Fatal(err)
	}
	grid.Frames = 7
	filled, err := fillGridDimensions(grid, 64, 64)
	if err != nil {
		t.Fatal(err)
	}
	if filled.Cols != 4 || filled.Rows != 2 || filled.Frames != 7 {
		t.Fatalf("filled grid = %+v, want 4 cols/2 rows/7 frames", filled)
	}
}

func TestBindDDSCreatesParticleTriangleAndDatBinding(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	sourceDat := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	if _, err := os.Stat(sourceDat); err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	tmp := t.TempDir()
	common := filepath.Join(tmp, "resources", "_common")
	datDir := filepath.Join(common, "dat")
	if err := os.MkdirAll(datDir, 0755); err != nil {
		t.Fatal(err)
	}
	datPath := filepath.Join(datDir, "empires2_x2_p1.dat")
	copyTestFile(t, sourceDat, datPath)
	ddsPath := filepath.Join(tmp, "source.dds")
	writeTestDDS(t, ddsPath, 64, 64)

	report, err := Bind(BindOptions{
		DDSPath:     ddsPath,
		Grid:        Grid{Rows: 4, Cols: 4},
		IntoCommon:  common,
		DatPath:     datPath,
		UnitID:      36,
		Slot:        "flying",
		FromGraphic: 1711,
		Name:        "aoe2kit_fx_bind_test",
		Preset:      "trail",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.NewGraphicID <= 0 || len(report.UnitPatches) == 0 {
		t.Fatalf("bad bind report: %+v", report)
	}
	lint, err := Lint(LintOptions{ModPath: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if !lint.OK {
		data, _ := json.MarshalIndent(lint, "", "  ")
		t.Fatalf("lint failed:\n%s", data)
	}
}

func copyTestFile(t *testing.T, input, output string) {
	t.Helper()
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeTestDDS(t *testing.T, path string, width, height int) {
	t.Helper()
	data := make([]byte, 128)
	copy(data[:4], []byte("DDS "))
	binary.LittleEndian.PutUint32(data[4:8], 124)
	binary.LittleEndian.PutUint32(data[12:16], uint32(height))
	binary.LittleEndian.PutUint32(data[16:20], uint32(width))
	binary.LittleEndian.PutUint32(data[76:80], 32)
	binary.LittleEndian.PutUint32(data[80:84], 4)
	copy(data[84:88], []byte("DXT5"))
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
