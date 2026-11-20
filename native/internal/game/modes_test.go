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
	src := strings.Replace(string(versusFile), `"timeLimitSeconds": 99`, `"timeLimitSeconds": `+secs, 1)
	if !strings.Contains(src, `"timeLimitSeconds": `+secs) {
		t.Fatal("test setup: time limit not found in versus.jsonc")
	}
	return []byte(src)
}

// modeWithID is the Versus mode file under another id and name.
func modeWithID(t *testing.T, id, name string) []byte {
	t.Helper()
	src := strings.Replace(string(versusWith(t, "99")), `"id": "versus"`, `"id": "`+id+`"`, 1)
	src = strings.Replace(src, `"name": "Versus"`, `"name": "`+name+`"`, 1)
	if !strings.Contains(src, `"id": "`+id+`"`) || !strings.Contains(src, `"name": "`+name+`"`) {
		t.Fatal("test setup: id or name not found in versus.jsonc")
	}
	return []byte(src)
}

func modeIDs(ms []mode.Mode) []string {
	var ids []string
	for _, m := range ms {
		ids = append(ids, m.ID)
	}
	return ids
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDiscoverModes_ListsEveryFileByNameIgnoringCase(t *testing.T) {
	fsys := fstest.MapFS{
		"modes/versus.jsonc":  {Data: versusWith(t, "45")},
		"modes/Arcade.jsonc":  {Data: modeWithID(t, "arcade", "Arcade")},
		"modes/training.json": {Data: modeWithID(t, "training", "Training")},
	}
	modes, notices := DiscoverModes(fsys)
	if want := []string{"arcade", "training", "versus"}; !sameStrings(modeIDs(modes), want) {
		t.Fatalf("ids = %v, want %v", modeIDs(modes), want)
	}
	if modes[2].Rules.TimeLimitSeconds != 45 || len(notices) != 0 {
		t.Fatalf("versus time = %d, notices = %v; want 45 from modes/ and no notice", modes[2].Rules.TimeLimitSeconds, notices)
	}
}

func TestDiscoverModes_FindsTheFolderAndExtensionWhateverTheirCase(t *testing.T) {
	fsys := fstest.MapFS{"Modes/Versus.JSONC": {Data: versusWith(t, "30")}}
	modes, notices := DiscoverModes(fsys)
	if len(modes) != 1 || modes[0].Rules.TimeLimitSeconds != 30 || len(notices) != 0 {
		t.Fatalf("modes = %v, notices = %v; want the one file read (30 s) and no notice", modeIDs(modes), notices)
	}
}

func TestDiscoverModes_MissingFolderGivesTheBuiltInVersusWithANotice(t *testing.T) {
	modes, notices := DiscoverModes(fstest.MapFS{"chars/.keep": {}})
	if len(modes) != 1 || modes[0].ID != "versus" || modes[0].Rules.TimeLimitSeconds != 99 {
		t.Fatalf("modes = %+v, want the built-in versus only", modes)
	}
	want := "modes/versus.jsonc not found or invalid, using the built-in Versus mode"
	if len(notices) != 1 || notices[0] != want {
		t.Fatalf("notices = %q, want [%q]", notices, want)
	}
}

func TestDiscoverModes_LeavesOutAnInvalidFileWithANoticeNamingItsField(t *testing.T) {
	broken := strings.Replace(string(versusWith(t, "60")), `"rounds": 3`, `"rounds": 2`, 1)
	fsys := fstest.MapFS{
		"modes/versus.jsonc": {Data: versusWith(t, "60")},
		"modes/broken.jsonc": {Data: []byte(broken)},
		"modes/arcade.jsonc": {Data: modeWithID(t, "arcade", "Arcade")},
	}
	modes, notices := DiscoverModes(fsys)
	if want := []string{"arcade", "versus"}; !sameStrings(modeIDs(modes), want) {
		t.Fatalf("ids = %v, want %v", modeIDs(modes), want)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "rules.rounds") || !strings.Contains(notices[0], "broken.jsonc") {
		t.Fatalf("notices = %q, want one naming broken.jsonc and rules.rounds", notices)
	}
}

func TestDiscoverModes_InvalidVersusFileFallsBackToTheBuiltInWithANotice(t *testing.T) {
	broken := strings.Replace(string(versusWith(t, "60")), `"rounds": 3`, `"rounds": 2`, 1)
	modes, notices := DiscoverModes(fstest.MapFS{"modes/versus.jsonc": {Data: []byte(broken)}})
	if len(modes) != 1 || modes[0].Rules.TimeLimitSeconds != 99 {
		t.Fatalf("modes = %+v, want the built-in versus", modes)
	}
	if len(notices) != 2 || !strings.Contains(notices[0], "rules.rounds") || !strings.Contains(notices[1], "using the built-in Versus mode") {
		t.Fatalf("notices = %q, want the field error then the fallback notice", notices)
	}
}

func TestDiscoverModes_KeepsTheFirstOfTwoFilesWithTheSameID(t *testing.T) {
	fsys := fstest.MapFS{
		"modes/a-versus.jsonc": {Data: versusWith(t, "10")},
		"modes/versus.jsonc":   {Data: versusWith(t, "20")},
	}
	modes, notices := DiscoverModes(fsys)
	if len(modes) != 1 || modes[0].Rules.TimeLimitSeconds != 10 {
		t.Fatalf("modes = %+v, want only the first file (10 s)", modes)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "versus.jsonc") || !strings.Contains(notices[0], `"versus"`) {
		t.Fatalf("notices = %q, want one naming the second file and the id", notices)
	}
}

func TestDiscoverModes_IgnoresOtherFilesAndFolders(t *testing.T) {
	fsys := fstest.MapFS{
		"modes/versus.jsonc":         {Data: versusWith(t, "99")},
		"modes/readme.txt":           {Data: []byte("not a mode")},
		"modes/old.jsonc/inner.json": {Data: []byte("{")},
	}
	modes, notices := DiscoverModes(fsys)
	if len(modes) != 1 || len(notices) != 0 {
		t.Fatalf("ids = %v, notices = %q; want versus only and no notice", modeIDs(modes), notices)
	}
}

func TestDiscoverModes_TheFixtureGameFolderShipsTheBuiltInFile(t *testing.T) {
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
