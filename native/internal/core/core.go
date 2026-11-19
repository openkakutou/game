// Package core is the match API the web build exposes to JavaScript (see
// cmd/core-wasm): content mounted as bytes into an in-memory file system,
// matches kept here in a registry and stepped one tick at a time. Every call
// takes and returns JSON strings (or plain values) and never panics or
// returns an error: a failure is the envelope {"ok":false,"error":{"reason"}}.
// It has no syscall/js dependency, so it is testable natively.
package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing/fstest"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/game/native/internal/game"
)

// Core holds the mounted content and the running matches.
type Core struct {
	files   fstest.MapFS
	matches map[int]*session
	nextID  int
}

// session is one running match and the buffers its ticks reuse.
type session struct {
	match *game.Match
	cam   game.StageCamera
	camX  float64
	cpu   *game.CPU // drives player 2 when set
	// in is the inputs of the tick being stepped; the button maps are reused.
	in   [2]input.TickInput
	cues []game.Cue
	buf  []byte // the JSON result under construction
	// roundTimerTicks and roundsToWin are what the HUD needs once per match.
	roundTimerTicks, roundsToWin int
}

// New returns a Core with no content and no match.
func New() *Core {
	return &Core{files: fstest.MapFS{}, matches: map[int]*session{}}
}

func fail(format string, args ...any) string {
	b, err := json.Marshal(map[string]any{"ok": false, "error": map[string]string{"reason": fmt.Sprintf(format, args...)}})
	if err != nil {
		return `{"ok":false,"error":{"reason":"cannot encode the error"}}`
	}
	return string(b)
}

const okReply = `{"ok":true}`

// Mount adds a file to the content, under name, a slash-separated path
// relative to the game root ("chars/kfm/kfm.def"); mounting a name again
// replaces the file. data is kept, not copied.
func (c *Core) Mount(name string, data []byte) string {
	if !fs.ValidPath(name) || name == "." || strings.Contains(name, `\`) {
		return fail("invalid file name %q: want a slash-separated path under the game root", name)
	}
	c.files[name] = &fstest.MapFile{Data: data}
	return okReply
}

type startRequest struct {
	P1    string `json:"p1"`
	P2    string `json:"p2"`
	Stage string `json:"stage"`
	Setup struct {
		Rounds           int `json:"rounds"`
		TimeLimitSeconds int `json:"timeLimitSeconds"`
	} `json:"setup"`
	P2CPU bool `json:"p2cpu"`
	// ViewWidth is the width the camera frames, in pixels.
	ViewWidth float64 `json:"viewWidth"`
}

// StartMatch loads the two characters and the stage (names of .def files
// mounted earlier) and starts round 1 of a match. It replies with the match
// id, what the HUD needs once (roundTimerTicks, roundsToWin) and the state
// to draw before the first tick.
func (c *Core) StartMatch(req string) string {
	var r startRequest
	dec := json.NewDecoder(strings.NewReader(req))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return fail("invalid request: %v", err)
	}
	if dec.More() {
		return fail("invalid request: unexpected data after the object")
	}
	if !(r.ViewWidth > 0) {
		return fail("invalid request: viewWidth must be positive, got %v", r.ViewWidth)
	}
	loader := game.DefaultLoader(c.files, game.CommonFolder(c.files))
	var fighters [2]*game.Fighter
	for i, def := range []string{r.P1, r.P2} {
		f, err := loader.Fighter(def)
		if err != nil {
			return fail("cannot load character %q: %v", def, err)
		}
		fighters[i] = f
	}
	st, err := loader.Stage(r.Stage)
	if err != nil {
		return fail("cannot load stage %q: %v", r.Stage, err)
	}
	setup := game.Setup{Rounds: r.Setup.Rounds, TimeLimitSeconds: r.Setup.TimeLimitSeconds, CPU: [2]bool{false, r.P2CPU}}
	m, err := game.NewMatch(fighters[0], fighters[1], st, setup)
	if err != nil {
		return fail("cannot start the match: %v", err)
	}
	s := &session{match: m, buf: make([]byte, 0, 1024)}
	s.cam = game.NewStageCamera(st, game.CameraTension(st), game.StageScale(st, r.ViewWidth), r.ViewWidth)
	s.roundTimerTicks, s.roundsToWin = setup.RoundTimerTicks(), m.Progress.RoundsToWin()
	if r.P2CPU {
		s.cpu = game.NewCPU(nil)
	}
	for i := range s.in {
		s.in[i].Buttons = make(map[string]bool, len(buttonNames))
	}
	s.followCamera(true)
	c.nextID++
	c.matches[c.nextID] = s
	return s.snapshot(c.nextID)
}

// NextRound starts the round after one that ended, and replies with the state
// to draw before its first tick.
func (c *Core) NextRound(id int) string {
	s, ok := c.matches[id]
	if !ok {
		return fail("no match %d", id)
	}
	if !s.match.RoundEnded() {
		return fail("the round is not over")
	}
	if err := s.match.NextRound(); err != nil {
		return fail("%v", err)
	}
	s.followCamera(true)
	return s.snapshot(0)
}

// CloseMatch forgets a match and frees its memory.
func (c *Core) CloseMatch(id int) string {
	if _, ok := c.matches[id]; !ok {
		return fail("no match %d", id)
	}
	delete(c.matches, id)
	return okReply
}

// followCamera moves the camera to frame both fighters; snap jumps straight
// there, as at the start of a round.
func (s *session) followCamera(snap bool) {
	x0 := s.match.State.Fighters[0].Position.X
	x1 := s.match.State.Fighters[1].Position.X
	if snap {
		s.camX = s.cam.Start()
	}
	s.camX = s.cam.Follow(s.camX, min(x0, x1), max(x0, x1), snap)
}

// Failure is the error envelope for a call the JavaScript glue rejected
// before reaching the core (wrong argument types).
func Failure(reason string) string { return fail("%s", reason) }
