package game

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

const reference = "../../testdata/reference"

// copyChar copies a reference character folder (flat files) into a new
// temporary folder and returns that folder as a file system and the .def's
// name inside it.
func copyChar(t *testing.T, name, def string, extra ...string) (fs.FS, string) {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join(reference, "chars", name)
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range extra {
		raw, err := os.ReadFile(filepath.Join(reference, "common", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, f), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return os.DirFS(dst), def
}

func startMatch(t *testing.T, p1, p2 *Fighter) *Match {
	t.Helper()
	m, err := NewMatch(p1, p2, closeStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// requireStartsTheRound expects both fighters to start the round as Ikemen
// GO starts them (system.go:2580-2595): in common1's state 5900 without
// control.
func requireStartsTheRound(t *testing.T, m *Match) {
	t.Helper()
	for i, rt := range m.runtimes {
		if rt.Context.StateNo != 5900 || rt.Context.Ctrl {
			t.Errorf("side %d starts in state %d (ctrl %v), want 5900 without control", i, rt.Context.StateNo, rt.Context.Ctrl)
		}
	}
}

// untilFight steps m with no input until the round's fight starts
// (RoundState 2), the intros played.
func untilFight(t *testing.T, m *Match) {
	t.Helper()
	for n := 0; n < 2000 && m.RoundState() != 2; n++ {
		if err := m.Step([2]input.TickInput{}); err != nil {
			t.Fatalf("tick %d: %v", n+1, err)
		}
	}
	if m.RoundState() != 2 {
		t.Fatal("the fight never started")
	}
}

// requireTicksIdle plays the intros, runs a few ticks of the fight and
// expects both fighters in the idle state 0 with anim 0 (common1's state 0
// is "ChangeAnim value = 0").
// Known engine gap: common1 state 0 reads Const(...), a trigger the engine
// lacks until its backlog item 042. That failure is kept as an expected
// failure (skipped with the item named), never weakened: once 042 lands the
// skip no longer triggers and the assertions below must pass.
func requireTicksIdle(t *testing.T, m *Match) {
	t.Helper()
	untilFight(t, m)
	for n := 0; n < 5; n++ {
		if err := m.Step([2]input.TickInput{}); err != nil {
			t.Fatalf("tick %d: %v", n+1, err)
		}
	}
	for i := range m.Anim {
		if m.Anim[i].No != 0 || m.runtimes[i].Context.StateNo != 0 {
			t.Errorf("side %d after ticks: state %d anim %d, want state 0 anim 0", i, m.runtimes[i].Context.StateNo, m.Anim[i].No)
		}
	}
}

func TestNewMatch_CnsAndZssKungFuManStartWithCommonFolder(t *testing.T) {
	cns, err := LoadFighter(os.DirFS(reference), "chars/kfm/kfm.def", "common")
	if err != nil {
		t.Fatal(err)
	}
	zss, err := LoadFighter(os.DirFS(reference), "chars/kfm_zss/kfm_zss.def", "common")
	if err != nil {
		t.Fatal(err)
	}
	if len(cns.Missing)+len(zss.Missing) != 0 {
		t.Fatalf("nothing should be missing, got %v and %v", cns.Missing, zss.Missing)
	}
	m := startMatch(t, cns, zss)
	requireStartsTheRound(t, m)
	requireTicksIdle(t, m)
}

func TestNewMatch_CommonFilesInTheCharacterFolderStartToo(t *testing.T) {
	cns, err := LoadFighter(copyChar(t, "kfm", "kfm.def", "common1.cns"))
	if err != nil {
		t.Fatal(err)
	}
	zss, err := LoadFighter(copyChar(t, "kfm_zss", "kfm_zss.def", "common1.cns.zss"))
	if err != nil {
		t.Fatal(err)
	}
	m := startMatch(t, cns, zss)
	requireStartsTheRound(t, m)
	requireTicksIdle(t, m)
}

func TestLoadFighter_CommonFoundNowhereIsReportedNotFatal(t *testing.T) {
	f, err := LoadFighter(copyChar(t, "kfm", "kfm.def"))
	if err != nil {
		t.Fatalf("a missing stcommon must not fail the load: %v", err)
	}
	if len(f.Missing) != 1 || f.Missing[0] != "common1.cns" {
		t.Fatalf("Missing = %v, want [common1.cns]", f.Missing)
	}
}

func TestNewMatch_StateZeroMissingErrorNamesTheMissingCommonFile(t *testing.T) {
	f, err := LoadFighter(copyChar(t, "kfm", "kfm.def"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewMatch(f, f, closeStage(), Setup{Rounds: 1})
	if err == nil {
		t.Fatal("want an error: state 0 lives only in the missing common file")
	}
	for _, want := range []string{"state 0 not found", "common1.cns"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

func TestCommonFolder_IsTheCommonFolderNextToTheContentWhateverItsCase(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Common"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := CommonFolder(os.DirFS(root)), "Common"; got != want {
		t.Errorf("CommonFolder = %q, want %q", got, want)
	}
}
