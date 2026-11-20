package game

import "testing"

// randomFlow is the test flow on its character select with the random cell on.
func randomFlow(t *testing.T, draw int) *Flow {
	t.Helper()
	f := newTitleFlow(t)
	f.RandomCell = true
	f.Rand = func(n int) int { return draw % n }
	return inMode(t, f)
}

func TestRandomCell_IsTheCellAfterTheLastCharacter(t *testing.T) {
	f := randomFlow(t, 0)
	n := len(f.VisibleChars())
	if f.SelectCells() != n+1 {
		t.Fatalf("cells = %d, want the %d characters and the random cell", f.SelectCells(), n)
	}
	for range n + 3 {
		f.Navigate(NavRight)
	}
	if f.Cursor > n {
		t.Fatalf("cursor = %d, beyond the random cell %d", f.Cursor, n)
	}
}

func TestRandomCell_ConfirmPicksADrawnCharacterWithTheRandomSound(t *testing.T) {
	f := randomFlow(t, 1)
	f.Cursor = len(f.VisibleChars())
	f.DrainMenuCues(nil)
	f.Navigate(NavConfirm)
	if f.Picks[0] != f.VisibleChars()[1] {
		t.Fatalf("pick = %d, want the drawn character %d", f.Picks[0], f.VisibleChars()[1])
	}
	cues := f.DrainMenuCues(nil)
	if len(cues) != 1 || cues[0].Sound != SoundRandom {
		t.Fatalf("cues = %+v, want the random sound alone", cues)
	}
}

func TestRandomCell_OffWithoutThePacksRandomCell(t *testing.T) {
	f := newTestFlow(t)
	n := len(f.VisibleChars())
	if f.SelectCells() != n {
		t.Fatalf("cells = %d, want %d: no random cell unless the screenpack draws one", f.SelectCells(), n)
	}
	f.Cursor = n - 1
	f.Navigate(NavConfirm)
	if f.Picks[0] != f.VisibleChars()[n-1] {
		t.Fatal("the last character is picked as usual")
	}
}

func TestRandomCell_CancelAfterARandomPickReturnsToThePickedCharacter(t *testing.T) {
	f := randomFlow(t, 1)
	f.Cursor = len(f.VisibleChars())
	f.Navigate(NavConfirm) // P1 random
	f.Navigate(NavCancel)  // back to P1
	if f.Cursor != 1 {
		t.Fatalf("cursor = %d, want the drawn character's cell 1", f.Cursor)
	}
}
