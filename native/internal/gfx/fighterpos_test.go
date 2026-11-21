package gfx

import "testing"

// The engine's Y points up (engine statemachine worldY): a fighter 30 units
// in the air is drawn 30 view pixels above the ground line, not below it.
func TestFighterScreenY_AirborneIsAboveTheGround(t *testing.T) {
	if got := fighterScreenY(180, 30); got != 150 {
		t.Errorf("y 30 over ground 180 = %v, want 150", got)
	}
	if got := fighterScreenY(180, 0); got != 180 {
		t.Errorf("on the ground = %v, want 180", got)
	}
}

// A frame's blend in the .air (Ikemen GO anim.go): A, A1 and AS…D… add the
// sprite to what is behind, so its black stays invisible.
func TestAdditiveBlend(t *testing.T) {
	for blend, want := range map[string]bool{"A": true, "a1": true, "AS256D256": true, "": false, "S": false, " A ": true} {
		if got := additiveBlend(blend); got != want {
			t.Errorf("additiveBlend(%q) = %v, want %v", blend, got, want)
		}
	}
}
