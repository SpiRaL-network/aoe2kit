package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/preview"
)

func TestPreviewDefaultsAndCropCLI(t *testing.T) {
	root := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Error(err)
		}
	})
	if err := os.WriteFile("kit-preview.json", []byte(`{"sprites":true,"textures":true,"shadows":true,"elevation":true,"scale":60}`), 0644); err != nil {
		t.Fatal(err)
	}
	s := preview.Snapshot{Schema: preview.SnapshotSchema, Width: 10, Height: 10}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			s.Terrain = append(s.Terrain, preview.Tile{X: x, Y: y})
		}
	}
	if err := preview.WriteSnapshot("scene.json", s); err != nil {
		t.Fatal(err)
	}
	runPreview([]string{"scene.json", "--no-sprites", "--no-textures", "--no-shadows", "--no-elevation", "--around", "1,1", "--radius", "2", "--projection", "topdown", "--scale", "4", "--out", "crop.png"})
	f, err := os.Open(filepath.Join(root, "crop.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 12 || img.Bounds().Dy() != 12 {
		t.Fatal(img.Bounds())
	}
	runPreview([]string{"scene.json", "--no-sprites", "--no-textures", "--no-shadows", "--no-elevation", "--around", "1,1", "--radius", "2", "--projection", "topdown", "--out", "default-scale.png"})
	configured, err := os.Open("default-scale.png")
	if err != nil {
		t.Fatal(err)
	}
	defer configured.Close()
	img, err = png.Decode(configured)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 180 || img.Bounds().Dy() != 180 {
		t.Fatal("configured scale ignored", img.Bounds())
	}
}
