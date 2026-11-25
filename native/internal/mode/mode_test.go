package mode

import (
	"encoding/json"
	"errors"
	"os"
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

func TestParse_LadderFinal_ReadsTheBossFolderName(t *testing.T) {
	m, err := Parse("arcade.jsonc", []byte(strings.Replace(valid, `"count": 8`, `"count": 8, "final": "Boss"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if m.Rules.Opponents.Final != "Boss" || m.Rules.Opponents.Count != 8 {
		t.Errorf("opponents = %+v, want count 8 and final Boss", m.Rules.Opponents)
	}
	plain, err := Parse("arcade.jsonc", []byte(valid))
	if err != nil || plain.Rules.Opponents.Final != "" {
		t.Errorf("opponents = %+v (%v), want no final when the file has none", plain.Rules.Opponents, err)
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
		{"final that is not a string", `"count": 8`, `"count": 8, "final": 3`, "rules.opponents.final", "non-empty string"},
		{"empty final", `"count": 8`, `"count": 8, "final": ""`, "rules.opponents.final", "non-empty string"},
		{"final on chosen opponents", `"kind": "ladder", "count": 8`, `"kind": "chosen", "final": "boss"`, "rules.opponents.final", "ladder"},
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

func TestParse_NamesTheFileInErrors(t *testing.T) {
	_, err := Parse("modes/x.jsonc", []byte(strings.Replace(valid, `"rounds": 3`, `"rounds": 0`, 1)))
	if err == nil || !strings.Contains(err.Error(), "modes/x.jsonc") {
		t.Fatalf("err = %v, want one naming modes/x.jsonc", err)
	}
}

func TestParse_JSONCTrailingCommas_AreAccepted(t *testing.T) {
	src := strings.Replace(valid, `{"screen": "match"}]`, `{"screen": "match"},]`, 1)
	src = strings.Replace(src, `"onLose": "continue"}`, `"onLose": "continue",}`, 1)
	if _, err := Parse("arcade.jsonc", []byte(src)); err != nil {
		t.Fatal(err)
	}
}

func TestBuiltIn_ListsArcadeTrainingVersusThenWatchParsed(t *testing.T) {
	modes, err := BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	if len(modes) != 4 || modes[0].ID != "arcade" || modes[1].ID != "training" || modes[2].ID != "versus" || modes[3].ID != "watch" || modes[3].Name != "Watch" {
		t.Fatalf("modes = %+v, want arcade, training, versus then watch", modes)
	}
	w := modes[3]
	if w.Rules.Sides[0].Control != ControlCPU || w.Rules.Sides[1].Control != ControlCPU || len(w.Flow) != 6 || w.Flow[0].Screen != ScreenCharacterSelect {
		t.Errorf("watch = %+v, want two cpu sides and a 6-step flow starting with character select", w)
	}
}

func TestBuiltIn_ArcadeIsALadderOfSixAgainstTheCPUWithContinueAndVictory(t *testing.T) {
	modes, err := BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	a := modes[0]
	want := []Step{
		{Screen: ScreenCharacterSelect, Sides: []string{"p1"}}, {Screen: ScreenVersus}, {Screen: ScreenMatch},
		{Screen: ScreenContinue}, {Screen: ScreenVictory},
	}
	if a.Name != "Arcade" || !reflect.DeepEqual(a.Flow, want) {
		t.Errorf("arcade = %q, flow %+v, want Arcade and %+v", a.Name, a.Flow, want)
	}
	wantRules := Rules{
		Sides:  [2]Side{{Control: ControlHuman, Characters: 1}, {Control: ControlCPU, Characters: 1}},
		Rounds: 3, TimeLimitSeconds: 99,
		Opponents: Opponents{Kind: OpponentsLadder, Count: 6},
		OnWin:     OutcomeNextOpponent, OnLose: OutcomeContinue,
	}
	if !reflect.DeepEqual(a.Rules, wantRules) {
		t.Errorf("rules = %+v, want %+v", a.Rules, wantRules)
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

func TestMarshal_EmptyIncludeKeepsItsMeaningInJSON(t *testing.T) {
	out, err := json.Marshal(Filter{Include: []string{}, Exclude: nil})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"include":[]}` {
		t.Errorf("json = %s, want {\"include\":[]}", out)
	}
	out, err = json.Marshal(Filter{Exclude: []string{"a"}})
	if err != nil || string(out) != `{"exclude":["a"]}` {
		t.Errorf("json = %s (%v), want an absent include dropped", out, err)
	}
}

func TestBuiltIn_TrainingPicksBothFightersThenPlaysAMatchThatNeverEnds(t *testing.T) {
	modes, err := BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	tr := modes[1]
	want := []Step{{Screen: ScreenCharacterSelect, Sides: []string{"p1", "p2"}}, {Screen: ScreenStageSelect}, {Screen: ScreenMatch}}
	if tr.Name != "Training" || !reflect.DeepEqual(tr.Flow, want) {
		t.Errorf("training = %q, flow %+v, want Training and %+v", tr.Name, tr.Flow, want)
	}
	wantRules := Rules{
		Sides:  [2]Side{{Control: ControlHuman, Characters: 1}, {Control: ControlCPU, Characters: 1}},
		Rounds: 1, TimeLimitSeconds: 0,
		Opponents: Opponents{Kind: OpponentsChosen},
		OnWin:     OutcomeResults, OnLose: OutcomeResults,
		Training: true,
	}
	if !reflect.DeepEqual(tr.Rules, wantRules) {
		t.Errorf("rules = %+v, want %+v", tr.Rules, wantRules)
	}
	if !slices.Equal(tr.Pause, []PauseEntry{PauseResetPositions, PauseDummy}) {
		t.Errorf("pause = %v, want resetPositions then dummy", tr.Pause)
	}
}

func TestBuiltIn_OtherModesListNoPauseEntryAndAreNotTraining(t *testing.T) {
	modes, err := BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range modes {
		if m.ID == "training" {
			continue
		}
		if len(m.Pause) != 0 || m.Rules.Training {
			t.Errorf("%s: pause %v, training %v, want none", m.ID, m.Pause, m.Rules.Training)
		}
	}
}

func TestParse_PauseEntries_AreReadInTheOrderListed(t *testing.T) {
	src := strings.Replace(valid, `"flow"`, `"pause": ["dummy", "resetPositions"], "flow"`, 1)
	m, err := Parse("m.jsonc", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Pause, []PauseEntry{PauseDummy, PauseResetPositions}) {
		t.Errorf("pause = %v, want dummy then resetPositions", m.Pause)
	}
}

func TestParse_Training_IsReadAsABoolean(t *testing.T) {
	src := strings.Replace(valid, `"rounds": 3,`, `"rounds": 3, "training": true,`, 1)
	m, err := Parse("m.jsonc", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Rules.Training {
		t.Error("rules.training = false, want true")
	}
	plain, err := Parse("m.jsonc", []byte(valid))
	if err != nil || plain.Rules.Training {
		t.Errorf("without the field: training = %v, err = %v, want false", plain.Rules.Training, err)
	}
}

func TestParse_InvalidPauseOrTraining_ReturnsAnErrorNamingFileFieldAndReason(t *testing.T) {
	cases := []struct{ name, from, to, field, reason string }{
		{"unknown entry", `"flow"`, `"pause": ["continue"], "flow"`, "pause[0]", "not one of"},
		{"unknown entry after a good one", `"flow"`, `"pause": ["dummy", "nope"], "flow"`, "pause[1]", "not one of"},
		{"duplicate entry", `"flow"`, `"pause": ["dummy", "dummy"], "flow"`, "pause[1]", "twice"},
		{"pause not an array", `"flow"`, `"pause": "dummy", "flow"`, "pause", "array"},
		{"training not a boolean", `"rounds": 3,`, `"rounds": 3, "training": "yes",`, "rules.training", "boolean"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse("m.jsonc", []byte(strings.Replace(valid, c.from, c.to, 1)))
			var me *Error
			if !errors.As(err, &me) {
				t.Fatalf("err = %v, want a *mode.Error", err)
			}
			if me.File != "m.jsonc" || me.Field != c.field || !strings.Contains(me.Reason, c.reason) {
				t.Errorf("err = %+v, want file m.jsonc, field %s, reason containing %q", me, c.field, c.reason)
			}
		})
	}
}
