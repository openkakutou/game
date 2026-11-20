package core

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkakutou/game/native/internal/game"
)

const content = "../../testdata/content"

// loaded is a Core with the whole testdata content mounted.
func loaded(t *testing.T) *Core {
	t.Helper()
	c := New()
	err := filepath.WalkDir(content, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(content, p)
		if err != nil {
			return err
		}
		if got := decode(t, c.Mount(filepath.ToSlash(rel), data)); got["ok"] != true {
			t.Fatalf("mount %s: %v", rel, got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The fixture characters cannot walk, so matches are played on a stage
	// narrow enough to start the fighters within reach of each other.
	arena, err := os.ReadFile(filepath.Join(content, "stages", "arena.def"))
	if err != nil {
		t.Fatal(err)
	}
	close := strings.NewReplacer("leftbound = -160", "leftbound = -6", "rightbound = 160", "rightbound = 6")
	c.Mount("stages/close.def", []byte(close.Replace(string(arena))))
	return c
}

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("not JSON: %q: %v", s, err)
	}
	return out
}

func reason(t *testing.T, s string) string {
	t.Helper()
	out := decode(t, s)
	if out["ok"] != false {
		t.Fatalf("want an error envelope, got %s", s)
	}
	return out["error"].(map[string]any)["reason"].(string)
}

const startReq = `{"p1":"chars/fighter/fighter.def","p2":"chars/rival/rival.def","stage":"stages/close.def",
"setup":{"rounds":3,"timeLimitSeconds":60},"viewWidth":320}`

func start(t *testing.T, c *Core, req string) (id int, out map[string]any) {
	t.Helper()
	out = decode(t, c.StartMatch(req))
	if out["ok"] != true {
		t.Fatalf("start: %v", out)
	}
	return int(out["id"].(float64)), out
}

const idle = `{"p1":{},"p2":{}}`
const walkRight = `{"p1":{"right":true},"p2":{}}`
const pressA = `{"p1":{"buttons":{"a":true}},"p2":{}}`

func TestMount_RejectsInvalidNames(t *testing.T) {
	c := New()
	for _, name := range []string{"", "/abs/x", "../x", "a//b", "a\\b"} {
		if got := reason(t, c.Mount(name, []byte("x"))); !strings.Contains(got, "name") {
			t.Errorf("Mount(%q) reason = %q, want it to name the problem", name, got)
		}
	}
}

func TestStartMatch_ReportsBadRequestsAsErrorEnvelopes(t *testing.T) {
	c := loaded(t)
	cases := map[string]struct{ req, want string }{
		"not json":      {`{`, "invalid request"},
		"unknown key":   {`{"p3":"x"}`, "invalid request"},
		"even rounds":   {strings.Replace(startReq, `"rounds":3`, `"rounds":2`, 1), "rounds must be a positive odd number"},
		"no view":       {strings.Replace(startReq, `"viewWidth":320`, `"viewWidth":0`, 1), "viewWidth"},
		"missing char":  {strings.Replace(startReq, "fighter/fighter.def", "nobody/nobody.def", 1), "nobody/nobody.def"},
		"missing map":   {strings.Replace(startReq, "close.def", "nowhere.def", 1), "nowhere.def"},
		"trailing data": {startReq + ` {"p1":"x"}`, "invalid request"},
	}
	for name, tc := range cases {
		if got := reason(t, c.StartMatch(tc.req)); !strings.Contains(got, tc.want) {
			t.Errorf("%s: reason = %q, want it to contain %q", name, got, tc.want)
		}
	}
}

func TestTickAndNextRoundAndClose_ReportUnknownIds(t *testing.T) {
	c := loaded(t)
	if got := reason(t, c.Tick(7, idle)); !strings.Contains(got, "no match 7") {
		t.Errorf("Tick: %q", got)
	}
	if got := reason(t, c.NextRound(7)); !strings.Contains(got, "no match 7") {
		t.Errorf("NextRound: %q", got)
	}
	if got := reason(t, c.CloseMatch(7)); !strings.Contains(got, "no match 7") {
		t.Errorf("CloseMatch: %q", got)
	}
}

func TestTick_RejectsBadInputs(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, startReq)
	for _, in := range []string{``, `{`, `{"p1":{"up":1}}`, `{"p3":{}}`, `{"p1":{"buttons":{"q":true}}}`, idle + `x`} {
		if got := reason(t, c.Tick(id, in)); !strings.Contains(got, "inputs") {
			t.Errorf("Tick(%q) reason = %q, want it to mention the inputs", in, got)
		}
	}
}

func TestCloseMatch_ForgetsTheMatch(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, startReq)
	if got := decode(t, c.CloseMatch(id)); got["ok"] != true {
		t.Fatalf("close: %v", got)
	}
	if got := reason(t, c.Tick(id, idle)); !strings.Contains(got, "no match") {
		t.Errorf("Tick after close: %q", got)
	}
}

func TestStartMatch_ReturnsRoundZeroState(t *testing.T) {
	c := loaded(t)
	_, out := start(t, c, startReq)
	// Pinned from the setup: 3 rounds -> 2 to win; 60 s -> 3600 ticks;
	// the stage is 6 units either side of the origin, so the fighters start
	// there, full health, no power.
	want := map[string]any{"roundsToWin": 2.0, "roundTimerTicks": 3600.0, "tick": 0.0, "round": 1.0, "timer": 3600.0,
		"roundEnded": false, "over": false, "winner": -1.0, "roundWinner": -1.0, "outcome": "none"}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("%s = %v, want %v", k, out[k], v)
		}
	}
	fighters := out["fighters"].([]any)
	f0, f1 := fighters[0].(map[string]any), fighters[1].(map[string]any)
	if f0["x"] != -6.0 || f1["x"] != 6.0 || f0["facing"] != 1.0 || f1["facing"] != -1.0 ||
		f0["health"] != 1000.0 || f1["health"] != 1000.0 || f0["power"] != 0.0 {
		t.Errorf("fighters = %v", fighters)
	}
}

// playTo has P1 tap attack for n ticks.
func playTo(t *testing.T, c *Core, id, n int) map[string]any {
	t.Helper()
	var out map[string]any
	for i := 0; i < n; i++ {
		in := pressA
		if i%2 == 1 {
			in = idle
		}
		out = decode(t, c.Tick(id, in))
		if out["ok"] != true {
			t.Fatalf("tick %d: %v", i, out)
		}
	}
	return out
}

func TestTick_SnapshotMatchesTheMatchItDrives(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, startReq)
	out := playTo(t, c, id, 300)
	m := c.matches[id].match

	if out["tick"] != float64(m.Ticks()) || out["timer"] != float64(m.State.RoundTimer) || out["round"] != float64(m.State.Round) {
		t.Errorf("tick/timer/round = %v/%v/%v, match has %d/%d/%d", out["tick"], out["timer"], out["round"], m.Ticks(), m.State.RoundTimer, m.State.Round)
	}
	for i, raw := range out["fighters"].([]any) {
		f := raw.(map[string]any)
		s := m.State.Fighters[i]
		if f["x"] != s.Position.X || f["y"] != s.Position.Y || f["health"] != float64(s.Health) ||
			f["power"] != float64(s.Power) || f["stateNo"] != float64(s.StateNo) ||
			f["anim"] != float64(m.AnimNo[i]) || f["animTime"] != float64(m.AnimTime[i]) {
			t.Errorf("fighter %d = %v, match has %+v anim %d/%d", i, f, s, m.AnimNo[i], m.AnimTime[i])
		}
	}
	// Pinned: 300 ticks of walking then attacking hurt P2 and not P1.
	f := out["fighters"].([]any)
	if f[1].(map[string]any)["health"].(float64) >= game.DefaultHealth || f[0].(map[string]any)["health"] != 1000.0 {
		t.Errorf("health after the attack = %v", f)
	}
	if out["camX"] != c.matches[id].camX {
		t.Errorf("camX = %v, session has %v", out["camX"], c.matches[id].camX)
	}
}

func TestTick_DrainsSoundCuesOnce(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, startReq)
	// The fixture's attack state (200) plays sound 5,0 on its first tick.
	var cues []any
	for i := 0; i < 5 && len(cues) == 0; i++ {
		cues = decode(t, c.Tick(id, pressA))["cues"].([]any)
	}
	if len(cues) != 1 {
		t.Fatalf("cues = %v, want one", cues)
	}
	cue := cues[0].(map[string]any)
	if cue["side"] != 0.0 || cue["group"] != 5.0 || cue["sample"] != 0.0 {
		t.Errorf("cue = %v, want side 0 sound 5,0", cue)
	}
	if again := decode(t, c.Tick(id, idle))["cues"].([]any); len(again) != 0 {
		t.Errorf("cues were not drained: %v", again)
	}
}

func TestTick_TheCPUDrivesPlayer2(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, strings.Replace(startReq, `"viewWidth"`, `"p2cpu":true,"viewWidth"`, 1))
	if s := c.matches[id]; s.cpu == nil || s.cpu.Level != game.DefaultAILevel {
		t.Fatalf("p2cpu: CPU = %+v, want one at Ikemen GO's default level", s.cpu)
	}
	// A draw that always wins jams every button: P2 presses a and attacks.
	c.matches[id].cpu = game.NewCPU(game.DefaultAILevel, func(lo, _ int) int { return lo })
	attacked := false
	for i := 0; i < 300 && !attacked; i++ {
		out := decode(t, c.Tick(id, idle))
		attacked = out["fighters"].([]any)[1].(map[string]any)["stateNo"] == 200.0
	}
	if !attacked {
		t.Error("the CPU never made P2 attack")
	}
}

func TestNextRound_RefusesARoundStillBeingPlayed(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, startReq)
	c.Tick(id, idle)
	if got := reason(t, c.NextRound(id)); !strings.Contains(got, "round is not over") {
		t.Errorf("NextRound during round 1: %q", got)
	}
	if out := decode(t, c.Tick(id, idle)); out["round"] != 1.0 {
		t.Errorf("round after a refused NextRound = %v, want 1", out["round"])
	}
}

func TestTick_NamesTheRoundWinnerOfATimeoutWithAHealthLead(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, strings.Replace(startReq, `"timeLimitSeconds":60`, `"timeLimitSeconds":2`, 1))
	out := playTo(t, c, id, 120)
	// Pinned: P1's attack lands within the first seconds (health of P2 drops,
	// see TestTick_SnapshotMatchesTheMatchItDrives), so P1 wins on time.
	if out["outcome"] != "timeout" || out["roundWinner"] != 0.0 {
		t.Fatalf("after the 2 s limit: outcome %v, roundWinner %v, fighters %v", out["outcome"], out["roundWinner"], out["fighters"])
	}
}

func TestTick_RoundEndsAndNextRoundResets(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, strings.Replace(startReq, `"timeLimitSeconds":60`, `"timeLimitSeconds":1`, 1))
	var out map[string]any
	for i := 0; i < 60; i++ {
		out = decode(t, c.Tick(id, idle))
	}
	if out["roundEnded"] != true || out["outcome"] != "draw" || out["timer"] != 0.0 || out["roundWinner"] != -1.0 {
		t.Fatalf("after the 1 s limit: %v", out)
	}
	if got := reason(t, c.Tick(id, idle)); !strings.Contains(got, "nextRound") {
		t.Errorf("tick after the round ended: %q", got)
	}
	next := decode(t, c.NextRound(id))
	if next["ok"] != true || next["round"] != 2.0 || next["tick"] != 0.0 || next["roundEnded"] != false || next["timer"] != 60.0 {
		t.Errorf("nextRound = %v", next)
	}
}

// The tick path must not allocate beyond its result string: the engine's own
// step is measured on a twin match, and the layer around it adds at most the
// one string.
func TestTick_AllocatesOnlyTheResultString(t *testing.T) {
	c := loaded(t)
	id, _ := start(t, c, startReq)
	twin, _ := start(t, c, startReq)
	for i := 0; i < 5; i++ { // warm the reused buffers
		c.Tick(id, walkRight)
	}
	for i := 0; i < 5; i++ {
		c.Tick(twin, walkRight)
	}
	s := c.matches[twin]
	engineOnly := testing.AllocsPerRun(50, func() {
		_ = s.match.Step(s.in)
	})
	whole := testing.AllocsPerRun(50, func() {
		c.Tick(id, walkRight)
	})
	t.Logf("allocs: Tick %v, engine step alone %v", whole, engineOnly)
	if whole > engineOnly+1 {
		t.Errorf("Tick allocates %v per call, the engine step alone %v: the layer adds more than the result string", whole, engineOnly)
	}
}
