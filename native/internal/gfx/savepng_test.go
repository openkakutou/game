package gfx

import (
	"image"
	"os"
	"path/filepath"
	"testing"
)

// A capture path without .png is refused before anything is written: the
// file already there keeps its content.
func TestWritePNG_RefusesANonPNGPathWithoutTouchingTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(path, image.NewRGBA(image.Rect(0, 0, 2, 2))); err == nil {
		t.Fatal("writePNG(notes.txt) = nil, want the .png suffix required")
	}
	if got, _ := os.ReadFile(path); string(got) != "keep me" {
		t.Errorf("notes.txt = %q, want it untouched", got)
	}
}

// A .png path gets the image, readable back.
func TestWritePNG_WritesAReadablePNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shot.png")
	if err := writePNG(path, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil || format != "png" || cfg.Width != 3 || cfg.Height != 2 {
		t.Fatalf("decoded %s %dx%d (%v), want a 3x2 png", format, cfg.Width, cfg.Height, err)
	}
}

// A path in a folder that does not exist fails.
func TestWritePNG_MissingFolderFails(t *testing.T) {
	if err := writePNG(filepath.Join(t.TempDir(), "nope", "shot.png"), image.NewRGBA(image.Rect(0, 0, 1, 1))); err == nil {
		t.Fatal("writePNG into a missing folder = nil, want an error")
	}
}
