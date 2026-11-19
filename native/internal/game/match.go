package game

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"path"
	"strings"

	"github.com/openkakutou/character"
	"github.com/openkakutou/character/cmd"
	"github.com/openkakutou/engine/v2"
	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/engine/v2/round"
	"github.com/openkakutou/stage"
)

// Placeholder gameplay constants. The web build plays through the same
// core, so both targets simulate identically; web/hud keeps DefaultHealth
// and MaxPower as its bar maxima.
const (
	TickRateHz         = 60
	DefaultHealth      = 1000
	MaxPower           = 3000
	DefaultGravity     = 0.5
	DefaultComboWindow = 20
	FallbackHalfWidth  = 200
	StartingOffset     = 70
	UnlimitedTimer     = 999_999
)

// Fighter is a loaded character plus the file names the renderer needs.
type Fighter struct {
	Char *character.Character
	Cmd  cmd.CommandFile
	// SffPath is the name of its sprite file in the content file system.
	SffPath string
	// Program is the character's state table, loaded like Ikemen GO does.
	Program engine.FighterProgram
	// Missing lists the state files (st, stN, cmd, stcommon entries) found in
	// the character folder or the search folders: loading went on without them.
	Missing []string

	sounds map[soundKey]character.Sound // built lazily by FindSound
}

// LoadFighter loads a character from its .def, a name in fsys, including its parsed .cmd
// (a missing or broken .cmd falls back to an empty command file so the match
// still starts, as on web) and its state table: every .cns/.zss state file
// named by the .def, searched in the character folder then in searchDirs
// (folders of fsys, the game's common/ folder). Files found nowhere are listed in Missing and
// logged on stderr, they do not fail the load.
func LoadFighter(fsys fs.FS, defName string, searchDirs ...string) (*Fighter, error) {
	c, err := character.LoadFS(fsys, defName)
	if err != nil {
		return nil, err
	}
	program, missing, err := engine.LoadFighterProgramFS(fsys, c, defName, searchDirs...)
	if err != nil {
		return nil, fmt.Errorf("loading the states of %q: %w", defName, err)
	}
	if len(missing) > 0 {
		log.Printf("%s: state files not found: %s", defName, strings.Join(missing, ", "))
	}
	dir := path.Dir(defName)
	commands := cmd.CommandFile{Remap: map[string]string{}}
	if c.CommandFile != "" {
		if raw, err := fs.ReadFile(fsys, joinResolved(fsys, []string{dir}, c.CommandFile)); err == nil {
			if parsed, err := character.ParseCmd(raw); err == nil {
				commands = parsed
			}
		}
	}
	return &Fighter{Char: c, Cmd: commands, SffPath: joinResolved(fsys, []string{dir}, c.SpriteFile), Program: program, Missing: missing}, nil
}

// LoadStage parses a stage .def, a name in fsys.
func LoadStage(fsys fs.FS, defName string) (*stage.Stage, error) {
	src, err := fs.ReadFile(fsys, defName)
	if err != nil {
		return nil, fmt.Errorf("opening stage: %w", err)
	}
	s, err := stage.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("parsing stage %q: %w", defName, err)
	}
	return &s, nil
}

// Setup is the match setup screen's result. TimeLimitSeconds 0 means no limit.
type Setup struct {
	Rounds           int
	TimeLimitSeconds int
	// CPU makes the CPU opponent drive a side instead of its controls.
	CPU [2]bool
	// Start overrides each side's life and power at the start of the match.
	Start [2]StartValues
}

// StartValues overrides a fighter's life and power when the match starts
// (quick versus -p<n>.life, -p<n>.power); 0 keeps the default.
type StartValues struct {
	Life, Power int
}

// Validate rejects a setup the engine cannot play (rounds must be odd, >0).
func (s Setup) Validate() error {
	if s.Rounds < 1 || s.Rounds%2 == 0 {
		return fmt.Errorf("rounds must be a positive odd number, got %d", s.Rounds)
	}
	if s.TimeLimitSeconds < 0 {
		return fmt.Errorf("time limit must not be negative, got %d", s.TimeLimitSeconds)
	}
	return nil
}

// RoundTimerTicks converts the setup's time limit to simulation ticks.
func (s Setup) RoundTimerTicks() int {
	if s.TimeLimitSeconds == 0 {
		return UnlimitedTimer
	}
	return s.TimeLimitSeconds * TickRateHz
}

// ResolveBounds returns the stage's own boundaries when valid, else a
// symmetric fallback derived from its coordinate width.
func ResolveBounds(s *stage.Stage) stage.StageBoundaries {
	own := s.StageBoundaries
	if own.Left < own.Right {
		return own
	}
	half := FallbackHalfWidth
	if s.BGdef.LocalCoordWidth > 0 {
		half = s.BGdef.LocalCoordWidth / 2
	}
	own.Left, own.Right = -half, half
	return own
}

// StartingFighters places both fighters symmetrically facing each other,
// clamping the standard offset to narrow stages.
func StartingFighters(b stage.StageBoundaries) [2]match.FighterState {
	offset := float64(StartingOffset)
	if maxOff := float64(b.Right-b.Left) / 2; maxOff < offset {
		offset = maxOff
	}
	return [2]match.FighterState{
		match.SideP1: {Side: match.SideP1, Position: match.Position{X: -offset}, Facing: match.FacingRight, Health: DefaultHealth},
		match.SideP2: {Side: match.SideP2, Position: match.Position{X: offset}, Facing: match.FacingLeft, Health: DefaultHealth},
	}
}

// Match is one in-progress best-of-N match.
type Match struct {
	Setup    Setup
	Bounds   stage.StageBoundaries
	State    match.MatchState
	Progress round.Progress
	// AnimNo/AnimTime are each fighter's current animation and the ticks
	// elapsed in it (engine Context.AnimElapsed), indexed by side.
	AnimNo   [2]int
	AnimTime [2]int
	// LastRound is the outcome of the round that just ended (OutcomeNone while fighting).
	LastRound round.RoundResult
	// Over/Winner report that the match is decided.
	Over   bool
	Winner match.Side

	starting [2]match.FighterState
	programs [2]engine.FighterProgram
	missing  [2][]string // state files each character could not find
	runtimes [2]engine.FighterRuntime
	tick     int
	cues     []Cue // PlaySnd events not yet drained
}

// NewMatch assembles round 1 of a match.
func NewMatch(p1, p2 *Fighter, st *stage.Stage, setup Setup) (*Match, error) {
	if p1 == nil || p2 == nil || st == nil {
		return nil, fmt.Errorf("a match needs two fighters and a stage")
	}
	if err := setup.Validate(); err != nil {
		return nil, err
	}
	progress, err := round.NewProgress(setup.Rounds)
	if err != nil {
		return nil, err
	}
	bounds := ResolveBounds(st)
	m := &Match{Setup: setup, Bounds: bounds, Progress: progress, starting: StartingFighters(bounds)}
	for i, f := range []*Fighter{p1, p2} {
		m.programs[i] = f.Program
		m.missing[i] = f.Missing
	}
	if err := m.reset(1); err != nil {
		return nil, err
	}
	m.applyStart()
	return m, nil
}

func (m *Match) reset(roundNo int) error {
	state, err := round.ResetRound(roundNo, m.Setup.RoundTimerTicks(), m.starting[match.SideP1], m.starting[match.SideP2])
	if err != nil {
		return err
	}
	for i := range m.runtimes {
		rt, err := m.programs[i].NewRuntime(m.starting[i])
		if err != nil {
			return fmt.Errorf("side %d: %w%s", i, err, m.missingNote(i))
		}
		m.runtimes[i] = rt
	}
	m.State = *state
	m.tick = 0
	m.cues = m.cues[:0]
	m.LastRound = round.RoundResult{}
	m.syncAnim()
	return nil
}

// applyStart sets the life and power the setup overrides, in round 1 only,
// as Ikemen GO does (system.go:4417-4430, commit 54d4148a).
func (m *Match) applyStart() {
	for i, start := range m.Setup.Start {
		if start.Life > 0 {
			m.State.Fighters[i].Health = start.Life
			m.runtimes[i].Context.Health = start.Life
		}
		if start.Power > 0 {
			m.State.Fighters[i].Power = start.Power
			m.runtimes[i].Context.Power = start.Power
		}
	}
}

// missingNote names the state files side i's character could not find, to
// complete a "state not found" error with its likely cause.
func (m *Match) missingNote(i int) string {
	if len(m.missing[i]) == 0 {
		return ""
	}
	return " (state files not found: " + strings.Join(m.missing[i], ", ") + ")"
}

func (m *Match) syncAnim() {
	for i := range m.runtimes {
		m.AnimNo[i] = m.runtimes[i].Context.Anim
		m.AnimTime[i] = m.runtimes[i].Context.AnimElapsed
	}
}

// Step advances the simulation by one tick. After a round-ending tick,
// LastRound is set and the caller decides when to call NextRound.
func (m *Match) Step(in [2]input.TickInput) error {
	if m.Over {
		return fmt.Errorf("match is already over")
	}
	m.tick++
	cfg := engine.TickConfig{Bounds: &m.Bounds, Gravity: DefaultGravity, Tick: m.tick, ComboWindow: DefaultComboWindow}
	res, err := engine.Tick(m.State, m.programs, m.runtimes, in, cfg)
	if err != nil {
		return err
	}
	m.State, m.runtimes = res.State, res.Fighters
	m.syncAnim()
	m.queueCues(res.Sounds)
	if res.Round.Outcome != round.OutcomeNone {
		m.LastRound = res.Round
		m.Progress = m.Progress.RecordRoundResult(res.Round)
		m.Over, m.Winner = m.Progress.MatchOutcome()
	}
	return nil
}

// Ticks is how many ticks the current round has simulated.
func (m *Match) Ticks() int { return m.tick }

// RoundEnded reports that the current round has an outcome.
func (m *Match) RoundEnded() bool { return m.LastRound.Outcome != round.OutcomeNone }

// NextRound resets both fighters for the following round.
func (m *Match) NextRound() error {
	if m.Over {
		return fmt.Errorf("match is already over")
	}
	return m.reset(m.State.Round + 1)
}
