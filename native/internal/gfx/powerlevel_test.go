package gfx

import "testing"

// The power bar fills against the fighter's own PowerMax: a 5000-power
// character at 3000 is 60% full, not full (engine v2.77.3).
func TestPowerLevel_FillsAgainstTheFightersPowerMax(t *testing.T) {
	for _, c := range []struct {
		p    powerLevel
		want float64
	}{
		{powerLevel{value: 3000, max: 5000}, 0.6},
		{powerLevel{value: 3000, max: 3000}, 1},
		{powerLevel{value: 1500, max: 1000}, 1},
		{powerLevel{value: 0, max: 0}, 0},
	} {
		if got := c.p.fraction(); got != c.want {
			t.Errorf("%+v.fraction() = %v, want %v", c.p, got, c.want)
		}
	}
}
