package game

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/mode"
)

// versusWith is the built-in Versus mode file with its time limit set to secs,
// so a test can tell a mode read from modes/ from the built-in one (99 s).
func versusWith(t *testing.T, secs string) []byte {
	t.Helper()
	versusFile, err := os.ReadFile("../mode/versus.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := mode.Versus()
	if err != nil || builtin.Rules.TimeLimitSeconds != 99 {
		t.Fatalf("built-in versus: %+v, %v", builtin.Rules, err)
	}
	src := strings.Replace(string(versusFile), `"timeLimitSeconds": 99`, `"timeLimitSeconds": `+secs, 1)
	if !strings.Contains(src, `"timeLimitSeconds": `+secs) {
		t.Fatal("test setup: time limit not found in versus.jsonc")
	}
	return []byte(src)
}

func TestVersusMode_ReadsModesFolderOfTheGame(t *testing.T) {
	fsys := fstest.MapFS{"modes/versus.jsonc": {Data: versusWith(t, "45")}}
	m, notices := VersusMode(fsys)
	if m.Rules.TimeLimitSeconds != 45 || len(notices) != 0 {
		t.Fatalf("time = %d, notices = %v; want 45 from modes/ and no notice", m.Rules.TimeLimitSeconds, notices)
	}
}

func TestVersusMode_FindsTheFileWhateverItsCase(t *testing.T) {
	fsys := fstest.MapFS{"Modes/Versus.JSONC": {Data: versusWith(t, "30")}}
	m, notices := VersusMode(fsys)
	if m.Rules.TimeLimitSeconds != 30 || len(notices) != 0 {
		t.Fatalf("time = %d, notices = %v; want 30 and no notice", m.Rules.TimeLimitSeconds, notices)
	}
}

func TestVersusMode_MissingFileFallsBackToBuiltInWithANotice(t *testing.T) {
	m, notices := VersusMode(fstest.MapFS{"chars/.keep": {}})
	if m.ID != "versus" || m.Rules.TimeLimitSeconds != 99 {
		t.Fatalf("mode = %+v, want the built-in versus", m)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "modes/versus.jsonc") {
		t.Fatalf("notices = %v, want one naming modes/versus.jsonc", notices)
	}
}

func TestVersusMode_InvalidFileFallsBackToBuiltInNamingTheError(t *testing.T) {
	broken := strings.Replace(string(versusWith(t, "60")), `"rounds": 3`, `"rounds": 2`, 1)
	m, notices := VersusMode(fstest.MapFS{"modes/versus.jsonc": {Data: []byte(broken)}})
	if m.Rules.Rounds != 3 || m.Rules.TimeLimitSeconds != 99 {
		t.Fatalf("rules = %+v, want the built-in versus", m.Rules)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "rules.rounds") {
		t.Fatalf("notices = %v, want one naming rules.rounds", notices)
	}
}

func TestVersusMode_TheFixtureGameFolderShipsTheBuiltInFile(t *testing.T) {
	shipped, err := os.ReadFile(content + "/modes/versus.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := os.ReadFile("../mode/versus.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	if string(shipped) != string(builtin) {
		t.Fatal("testdata/content/modes/versus.jsonc differs from internal/mode/versus.jsonc: copy it again")
	}
}
