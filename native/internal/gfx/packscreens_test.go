package gfx

import "testing"

// Every screenpack screen drawn here has one Flow.PackScreen name: a screen
// added to one list and not the other fails here.
func TestPackScreens_EachScreenHasOneName(t *testing.T) {
	seen := map[packScreen]string{}
	for name, s := range packScreens {
		if s == packNone {
			t.Errorf("%q maps to no screen", name)
		}
		if other, dup := seen[s]; dup {
			t.Errorf("%q and %q map to the same screen", name, other)
		}
		seen[s] = name
	}
	for s := packTitle; s <= packError; s++ {
		if _, ok := seen[s]; !ok {
			t.Errorf("screen %d has no name", s)
		}
	}
}
