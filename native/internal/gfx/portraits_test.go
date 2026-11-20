package gfx

import (
	"testing"

	"github.com/openkakutou/game/native/internal/ui"
)

// countingCache is a portrait cache whose decoder records its calls and
// gives every character a 1x1 texture without touching OpenGL.
func countingCache(calls *int) *portraitCache {
	c := newPortraitCache()
	c.decode = func(ref ui.PortraitRef) *Texture {
		*calls++
		return &Texture{W: 1, H: 1}
	}
	return c
}

func TestPortraitCache_DecodesAtMostTheBudgetPerFrame(t *testing.T) {
	calls := 0
	c := countingCache(&calls)
	c.newFrame()
	shown := 0
	for i := range 3 * portraitsPerFrame {
		if c.texture(ui.PortraitRef{Character: i}) != nil {
			shown++
		}
	}
	if calls != portraitsPerFrame || shown != portraitsPerFrame {
		t.Fatalf("first frame: %d decoded, %d shown, want %d", calls, shown, portraitsPerFrame)
	}
	for range 2 {
		c.newFrame()
		for i := range 3 * portraitsPerFrame {
			c.texture(ui.PortraitRef{Character: i})
		}
	}
	if calls != 3*portraitsPerFrame {
		t.Fatalf("after 3 frames: %d decoded, want every portrait once (%d)", calls, 3*portraitsPerFrame)
	}
}

func TestPortraitCache_AMissingPortraitIsDecodedOnce(t *testing.T) {
	calls := 0
	c := newPortraitCache()
	c.decode = func(ui.PortraitRef) *Texture { calls++; return nil }
	for range 3 {
		c.newFrame()
		if c.texture(ui.PortraitRef{Character: 0}) != nil {
			t.Fatal("no portrait: want nil")
		}
	}
	if calls != 1 {
		t.Fatalf("decoded %d times, want once: a missing portrait is remembered", calls)
	}
}
