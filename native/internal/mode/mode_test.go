package mode

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// valid is a minimal valid mode the error tests break one field at a time.
const valid = `{
  "formatVersion": 1, "id": "arcade", "name": "Arcade",
  "flow": [{"screen": "characterSelect", "params": {"sides": ["p1"]}}, {"screen": "match"}],
  "rules": {"sides": [{"control": "human", "characters": 1}, {"control": "cpu", "characters": 1}],
            "rounds": 3, "timeLimitSeconds": 0,
            "opponents": {"kind": "ladder", "count": 8}, "onWin": "nextOpponent", "onLose": "continue"}
}`

func TestParse_VersusExample_LoadsEveryField(t *testing.T) {
	m, err := Parse("versus.jsonc", mustRead(t, "versus.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "versus" || m.Name != "Versus" || len(m.Flow) != 6 {
		t.Fatalf("mode = %+v", m)
	}
	if m.Flow[0].Screen != ScreenCharacterSelect || strings.Join(m.Flow[0].Sides, ",") != "p1,p2" {
		t.Errorf("flow[0] = %+v", m.Flow[0])
	}
	r := m.Rules
	if r.Rounds != 3 || r.TimeLimitSeconds != 99 || r.Sides[1].Control != ControlHumanOrCPU || r.Sides[0].Characters != 1 {
		t.Errorf("rules = %+v", r)
	}
	if r.Opponents.Kind != OpponentsChosen || r.OnWin != OutcomeResults || r.OnLose != OutcomeResults {
		t.Errorf("rules = %+v", r)
	}
}

func TestParse_JSONCComments_AreIgnoredButNotInsideStrings(t *testing.T) {
	src := strings.Replace(valid, `"name": "Arcade"`, `/* block */ "name": "Arcade // not a comment", // line`, 1)
	m, err := Parse("arcade.jsonc", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Arcade // not a comment" {
		t.Errorf("name = %q", m.Name)
	}
}

func TestParse_LadderOpponents_ReadsTheCount(t *testing.T) {
	m, err := Parse("arcade.jsonc", []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if m.Rules.Opponents.Kind != OpponentsLadder || m.Rules.Opponents.Count != 8 || m.Rules.TimeLimitSeconds != 0 {
		t.Errorf("rules = %+v", m.Rules)
	}
}

func TestParse_InvalidMode_ReturnsAnErrorNamingFileFieldAndReason(t *testing.T) {
	cases := []struct {
		name, from, to, field, reason string
	}{
		{"even rounds", `"rounds": 3`, `"rounds": 2`, "rules.rounds", "odd"},
		{"unknown screen", `"screen": "match"`, `"screen": "minigame"`, "flow[1].screen", "minigame"},
		{"no match screen", `, {"screen": "match"}`, ``, "flow", "match"},
		{"unknown field", `"id": "arcade"`, `"id": "arcade", "script": "x.lua"`, "script", "unknown"},
		{"missing name", `"name": "Arcade",`, ``, "name", "required"},
		{"wrong type", `"rounds": 3`, `"rounds": "three"`, "rules.rounds", "integer"},
		{"one side", `, {"control": "cpu", "characters": 1}`, ``, "rules.sides", "2"},
		{"bad control", `"control": "cpu"`, `"control": "robot"`, "rules.sides[1].control", "robot"},
		{"ladder without count", `"kind": "ladder", "count": 8`, `"kind": "ladder"`, "rules.opponents.count", "required"},
		{"future format", `"formatVersion": 1`, `"formatVersion": 2`, "formatVersion", "1"},
		{"bad side param", `"sides": ["p1"]}`, `"sides": ["p3"]}`, "flow[0].params.sides[0]", "p3"},
		{"results before the match", `{"screen": "match"}]`, `{"screen": "results"}, {"screen": "match"}]`, "flow[1].screen", "after"},
		{"character select without sides", `{"screen": "characterSelect", "params": {"sides": ["p1"]}}`, `{"screen": "characterSelect"}`, "flow[0].params.sides", "required"},
		{"param on a screen without params", `{"screen": "match"}`, `{"screen": "match", "params": {"x": 1}}`, "flow[1].params.x", "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := strings.Replace(valid, c.from, c.to, 1)
			if src == valid {
				t.Fatalf("test setup: %q not found", c.from)
			}
			_, err := Parse("arcade.jsonc", []byte(src))
			var me *Error
			if !errors.As(err, &me) {
				t.Fatalf("err = %v, want a *mode.Error", err)
			}
			if me.File != "arcade.jsonc" || me.Field != c.field || !strings.Contains(me.Reason, c.reason) {
				t.Errorf("err = %+v, want file arcade.jsonc, field %s, reason containing %q", me, c.field, c.reason)
			}
			if !strings.Contains(err.Error(), "arcade.jsonc") || !strings.Contains(err.Error(), c.field) {
				t.Errorf("message %q does not name file and field", err.Error())
			}
		})
	}
}

func TestParse_NotJSON_ReturnsAnErrorNamingTheFile(t *testing.T) {
	_, err := Parse("broken.jsonc", []byte(`{"id": `))
	var me *Error
	if !errors.As(err, &me) || me.File != "broken.jsonc" {
		t.Fatalf("err = %v", err)
	}
}

func TestLoad_ReadsAFileAndNamesItInErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.jsonc")
	if err := os.WriteFile(path, []byte(strings.Replace(valid, `"rounds": 3`, `"rounds": 0`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("err = %v, want one naming %s", err, path)
	}
	if _, err := Load(filepath.Join(dir, "missing.jsonc")); err == nil {
		t.Fatal("missing file: want an error")
	}
}

func TestParse_JSONCTrailingCommas_AreAccepted(t *testing.T) {
	src := strings.Replace(valid, `{"screen": "match"}]`, `{"screen": "match"},]`, 1)
	src = strings.Replace(src, `"onLose": "continue"}`, `"onLose": "continue",}`, 1)
	if _, err := Parse("arcade.jsonc", []byte(src)); err != nil {
		t.Fatal(err)
	}
}

func TestBuiltIn_ListsVersusThenWatchParsed(t *testing.T) {
	modes, err := BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	if len(modes) != 2 || modes[0].ID != "versus" || modes[1].ID != "watch" || modes[1].Name != "Watch" {
		t.Fatalf("modes = %+v, want versus then watch", modes)
	}
	w := modes[1]
	if w.Rules.Sides[0].Control != ControlCPU || w.Rules.Sides[1].Control != ControlCPU || len(w.Flow) != 6 || w.Flow[0].Screen != ScreenCharacterSelect {
		t.Errorf("watch = %+v, want two cpu sides and a 6-step flow starting with character select", w)
	}
}

// arcadeRoster is an arcade-like roster rule: a boss that cannot be picked
// but is fought, and a stage that cannot be picked.
const arcadeRoster = `"roster": {
  "characters": {"select": {"exclude": ["Boss"]}, "opponents": {"include": ["kfm", "BOSS"], "exclude": ["kfm"]}},
  "stages": {"select": {"include": ["stage0"]}}
}, `

func withRoster(roster string) string {
	return strings.Replace(valid, `"rounds": 3`, roster+`"rounds": 3`, 1)
}

func TestParse_RosterFilter_ReadsEveryList(t *testing.T) {
	m, err := Parse("arcade.jsonc", []byte(withRoster(arcadeRoster)))
	if err != nil {
		t.Fatal(err)
	}
	r := m.Rules.Roster
	if !slices.Equal(r.Characters.Select.Exclude, []string{"Boss"}) || r.Characters.Select.Include != nil {
		t.Errorf("characters.select = %+v, want exclude [Boss] and no include", r.Characters.Select)
	}
	if !slices.Equal(r.Characters.Opponents.Include, []string{"kfm", "BOSS"}) || !slices.Equal(r.Characters.Opponents.Exclude, []string{"kfm"}) {
		t.Errorf("characters.opponents = %+v", r.Characters.Opponents)
	}
	if !slices.Equal(r.Stages.Select.Include, []string{"stage0"}) || r.Stages.Select.Exclude != nil {
		t.Errorf("stages.select = %+v", r.Stages.Select)
	}
}

func TestParse_NoRoster_LeavesEveryFilterAbsent(t *testing.T) {
	m, err := Parse("arcade.jsonc", []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Rules.Roster, Roster{}) {
		t.Errorf("roster = %+v, want the zero Roster", m.Rules.Roster)
	}
}

func TestParse_EmptyIncludeArray_IsKeptApartFromAnAbsentInclude(t *testing.T) {
	m, err := Parse("arcade.jsonc", []byte(withRoster(`"roster": {"stages": {"select": {"include": []}}}, `)))
	if err != nil {
		t.Fatal(err)
	}
	inc := m.Rules.Roster.Stages.Select.Include
	if inc == nil || len(inc) != 0 {
		t.Errorf("include = %#v, want an empty non-nil list (shows nothing)", inc)
	}
}

func TestParse_InvalidRoster_ReturnsAnErrorNamingFileFieldAndReason(t *testing.T) {
	cases := []struct{ name, roster, field, reason string }{
		{"roster not an object", `"roster": []`, "rules.roster", "object"},
		{"unknown roster key", `"roster": {"fighters": {}}`, "rules.roster.fighters", "unknown"},
		{"unknown characters key", `"roster": {"characters": {"watch": {}}}`, "rules.roster.characters.watch", "unknown"},
		{"stages have no opponents", `"roster": {"stages": {"opponents": {}}}`, "rules.roster.stages.opponents", "unknown"},
		{"unknown filter key", `"roster": {"characters": {"select": {"allow": []}}}`, "rules.roster.characters.select.allow", "unknown"},
		{"include not an array", `"roster": {"characters": {"select": {"include": "kfm"}}}`, "rules.roster.characters.select.include", "array"},
		{"exclude item not a string", `"roster": {"characters": {"opponents": {"exclude": ["a", 3]}}}`, "rules.roster.characters.opponents.exclude[1]", "non-empty string"},
		{"empty name", `"roster": {"stages": {"select": {"exclude": [""]}}}`, "rules.roster.stages.select.exclude[0]", "non-empty string"},
		{"select not an object", `"roster": {"stages": {"select": ["a"]}}`, "rules.roster.stages.select", "object"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse("arcade.jsonc", []byte(withRoster(c.roster+", ")))
			var me *Error
			if !errors.As(err, &me) {
				t.Fatalf("err = %v, want a *mode.Error", err)
			}
			if me.File != "arcade.jsonc" || me.Field != c.field || !strings.Contains(me.Reason, c.reason) {
				t.Errorf("err = %+v, want file arcade.jsonc, field %s, reason containing %q", me, c.field, c.reason)
			}
		})
	}
}

func TestFilter_Allows(t *testing.T) {
	cases := []struct {
		name   string
		filter Filter
		of     string
		want   bool
	}{
		{"no lists allow everything", Filter{}, "kfm", true},
		{"include allows a listed name", Filter{Include: []string{"kfm"}}, "kfm", true},
		{"include refuses an unlisted name", Filter{Include: []string{"kfm"}}, "boss", false},
		{"empty include refuses everything", Filter{Include: []string{}}, "kfm", false},
		{"exclude refuses a listed name", Filter{Exclude: []string{"boss"}}, "boss", false},
		{"exclude allows the others", Filter{Exclude: []string{"boss"}}, "kfm", true},
		{"exclude wins over include", Filter{Include: []string{"kfm", "boss"}, Exclude: []string{"boss"}}, "boss", false},
		{"names match ignoring case", Filter{Include: []string{"KFM"}}, "kfm", true},
		{"exclusion matches ignoring case", Filter{Exclude: []string{"Boss"}}, "BOSS", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.filter.Allows(c.of); got != c.want {
				t.Errorf("%+v.Allows(%q) = %v, want %v", c.filter, c.of, got, c.want)
			}
		})
	}
}
