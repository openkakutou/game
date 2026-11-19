package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

// Quick versus follows Ikemen GO's command line (commit 54d4148a):
// processCommandLine (src/main.go:241-361) for the parsing, main.f_commandLine
// (external/script/main.lua:1023-1256) for the match.

func TestParseArgs_TwoLeadingNamesAreP1AndP2(t *testing.T) {
	cl := ParseArgs([]string{"kfm", "kfm", "-s", "Training", "-rounds", "3"})
	want := map[string]string{"-p1": "kfm", "-p2": "kfm", "-s": "Training", "-rounds": "3"}
	if !mapsEqual(cl.Flags, want) {
		t.Errorf("flags = %v, want %v", cl.Flags, want)
	}
}

func TestParseArgs_NamesAfterAFlagAreNotPlayers(t *testing.T) {
	// main.go:351: positional names count only before the first flag.
	cl := ParseArgs([]string{"-nosound", "kfm", "kfm"})
	if _, ok := cl.Flags["-p1"]; ok {
		t.Errorf("flags = %v: a name after a flag must not become -p1", cl.Flags)
	}
	if cl.Flags["-nosound"] != "true" {
		t.Errorf("-nosound = %q, want true (a boolean flag takes no value)", cl.Flags["-nosound"])
	}
}

func TestParseArgs_NegativeNumberIsAValueAndALastFlagWithoutValueIsTrue(t *testing.T) {
	cl := ParseArgs([]string{"-p1", "kfm", "-time", "-1", "-p2", "kfm", "-p2.ai", "4", "-loadmotif"})
	if cl.Flags["-time"] != "-1" || cl.Flags["-p2.ai"] != "4" || cl.Flags["-loadmotif"] != "true" {
		t.Errorf("flags = %v", cl.Flags)
	}
}

func TestParseArgs_PowerShellSplitPlayerOptionIsJoined(t *testing.T) {
	// main.go:244-259: PowerShell may pass "-p1.ai" as "-p1" ".ai".
	cl := ParseArgs([]string{"kfm", "kfm", "-p1", ".ai", "3"})
	if cl.Flags["-p1.ai"] != "3" {
		t.Errorf("flags = %v, want -p1.ai 3", cl.Flags)
	}
}

func TestParseArgs_HelpFlags(t *testing.T) {
	for _, h := range []string{"-h", "-?", "-%"} {
		if !ParseArgs([]string{"kfm", h}).Help {
			t.Errorf("%s: Help = false", h)
		}
	}
	if ParseArgs([]string{"-height", "720"}).Help {
		t.Error("-height is not help")
	}
}

func TestQuickVersusFrom_NeedsBothPlayers(t *testing.T) {
	_, ok, _, err := QuickVersusFrom(ParseArgs([]string{"-p1", "kfm"}))
	if ok || err != nil {
		t.Errorf("ok, err = %v, %v, want false, nil: one player starts the menus", ok, err)
	}
	_, ok, _, _ = QuickVersusFrom(ParseArgs([]string{"kfm", "kfm", "-loadmotif"}))
	if ok {
		t.Error("-loadmotif: quick versus starts only without it (main.lua:1255)")
	}
}

func TestQuickVersusFrom_DefaultsAndOptions(t *testing.T) {
	qv, ok, notices, err := QuickVersusFrom(ParseArgs([]string{"kfm", "Ryu", "-s", "Training", "-rounds", "3", "-time", "60",
		"-p2.ai", "8", "-p1.life", "500", "-p2.power", "3000"}))
	if err != nil || !ok {
		t.Fatalf("ok, err = %v, %v", ok, err)
	}
	if qv.P1 != "kfm" || qv.P2 != "Ryu" || qv.Stage != "Training" {
		t.Errorf("qv = %+v", qv)
	}
	s := qv.Setup()
	// -rounds is the rounds to win (main.lua:1110-1115, setMatchWins): 3 wins = best of 5.
	if s.Rounds != 5 || s.TimeLimitSeconds != 60 || !s.CPU[1] || s.CPU[0] {
		t.Errorf("setup = %+v", s)
	}
	if s.Start[0].Life != 500 || s.Start[1].Power != 3000 || s.Start[1].Life != 0 {
		t.Errorf("start = %+v", s.Start)
	}
	if !slices.ContainsFunc(notices, func(n string) bool { return strings.Contains(n, "-p2.ai") && strings.Contains(n, "level") }) {
		t.Errorf("notices = %v, want one saying the AI level is not used", notices)
	}

	qv, _, _, _ = QuickVersusFrom(ParseArgs([]string{"kfm", "kfm"}))
	if s := qv.Setup(); s.Rounds != 3 || s.TimeLimitSeconds != 99 || s.CPU != [2]bool{} {
		t.Errorf("defaults = %+v, want best of 3 (2 wins), 99 s, both human (Ikemen GO's defaults)", s)
	}
	qv, _, _, _ = QuickVersusFrom(ParseArgs([]string{"kfm", "kfm", "-time", "-1"}))
	if qv.Setup().TimeLimitSeconds != 0 {
		t.Error("-time -1: want no time limit")
	}
}

func TestQuickVersusFrom_UnsupportedOptionsAreNamedNotFatal(t *testing.T) {
	_, ok, notices, err := QuickVersusFrom(ParseArgs([]string{"kfm", "kfm", "-p3", "kfm", "-tmode1", "1", "-p1.color", "3", "-zzz", "1"}))
	if !ok || err != nil {
		t.Fatalf("ok, err = %v, %v", ok, err)
	}
	for _, flag := range []string{"-p3", "-tmode1", "-p1.color", "-zzz"} {
		if !slices.ContainsFunc(notices, func(n string) bool { return strings.Contains(n, flag) }) {
			t.Errorf("notices = %v, want one naming %s", notices, flag)
		}
	}
}

func TestQuickVersusFrom_InvalidValuesAreErrorsNamingTheOption(t *testing.T) {
	for _, args := range [][]string{
		{"-rounds", "0"}, {"-rounds", "x"}, {"-time", "0"}, {"-time", "-2"},
		{"-p1.ai", "-1"}, {"-p1.life", "0"}, {"-p2.life", "1001"}, {"-p1.power", "3001"},
	} {
		_, _, _, err := QuickVersusFrom(ParseArgs(append([]string{"kfm", "kfm"}, args...)))
		if err == nil || !strings.Contains(err.Error(), args[0]) {
			t.Errorf("%v: err = %v, want one naming %s", args, err, args[0])
		}
	}
}

// contentTree builds root/chars/kfm/kfm.def, root/chars/Ryu/Ryu Groove.def,
// root/stages/Training.def, root/stages/stage0.def.
func contentTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range []string{"chars/kfm/kfm.def", "chars/Ryu/Ryu Groove.def", "stages/Training.def", "stages/stage0.def", "stages/Dojo/Dojo.def"} {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestResolveCharDef_FollowsIkemenGOsSearch(t *testing.T) {
	root := contentTree(t)
	cases := map[string]string{
		"kfm":                        "chars/kfm/kfm.def", // folder name (system.go:5414-5422)
		"KFM":                        "chars/kfm/kfm.def", // case-insensitive, as the rest of the content
		"Ryu/Ryu Groove.def":         "chars/Ryu/Ryu Groove.def",
		"Ryu/Ryu Groove":             "chars/Ryu/Ryu Groove.def", // a path without .def tries the file first
		"chars/kfm/kfm.def":          "chars/kfm/kfm.def",
		`Ryu\Ryu Groove.def`:         "chars/Ryu/Ryu Groove.def",
		filepath.Join(root, "chars"): "",
	}
	for decl, want := range cases {
		got, err := ResolveCharDef(root, decl)
		if want == "" {
			if err == nil {
				t.Errorf("%q: got %q, want an error", decl, got)
			}
			continue
		}
		if err != nil || got != filepath.Join(root, filepath.FromSlash(want)) {
			t.Errorf("%q: got %q, %v, want %s", decl, got, err, want)
		}
	}
	abs := filepath.Join(root, "chars", "kfm", "kfm.def")
	if got, err := ResolveCharDef(root, abs); err != nil || got != abs {
		t.Errorf("absolute path: got %q, %v", got, err)
	}
	if _, err := ResolveCharDef(root, "nobody"); err == nil || !strings.Contains(err.Error(), "nobody") {
		t.Errorf("missing: err = %v, want one naming it", err)
	}
}

func TestResolveStageDef_FollowsIkemenGOsSearch(t *testing.T) {
	root := contentTree(t)
	for decl, want := range map[string]string{
		"Training":            "stages/Training.def", // system.go:5476-5480
		"training.def":        "stages/Training.def",
		"Dojo":                "stages/Dojo/Dojo.def",
		"stages/Training.def": "stages/Training.def",
	} {
		got, err := ResolveStageDef(root, decl)
		if err != nil || got != filepath.Join(root, filepath.FromSlash(want)) {
			t.Errorf("%q: got %q, %v, want %s", decl, got, err, want)
		}
	}
	if _, err := ResolveStageDef(root, "nowhere"); err == nil {
		t.Error("missing stage: want an error")
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func quickFlow(t *testing.T) *Flow {
	t.Helper()
	stages, _ := DiscoverStages(filepath.Join(content, "stages"))
	return NewFlow(nil, stages, DefaultLoader())
}

func TestStartQuickVersus_StartsTheMatchWithoutMenus(t *testing.T) {
	f := quickFlow(t)
	q, _, _, _ := QuickVersusFrom(ParseArgs([]string{"fighter", "rival", "-rounds", "1", "-time", "-1", "-p2.ai", "1", "-p1.life", "300"}))
	notices := f.StartQuickVersus(q, content)
	if f.Screen != ScreenMatch || f.Match == nil {
		t.Fatalf("screen = %v (%s), want the match", f.Screen, f.Message)
	}
	if len(notices) != 0 {
		t.Errorf("notices = %v", notices)
	}
	if s := f.Match.Setup; s.Rounds != 1 || s.TimeLimitSeconds != 0 || f.cpus[1] == nil || f.cpus[0] != nil {
		t.Errorf("setup = %+v, cpus = %v", s, f.cpus)
	}
	if h := f.Match.State.Fighters[0].Health; h != 300 {
		t.Errorf("P1 life = %d, want 300", h)
	}
}

func TestStartQuickVersus_QuitsWhenTheMatchEndsOrThePlayerLeaves(t *testing.T) {
	f := quickFlow(t)
	q, _, _, _ := QuickVersusFrom(ParseArgs([]string{"fighter", "rival", "-rounds", "1"}))
	f.StartQuickVersus(q, content)
	f.Match.Over = true
	f.Match.LastRound.Outcome = 1
	f.RoundOverTicks = 1
	f.Tick([2]input.TickInput{})
	if !f.Quit || f.Screen == ScreenResult {
		t.Errorf("Quit, screen = %v, %v, want the game to quit after the match (main.lua:1252)", f.Quit, f.Screen)
	}

	f = quickFlow(t)
	f.StartQuickVersus(q, content)
	f.Navigate(NavCancel) // pause menu: Continue, Restart, Exit
	f.Navigate(NavUp)     // wraps to Exit
	f.Navigate(NavConfirm)
	if !f.Quit {
		t.Error("Exit from a quick versus match must quit, there is no menu to go back to")
	}
}

func TestStartQuickVersus_UnknownStageFallsBackWithANoticeAndUnknownCharacterIsAnError(t *testing.T) {
	f := quickFlow(t)
	q, _, _, _ := QuickVersusFrom(ParseArgs([]string{"fighter", "rival", "-s", "nowhere"}))
	notices := f.StartQuickVersus(q, content)
	if f.Screen != ScreenMatch || len(notices) != 1 || !strings.Contains(notices[0], "nowhere") {
		t.Errorf("screen, notices = %v, %v, want the match on the default stage and a notice naming nowhere", f.Screen, notices)
	}

	f = quickFlow(t)
	q, _, _, _ = QuickVersusFrom(ParseArgs([]string{"fighter", "nobody"}))
	f.StartQuickVersus(q, content)
	if f.Screen != ScreenError || !strings.Contains(f.Message, "nobody") {
		t.Errorf("screen, message = %v, %q, want an error naming nobody", f.Screen, f.Message)
	}
}
