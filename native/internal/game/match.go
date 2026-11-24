package game

import (
	"bytes"
	"fmt"
	"github.com/openkakutou/character/air"
	"github.com/openkakutou/sff"
	"io/fs"
	"log"
	"math"
	"path"
	"strings"

	"github.com/openkakutou/character"
	"github.com/openkakutou/character/cmd"
	"github.com/openkakutou/engine/v2"
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/engine/v2/round"
	"github.com/openkakutou/stage"
)

// Placeholder gameplay constants. The web build plays through the same
// core, so both targets simulate identically.
const (
	TickRateHz         = 60
	DefaultHealth      = 1000
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
	// Palette is the character's first .act palette, its default one (Ikemen
	// GO loadPalettes, char.go:4743-4800); nil without one, and the sprites
	// keep their own palettes.
	Palette *sff.Palette

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
	f := &Fighter{Char: c, Cmd: commands, SffPath: joinResolved(fsys, []string{dir}, c.SpriteFile), Program: program, Missing: missing}
	f.Palette = firstActPalette(fsys, dir, c.Palettes)
	return f, nil
}

// firstActPalette is the first of the character's .act palettes that reads,
// nil when none does (an unreadable one is skipped, as Ikemen GO does).
func firstActPalette(fsys fs.FS, dir string, palettes []string) *sff.Palette {
	for _, name := range palettes {
		raw, err := fs.ReadFile(fsys, joinResolved(fsys, []string{dir}, name))
		if err != nil {
			continue
		}
		pal, err := sff.DecodeExternalPalette(raw)
		if err != nil {
			continue
		}
		return &pal
	}
	return nil
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
	// Timings are the round's announcement times, the lifebar's [Round]
	// values; zero ones are Ikemen GO's defaults.
	Timings round.Timings
	// Slow is the KO slow motion, the lifebar's slow.time, slow.fadetime
	// and slow.speed; zero ones are Ikemen GO's defaults.
	Slow SlowMotion
}

// SlowMotion is the slow motion after a KO (fs.go:3303-3305): Time ticks at
// Speed, back to full speed over the last FadeTime.
type SlowMotion struct {
	Time, FadeTime int
	Speed          float64
}

// slowMotion is the setup's slow motion, Ikemen GO's defaults (60, 45,
// 0.25) for the unset values.
func (s Setup) slowMotion() SlowMotion {
	m := s.Slow
	if m.Time <= 0 {
		m.Time = 60
	}
	if m.FadeTime <= 0 {
		m.FadeTime = m.Time * 3 / 4
	}
	if m.Speed <= 0 || m.Speed > 1 {
		m.Speed = 0.25
	}
	return m
}

// roundTimings are the setup's timings, Ikemen GO's defaults for the unset
// ones (fs.go:3301-3310).
func (s Setup) roundTimings() round.Timings {
	t := s.Timings
	def := round.DefaultTimings
	for _, f := range []struct{ v, d *int }{
		{&t.StartWaitTime, &def.StartWaitTime}, {&t.CtrlTime, &def.CtrlTime},
		{&t.OverWaitTime, &def.OverWaitTime}, {&t.OverWinTime, &def.OverWinTime}, {&t.OverTime, &def.OverTime},
	} {
		if *f.v <= 0 {
			*f.v = *f.d
		}
	}
	return t
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
	// Anim is each fighter's current animation, indexed by side.
	Anim [2]Anim
	// LastRound is the outcome of the round that just ended (OutcomeNone while fighting).
	LastRound round.RoundResult
	// Wins are the rounds each side won, in order, for the win icons.
	Wins [2][]Win
	// LifeMax is each fighter's life, its character's [Data] life (Ikemen GO
	// lifeMax): the life a round starts with and the bars' full length.
	LifeMax [2]int
	// PowerMax is each fighter's power cap, its character's [Data] power
	// (Ikemen GO powerMax, engine Context.PowerMax): the power bars' full
	// length.
	PowerMax [2]int
	// Sparks are the hit sparks being shown.
	Sparks Sparks
	// Shake is the screen shake (EnvShake, FallEnvShake).
	Shake Shake
	// PalFX is each fighter's palette effect.
	PalFX [2]PalFX
	// AfterImages are each fighter's afterimage trail.
	AfterImages [2]AfterImage
	// Env is the EnvColor screen fill; StagePalFX tints the stage (BGPalFX,
	// AllPalFX), and AllPalFX the fighters too.
	Env        EnvColor
	StagePalFX PalFX
	AllPalFX   bool
	// Over/Winner report that the match is decided.
	Over   bool
	Winner match.Side

	// clock drives the round as Ikemen GO does (intros, fight, KO, poses),
	// nil in training, which fights from the first tick; progress is what
	// the fighters see of it, decided the round's result once known and
	// roundOver set when the clock ends the round.
	clock *round.Clock
	// slowLeft counts the slow motion's ticks down; slowAcc is how much of
	// a simulation tick the slowed frames have built up.
	slowLeft  int
	slowAcc   float64
	progress  engine.RoundProgress
	decided   round.RoundResult
	roundOver bool

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
	// The fighters move in a 320-wide space whatever the stage's localcoord
	// (Ikemen GO scales a stage's values by its localscl): the camera, the
	// screen edges and the stage bounds are taken into it.
	width := stageWidth(st)
	k := FighterScale(st)
	bounds := scaleBounds(ResolveBounds(st), k)
	m := &Match{Setup: setup, Bounds: bounds, Progress: progress, starting: StartingFighters(bounds)}
	m.cam = NewStageCamera(st, CameraTension(st), k, mugenLocalWidth)
	margins := screenMargins(st, width)
	m.margins = [2]float64{margins[0] * k, margins[1] * k}
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
	for i := range m.runtimes {
		m.PowerMax[i] = m.runtimes[i].Context.PowerMax()
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
	m.clock = nil
	if !m.Setup.Training {
		m.clock = round.NewClock(m.Setup.roundTimings())
	}
	for i := range m.runtimes {
		// A round starts in state 5900 without control, as in Ikemen GO; a
		// training session fights at once.
		var rt engine.FighterRuntime
		if m.clock != nil {
			rt, err = m.programs[i].NewRoundRuntime(m.starting[i])
		} else {
			rt, err = m.programs[i].NewRuntime(m.starting[i])
		}
		if err != nil {
			return fmt.Errorf("side %d: %w%s", i, err, m.missingNote(i))
		}
		rt.Context.AILevel = m.Setup.AILevel[i]
		m.runtimes[i] = rt
		state.Fighters[i].StateNo = rt.Context.StateNo
	}
	m.State = *state
	m.decided, m.roundOver = round.RoundResult{}, false
	m.slowLeft, m.slowAcc = 0, 0
	m.tick = 0
	m.camX = m.cam.Follow(m.cam.Start(), m.starting[0].Position.X, m.starting[1].Position.X, true)
	m.cues = m.cues[:0]
	m.Sparks.Live = m.Sparks.Live[:0]
	m.Shake = Shake{}
	m.PalFX = [2]PalFX{}
	m.AfterImages = [2]AfterImage{}
	m.Env, m.StagePalFX, m.AllPalFX = EnvColor{}, PalFX{}, false
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
		// Clamp(ocd.power, 0, powerMax), system.go:4426-4432.
		power := min(start.Power, m.PowerMax[i])
		if m.Setup.MaxPowerMode {
			power = m.PowerMax[i]
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
		ctx := &m.runtimes[i].Context
		owner := match.Side(i)
		if ctx.AnimFromOpponent {
			owner = match.Side(1 - i)
		}
		m.Anim[i] = Anim{No: ctx.Anim, Time: ctx.AnimElapsed, Owner: owner}
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
	if m.clock != nil && m.clock.InputWait() {
		in = [2]input.TickInput{} // the players' inputs wait (Ikemen GO inputWait)
	}
	if m.clock != nil {
		m.progress.State, m.progress.Result = m.clock.State(), m.decided
		m.progress.MatchOver = m.decided.Outcome != round.OutcomeNone && m.matchEndsWith(m.decided)
		cfg.Round = &m.progress
	}
	res, err := engine.Tick(m.State, m.programs, m.runtimes, in, cfg)
	if err != nil {
		return err
	}
	m.State, m.runtimes = res.State, res.Fighters
	if leftest, rightest, ok := m.cameraTargets(); ok {
		m.camX = m.cam.Follow(m.camX, leftest, rightest, false)
	}
	m.syncAnim()
	m.queueCues(res.Sounds)
	m.Sparks.Step(m.commonAnims, m.fighterAnims)
	for i := range m.PalFX {
		m.PalFX[i].Step()
	}
	m.Env.Step()
	m.StagePalFX.Step()
	if !m.StagePalFX.Active() {
		m.AllPalFX = false
	}
	for side, evs := range res.Events {
		for _, e := range evs {
			switch e.Type {
			case "EnvShake":
				m.Shake.Start(e)
			case "PalFX":
				m.PalFX[side].Start(e)
			case "AfterImage":
				m.AfterImages[side].Start(e)
			case "AfterImageTime":
				m.AfterImages[side].SetTime(e)
			case "EnvColor":
				m.Env.Start(e)
			case "BGPalFX":
				m.StagePalFX.Start(e)
				m.AllPalFX = false
			case "AllPalFX":
				m.StagePalFX.Start(e)
				m.AllPalFX = true
			case "StopSnd":
				m.queueStop(match.Side(side), e)
			}
		}
	}
	m.Shake.Step()
	for i := range m.AfterImages {
		m.AfterImages[i].Record(Ghost{Anim: m.Anim[i], Position: m.State.Fighters[i].Position, Facing: m.State.Fighters[i].Facing})
	}
	m.Sparks.Add(res.Events, [2]match.Facing{m.State.Fighters[0].Facing, m.State.Fighters[1].Facing})
	if m.Setup.Training {
		// A KO the engine reports is dropped: the life is back before anyone sees it.
		m.refillLife()
		return nil
	}
	if res.Round.Outcome != round.OutcomeNone && m.decided.Outcome == round.OutcomeNone {
		m.decided, m.LastRound = res.Round, res.Round
		// A KO (not a time over) slows the match down (system.go:2986-2997).
		if res.Round.Outcome == round.OutcomeKO || res.Round.Outcome == round.OutcomeDoubleKO {
			m.slowLeft = m.Setup.slowMotion().Time
		}
	}
	return m.stepClock()
}

// stepClock advances the round clock after a tick and does what it asks:
// start the fight, send the fighters to their poses, end the round (and
// count it).
func (m *Match) stepClock() error {
	intro := engine.IntroAsserted(&m.runtimes[0]) || engine.IntroAsserted(&m.runtimes[1])
	settled := engine.Settled(&m.runtimes[0]) && engine.Settled(&m.runtimes[1])
	switch m.clock.Step(intro, m.decided.Outcome != round.OutcomeNone, settled) {
	case round.EventFightStart:
		if err := engine.StartFight(m.programs, &m.State, &m.runtimes); err != nil {
			return err
		}
	case round.EventPoses:
		if err := engine.EnterPoses(m.programs, &m.State, &m.runtimes, m.decided); err != nil {
			return err
		}
	case round.EventRoundOver:
		m.roundOver = true
		m.Progress = m.Progress.RecordRoundResult(m.decided)
		m.recordWin(m.decided)
		m.Over, m.Winner = m.Progress.MatchOutcome()
	}
	m.syncAnim()
	return nil
}

// matchEndsWith reports whether the round result ends the match.
func (m *Match) matchEndsWith(r round.RoundResult) bool {
	over, _ := m.Progress.RecordRoundResult(r).MatchOutcome()
	return over
}

// Advances reports whether this frame steps the simulation: always, but in
// the KO slow motion, where frames build up its speed (slow.speed, eased back
// to full speed over the last slow.fadetime, system.go:2986-2997) and step
// once a whole tick has built up.
func (m *Match) Advances() bool {
	if m.slowLeft <= 0 {
		return true
	}
	slow := m.Setup.slowMotion()
	speed := slow.Speed
	if m.slowLeft < slow.FadeTime {
		speed += (1 - speed) * float64(slow.FadeTime-m.slowLeft) / float64(slow.FadeTime)
	}
	m.slowLeft--
	m.slowAcc += speed
	if m.slowAcc < 1 {
		return false
	}
	m.slowAcc--
	return true
}

// FighterContext is side's state machine context, for drawing (read only).
func (m *Match) FighterContext(side int) *evaluator.Context {
	return &m.runtimes[side].Context
}

// Trans is how side is drawn this tick: its Trans controller's type ("" for
// its animation's blend) and alpha.
func (m *Match) Trans(side int) (string, [2]int) {
	c := &m.runtimes[side].Context
	return c.Trans, c.TransAlpha
}

// Ghosts appends to dst side's afterimage ghosts to draw, newest first.
func (m *Match) Ghosts(side int, dst []GhostDraw) []GhostDraw {
	return m.AfterImages[side].Ghosts(dst, m.programs[side].IkemenVersion == [3]uint16{})
}

// cameraTargets are the leftmost and rightmost fighters the camera follows:
// those a ScreenBound has not taken off it (Ikemen GO CSF_movecamera_x,
// char.go:12826); ok is false when it follows none.
func (m *Match) cameraTargets() (leftest, rightest float64, ok bool) {
	for i := range m.runtimes {
		if !m.runtimes[i].Context.MoveCamera[0] {
			continue
		}
		x := m.State.Fighters[i].Position.X
		if !ok {
			leftest, rightest, ok = x, x, true
			continue
		}
		leftest, rightest = min(leftest, x), max(rightest, x)
	}
	return leftest, rightest, ok
}

// Intro is the round clock's Ikemen GO sys.intro, and whether the match has
// a round clock (training has none).
func (m *Match) Intro() (int, bool) {
	if m.clock == nil {
		return 0, false
	}
	return m.clock.Intro(), true
}

// DecidedResult is the round's result so far.
func (m *Match) DecidedResult() round.RoundResult { return m.decided }

// RoundState is the round's Ikemen GO RoundState (0 to 4); 2 in training.
func (m *Match) RoundState() int {
	if m.clock == nil {
		return 2
	}
	return m.clock.State()
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

// FighterScale is how many fighter units a stage unit is: 320 over the
// stage's localcoord width (1 for a MUGEN 320-wide stage, 0.25 for a
// 1280-wide one).
func FighterScale(st *stage.Stage) float64 {
	return mugenLocalWidth / stageWidth(st)
}

// scaleBounds is b in fighter units.
func scaleBounds(b stage.StageBoundaries, k float64) stage.StageBoundaries {
	b.Left, b.Right = int(math.Round(float64(b.Left)*k)), int(math.Round(float64(b.Right)*k))
	return b
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

// RoundEnded reports that the current round is over: decided, its poses
// played and its over.time passed (training rounds end at the KO).
func (m *Match) RoundEnded() bool { return m.roundOver }

// NextRound resets both fighters for the following round.
func (m *Match) NextRound() error {
	if m.Over {
		return fmt.Errorf("match is already over")
	}
	return m.reset(m.State.Round + 1)
}
