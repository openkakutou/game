package game

import (
	"github.com/openkakutou/character/air"
	"github.com/openkakutou/engine/v2/round"
	sharedlifebar "github.com/openkakutou/lifebar"
)

// Call is a lifebar announcement.
type Call int

const (
	CallNone Call = iota
	CallRound
	CallFight
	CallKO
	CallDKO
	CallTO
	CallWin
	CallDraw
)

// Announcer paces the lifebar's round, fight and KO calls as Ikemen GO's
// FightScreenRound does (fs.go:3563-3900): the round call ctrl.time into the
// intro, round.time after it starts, until its animation or display time ends;
// the fight call callfight.time after the round call shows; once the round
// is decided, the KO, double KO or time over call, its time (plus MUGEN's 10
// ticks) after, shown at the game's speed rather than the slow motion's.
// Each call's sound plays at its sndtime.
// The win or draw call follows over.waittime after the decision.
// shortcut: round.single / round.final, win2-4, ai.win / ai.lose, the
// per-win-type calls and the bg and top layers are not played; MUGEN's 10-tick KO delay is
// always applied (the lifebar library does not report ikemenversion).
type Announcer struct {
	roundPhase, roundTimer int
	fightPhase, fightTimer int
	koPhase, koTimer       int
	winPhase, winTimer     int
	outro                  bool
	result                 round.RoundResult
	// lastIntro is the round clock's intro at the last tick.
	lastIntro int
	// sounds are the calls' sounds due, drained by DrainSounds.
	sounds []sharedlifebar.SoundRef
}

// queueSound queues a call's sound, none for a negative group.
func (a *Announcer) queueSound(ref sharedlifebar.SoundRef) {
	if ref.Group >= 0 {
		a.sounds = append(a.sounds, ref)
	}
}

// DrainSounds returns dst followed by the calls' sounds due since the last
// drain, and forgets them.
func (a *Announcer) DrainSounds(dst []sharedlifebar.SoundRef) []sharedlifebar.SoundRef {
	dst = append(dst, a.sounds...)
	a.sounds = a.sounds[:0]
	return dst
}

// Result is the round result the win call announces.
func (a *Announcer) Result() round.RoundResult { return a.result }

// mugenKODelay is the delay Ikemen GO adds to a MUGEN lifebar's KO call.
const mugenKODelay = 10

// Shown is a call on screen and how many ticks its animation has played.
type Shown struct {
	Call    Call
	Elapsed int
}

// Showing is the calls on screen, round, fight, KO then win, CallNone in
// the slots of those not showing; Ikemen GO draws them together.
func (a *Announcer) Showing(lb *sharedlifebar.Lifebar) [4]Shown {
	r := &lb.Round
	var out [4]Shown
	if a.roundPhase == 1 {
		out[0] = Shown{CallRound, max(a.roundTimer-r.RoundTime-1, 0)}
	}
	if a.fightPhase == 1 {
		out[1] = Shown{CallFight, max(a.fightTimer-r.FightTime-1, 0)}
	}
	if a.koPhase == 1 {
		out[2] = Shown{koCall(a.result.Outcome), max(a.koTimer-KOCallTime(lb, a.result.Outcome)-mugenKODelay-1, 0)}
	}
	if call := winCall(lb, a.result); a.winPhase == 1 && call != CallNone {
		out[3] = Shown{call, max(a.winTimer-r.WinTime-1, 0)}
	}
	return out
}

func koCall(o round.Outcome) Call {
	switch o {
	case round.OutcomeDoubleKO:
		return CallDKO
	case round.OutcomeTimeout:
		return CallTO
	}
	return CallKO
}

// Tick advances the calls by a simulated tick: intro is the round clock's,
// finish the round's outcome so far.
func (a *Announcer) Tick(lb *sharedlifebar.Lifebar, roundNo, intro int, result round.RoundResult) {
	r := &lb.Round
	finish := result.Outcome
	a.lastIntro = intro
	if intro > r.CtrlTime {
		*a = Announcer{}
		return
	}
	if a.roundPhase < 2 || a.fightPhase < 2 {
		a.intro(lb, roundNo)
	}
	if a.fightPhase == 2 && intro < 0 && finish != round.OutcomeNone && !a.outro {
		a.outro, a.result = true, result
		a.stepOutro(lb)
	}
}

// Frame advances the KO call by a rendered frame, slow motion or not.
func (a *Announcer) Frame(lb *sharedlifebar.Lifebar) {
	if a.outro {
		a.stepOutro(lb)
	}
}

func (a *Announcer) intro(lb *sharedlifebar.Lifebar, roundNo int) {
	r := &lb.Round
	if a.roundPhase < 2 {
		if a.roundTimer == r.RoundSndTime {
			a.queueSound(roundSound(lb, roundNo))
		}
		if a.roundTimer >= r.RoundTime {
			a.roundPhase = 1
			if ended(lb, RoundCall(lb, roundNo), a.roundTimer-r.RoundTime+1, true) {
				a.roundPhase = 2
			}
		}
		a.roundTimer++
	}
	switch a.fightPhase {
	case 0:
		if a.roundPhase == 0 {
			return
		}
		if a.fightTimer >= r.CallFightTime {
			a.fightPhase, a.fightTimer = 1, 0
			return
		}
		a.fightTimer++
	case 1:
		if a.fightTimer == r.FightSndTime {
			a.queueSound(r.Fight.Sound)
		}
		if a.fightTimer >= r.FightTime && ended(lb, r.Fight, a.fightTimer-r.FightTime+1, true) && a.fightTimer >= r.FightSndTime {
			a.fightPhase = 2
		}
		a.fightTimer++
	}
}

func (a *Announcer) stepOutro(lb *sharedlifebar.Lifebar) {
	a.stepKO(lb)
	a.stepWin(lb)
}

// stepWin paces the win or draw call: over.waittime after the decision, then
// win.time, its sound at win.sndtime (fs.go handleRoundOutro, 3880-3960).
func (a *Announcer) stepWin(lb *sharedlifebar.Lifebar) {
	r := &lb.Round
	if a.winPhase >= 2 || a.lastIntro >= -r.OverWaitTime {
		return
	}
	call := winCall(lb, a.result)
	if call == CallNone {
		return
	}
	c := CallOf(lb, call, 0, a.result)
	if a.winTimer == r.WinSndTime {
		a.queueSound(c.Sound)
	}
	if a.winTimer >= r.WinTime {
		a.winPhase = 1
		if ended(lb, c, a.winTimer-r.WinTime+1, false) {
			a.winPhase = 2
		}
	}
	a.winTimer++
}

// winCall is the call a round result ends on: draw for a time over draw or
// a double KO with dko.showdraw, the winner's win call, none for a double KO
// without.
func winCall(lb *sharedlifebar.Lifebar, r round.RoundResult) Call {
	switch r.Outcome {
	case round.OutcomeTimeoutDraw:
		return CallDraw
	case round.OutcomeDoubleKO:
		if lb.Round.DkoShowDraw {
			return CallDraw
		}
		return CallNone
	case round.OutcomeKO, round.OutcomeTimeout:
		return CallWin
	}
	return CallNone
}

func (a *Announcer) stepKO(lb *sharedlifebar.Lifebar) {
	if a.koPhase >= 2 {
		return
	}
	start := KOCallTime(lb, a.result.Outcome) + mugenKODelay
	if a.koTimer == koSoundTime(lb, a.result.Outcome)+mugenKODelay {
		a.queueSound(KOCall(lb, a.result.Outcome).Sound)
	}
	if a.koTimer >= start {
		a.koPhase = 1
		if ended(lb, KOCall(lb, a.result.Outcome), a.koTimer-start+1, false) {
			a.koPhase = 2
		}
	}
	a.koTimer++
}

// RoundCall is the round call of round roundNo: its own when the lifebar
// draws one, else round.default.
func RoundCall(lb *sharedlifebar.Lifebar, roundNo int) sharedlifebar.Announcement {
	if roundNo >= 1 && roundNo <= len(lb.Round.Round) && drawable(lb.Round.Round[roundNo-1]) {
		return lb.Round.Round[roundNo-1]
	}
	return lb.Round.RoundDefault
}

// roundSound is round roundNo's call sound: its own when it has one, else
// round.default's (fs.go handleRoundIntro).
func roundSound(lb *sharedlifebar.Lifebar, roundNo int) sharedlifebar.SoundRef {
	if roundNo >= 1 && roundNo <= len(lb.Round.Round) && lb.Round.Round[roundNo-1].Sound.Group >= 0 {
		return lb.Round.Round[roundNo-1].Sound
	}
	return lb.Round.RoundDefault.Sound
}

// koSoundTime is when the call of a round decided with o sounds.
func koSoundTime(lb *sharedlifebar.Lifebar, o round.Outcome) int {
	switch koCall(o) {
	case CallDKO:
		return lb.Round.DkoSndTime
	case CallTO:
		return lb.Round.ToSndTime
	}
	return lb.Round.KoSndTime
}

// KOCall is the call of a round decided with o.
func KOCall(lb *sharedlifebar.Lifebar, o round.Outcome) sharedlifebar.Announcement {
	switch koCall(o) {
	case CallDKO:
		return lb.Round.Dko
	case CallTO:
		return lb.Round.To
	}
	return lb.Round.Ko
}

// KOCallTime is when that call starts after the decision.
func KOCallTime(lb *sharedlifebar.Lifebar, o round.Outcome) int {
	switch koCall(o) {
	case CallDKO:
		return lb.Round.DkoTime
	case CallTO:
		return lb.Round.ToTime
	}
	return lb.Round.KoTime
}

// drawable reports a call with something to show.
func drawable(c sharedlifebar.Announcement) bool {
	return c.Element.HasAnim || c.Element.HasSprite || c.Text.Text != ""
}

// ended is Ikemen GO's AnimTextSnd.End (common.go:1718-1737), checked dt
// ticks in, before that tick's animation step: past its display time when
// it has one, else once its animation looped, or (inf) reached a last frame
// held for ever.
func ended(lb *sharedlifebar.Lifebar, c sharedlifebar.Announcement, dt int, inf bool) bool {
	if c.DisplayTime >= 0 {
		return dt >= c.DisplayTime
	}
	frames := callFrames(lb, c.Element)
	if len(frames) == 0 {
		return true
	}
	before := 0
	for _, fr := range frames[:len(frames)-1] {
		before += max(fr.Time, 0)
	}
	last := frames[len(frames)-1].Time
	if last < 0 {
		return inf && dt > before
	}
	return dt > before+last
}

// callFrames is the animation an element plays: its action, else its sprite
// held for ever.
func callFrames(lb *sharedlifebar.Lifebar, e sharedlifebar.Element) []air.Frame {
	if e.HasAnim {
		return lb.Animations[e.Anim].Frames
	}
	if e.HasSprite {
		return []air.Frame{{Group: e.Sprite.Group, Image: e.Sprite.Image, Time: -1}}
	}
	return nil
}

// HasCalls reports a lifebar drawing its own round, fight and KO calls.
func HasCalls(lb *sharedlifebar.Lifebar) bool {
	r := &lb.Round
	return drawable(r.RoundDefault) || drawable(r.Fight) || drawable(r.Ko)
}

// CallOf is the announcement a shown call draws, in round roundNo ending
// with result.
func CallOf(lb *sharedlifebar.Lifebar, c Call, roundNo int, result round.RoundResult) sharedlifebar.Announcement {
	switch c {
	case CallRound:
		return RoundCall(lb, roundNo)
	case CallFight:
		return lb.Round.Fight
	case CallWin:
		return lb.Round.Win[min(max(int(result.Winner), 0), 1)]
	case CallDraw:
		return lb.Round.Draw
	}
	return KOCall(lb, result.Outcome)
}
