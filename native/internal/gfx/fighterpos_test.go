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
