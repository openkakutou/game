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
)

// Announcer paces the lifebar's round, fight and KO calls as Ikemen GO's
// FightScreenRound does (fs.go:3563-3900): the round call ctrl.time into the
// intro, round.time after it starts, until its animation or display time ends;
// the fight call callfight.time after the round call shows; once the round
// is decided, the KO, double KO or time over call, its time (plus MUGEN's 10
// ticks) after, shown at the game's speed rather than the slow motion's.
// Each call's sound plays at its sndtime.
// shortcut: round.single / round.final, the win and draw calls and the bg
// and top layers are not played; MUGEN's 10-tick KO delay is
// always applied (the lifebar library does not report ikemenversion).
type Announcer struct {
	roundPhase, roundTimer int
	fightPhase, fightTimer int
	koPhase, koTimer       int
	outro                  bool
	finish                 round.Outcome
	// sounds are the calls' sounds due, drained by DrainSounds.
	sounds []sharedlifebar.SoundRef
}

// DrainSounds returns dst followed by the calls' sounds due since the last
// drain, and forgets them.
func (a *Announcer) DrainSounds(dst []sharedlifebar.SoundRef) []sharedlifebar.SoundRef {
	dst = append(dst, a.sounds...)
	a.sounds = a.sounds[:0]
	return dst
}

// Finish is the outcome the KO call announces.
func (a *Announcer) Finish() round.Outcome { return a.finish }

// mugenKODelay is the delay Ikemen GO adds to a MUGEN lifebar's KO call.
const mugenKODelay = 10

// Shown is a call on screen and how many ticks its animation has played.
type Shown struct {
	Call    Call
	Elapsed int
}

// Showing is the calls on screen, round, fight then KO, CallNone in the
// slots of those not showing; Ikemen GO draws them together.
func (a *Announcer) Showing(lb *sharedlifebar.Lifebar) [3]Shown {
	r := &lb.Round
	var out [3]Shown
	if a.roundPhase == 1 {
		out[0] = Shown{CallRound, max(a.roundTimer-r.RoundTime-1, 0)}
	}
	if a.fightPhase == 1 {
		out[1] = Shown{CallFight, max(a.fightTimer-r.FightTime-1, 0)}
	}
	if a.koPhase == 1 {
		out[2] = Shown{koCall(a.finish), max(a.koTimer-KOCallTime(lb, a.finish)-mugenKODelay-1, 0)}
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
func (a *Announcer) Tick(lb *sharedlifebar.Lifebar, roundNo, intro int, finish round.Outcome) {
	r := &lb.Round
	if intro > r.CtrlTime {
		*a = Announcer{}
		return
	}
	if a.roundPhase < 2 || a.fightPhase < 2 {
		a.intro(lb, roundNo)
	}
	if a.fightPhase == 2 && intro < 0 && finish != round.OutcomeNone && !a.outro {
		a.outro, a.finish = true, finish
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
			a.sounds = append(a.sounds, roundSound(lb, roundNo))
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
			a.sounds = append(a.sounds, r.Fight.Sound)
		}
		if a.fightTimer >= r.FightTime && ended(lb, r.Fight, a.fightTimer-r.FightTime+1, true) && a.fightTimer >= r.FightSndTime {
			a.fightPhase = 2
		}
		a.fightTimer++
	}
}

func (a *Announcer) stepOutro(lb *sharedlifebar.Lifebar) {
	if a.koPhase >= 2 {
		return
	}
	start := KOCallTime(lb, a.finish) + mugenKODelay
	if a.koTimer == koSoundTime(lb, a.finish)+mugenKODelay {
		a.sounds = append(a.sounds, KOCall(lb, a.finish).Sound)
	}
	if a.koTimer >= start {
		a.koPhase = 1
		if ended(lb, KOCall(lb, a.finish), a.koTimer-start+1, false) {
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

// CallOf is the announcement a shown call draws, in round roundNo decided
// with finish.
func CallOf(lb *sharedlifebar.Lifebar, c Call, roundNo int, finish round.Outcome) sharedlifebar.Announcement {
	switch c {
	case CallRound:
		return RoundCall(lb, roundNo)
	case CallFight:
		return lb.Round.Fight
	}
	return KOCall(lb, finish)
}
