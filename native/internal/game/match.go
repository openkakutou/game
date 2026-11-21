package game

import (
	"bytes"
	"fmt"
	"github.com/openkakutou/character/air"
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
	// TrainingRefillTicks is how long a training fighter must go without a
	// hit to get its life back: 1 second.
	TrainingRefillTicks = TickRateHz
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
	// Training refills a fighter's life after a second without a hit and ends
	// no round (mode rule training).
	Training bool
	// MaxPowerMode starts round 1 with full power (-maxpowermode).
	MaxPowerMode bool
	// AILevel is each CPU side's level, 1 to 8 (Ikemen GO's sys.aiLevel),
	// given to its state machine for the AILevel trigger; 0 for a human side.
	AILevel [2]float64
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
	// LifeMax is each fighter's life, its character's [Data] life (Ikemen GO
	// lifeMax): the life a round starts with and the bars' full length.
	LifeMax [2]int
	// Sparks are the hit sparks being shown.
	Sparks Sparks
	// Over/Winner report that the match is decided.
	Over   bool
	Winner match.Side

	starting [2]match.FighterState
	// cam is the match's own camera, in stage units: Ikemen GO's camera is
	// part of the simulation and sets the screen edges the fighters stay
	// between (system.go:2871-2881). camX is its position, margins the
	// stage's screenleft and screenright.
	cam      StageCamera
	camX     float64
	margins  [2]float64
	programs [2]engine.FighterProgram
	missing  [2][]string // state files each character could not find
	runtimes [2]engine.FighterRuntime
	tick     int
	// Training life tracking, per side: the full life, the life seen last
	// tick, and the ticks it has stayed the same.
	maxHealth, lastHealth, quiet [2]int
	cues                         []Cue           // PlaySnd events not yet drained
	commonAnims                  []air.Animation // common fightfx, for the sparks
	fighterAnims                 [2][]air.Animation
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
	width := stageWidth(st)
	m.cam = NewStageCamera(st, CameraTension(st), 1, width)
	m.margins = screenMargins(st, width)
	for i, f := range []*Fighter{p1, p2} {
		m.programs[i] = f.Program
		m.missing[i] = f.Missing
		if f.Char != nil {
			m.fighterAnims[i] = f.Char.Animations
		}
		m.LifeMax[i] = f.Program.Constants.Lookup("data.life").Int()
		if m.LifeMax[i] <= 0 {
			m.LifeMax[i] = DefaultHealth
		}
		m.starting[i].Health = m.LifeMax[i]
	}
	if err := m.reset(1); err != nil {
		return nil, err
	}
	m.applyStart()
	m.trackLife()
	return m, nil
}

// ResetPositions puts both fighters back at their round-start positions, life
// and state, in the round being played; the score is kept.
func (m *Match) ResetPositions() error {
	err := m.reset(m.State.Round)
	if err != nil {
		return err
	}
	m.applyStart()
	m.trackLife()
	return nil
}

// trackLife takes the fighters' life now as their full life, for training.
func (m *Match) trackLife() {
	for i := range m.State.Fighters {
		m.maxHealth[i] = m.State.Fighters[i].Health
		m.lastHealth[i] = m.maxHealth[i]
		m.quiet[i] = 0
	}
}

// refillLife is a training tick's life rule: a fighter at zero life, or short
// of life and not hit for TrainingRefillTicks (its life unchanged), gets its
// full life back, in the state and in the state machine's context.
func (m *Match) refillLife() {
	for i := range m.State.Fighters {
		health := m.State.Fighters[i].Health
		if health != m.lastHealth[i] {
			m.lastHealth[i], m.quiet[i] = health, 0
		} else {
			m.quiet[i]++
		}
		if health > 0 && (health >= m.maxHealth[i] || m.quiet[i] < TrainingRefillTicks) {
			continue
		}
		m.State.Fighters[i].Health = m.maxHealth[i]
		m.runtimes[i].Context.Health = m.maxHealth[i]
		m.lastHealth[i], m.quiet[i] = m.maxHealth[i], 0
	}
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
		rt.Context.AILevel = m.Setup.AILevel[i]
		m.runtimes[i] = rt
	}
	m.State = *state
	m.tick = 0
	m.camX = m.cam.Follow(m.cam.Start(), m.starting[0].Position.X, m.starting[1].Position.X, true)
	m.cues = m.cues[:0]
	m.Sparks.Live = m.Sparks.Live[:0]
	m.LastRound = round.RoundResult{}
	m.syncAnim()
	return nil
}

// applyStart sets the life and power the setup overrides, in round 1 only,
// as Ikemen GO does (system.go:4417-4430, commit 54d4148a).
func (m *Match) applyStart() {
	for i, start := range m.Setup.Start {
		if start.Life > 0 {
			life := min(start.Life, m.LifeMax[i]) // Clamp(ocd.life, 0, lifeMax), system.go:4419
			m.State.Fighters[i].Health = life
			m.runtimes[i].Context.Health = life
		}
		power := start.Power
		if m.Setup.MaxPowerMode {
			power = MaxPower
		}
		if power > 0 {
			m.State.Fighters[i].Power = power
			m.runtimes[i].Context.Power = power
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
	cfg := engine.TickConfig{Bounds: &m.Bounds, Screen: m.Screen(), Gravity: DefaultGravity, Tick: m.tick, ComboWindow: DefaultComboWindow}
	res, err := engine.Tick(m.State, m.programs, m.runtimes, in, cfg)
	if err != nil {
		return err
	}
	m.State, m.runtimes = res.State, res.Fighters
	x0, x1 := m.State.Fighters[0].Position.X, m.State.Fighters[1].Position.X
	m.camX = m.cam.Follow(m.camX, min(x0, x1), max(x0, x1), false)
	m.syncAnim()
	m.queueCues(res.Sounds)
	m.Sparks.Step(m.commonAnims, m.fighterAnims)
	m.Sparks.Add(res.Events, [2]match.Facing{m.State.Fighters[0].Facing, m.State.Fighters[1].Facing})
	if m.Setup.Training {
		// A KO the engine reports is dropped: the life is back before anyone sees it.
		m.refillLife()
		return nil
	}
	if res.Round.Outcome != round.OutcomeNone {
		m.LastRound = res.Round
		m.Progress = m.Progress.RecordRoundResult(res.Round)
		m.Over, m.Winner = m.Progress.MatchOutcome()
	}
	return nil
}

// CameraX is the match camera's position, in stage units.
func (m *Match) CameraX() float64 { return m.camX }

// Screen is the left and right edges the fighters stay between: the
// camera's view less the stage's margins (Ikemen GO sys.xmin, sys.xmax,
// system.go:2871-2876), as of the last tick.
func (m *Match) Screen() [2]float64 {
	half := m.cam.halfWidth
	left, right := m.camX-half+m.margins[0], m.camX+half-m.margins[1]
	if left > right {
		left = (left + right) / 2
		right = left
	}
	return [2]float64{left, right}
}

// stageWidth is the stage's localcoord width, the width of the match's view
// in stage units.
func stageWidth(st *stage.Stage) float64 {
	if st.BGdef.LocalCoordWidth > 0 {
		return float64(st.BGdef.LocalCoordWidth)
	}
	return mugenLocalWidth
}

// defaultScreenMargin is Ikemen GO's screenleft and screenright default at a
// 320-wide localcoord, scaled to the stage's (stage.go:1149, 1310-1316).
const defaultScreenMargin = 15

func screenMargins(st *stage.Stage, width float64) [2]float64 {
	def := float64(int(defaultScreenMargin * width / mugenLocalWidth))
	margins := [2]float64{def, def}
	if l := st.StageBoundaries.ScreenLeft; l != nil {
		margins[0] = float64(*l)
	}
	if r := st.StageBoundaries.ScreenRight; r != nil {
		margins[1] = float64(*r)
	}
	return margins
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
