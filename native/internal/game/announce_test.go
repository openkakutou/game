package game

import (
	"github.com/openkakutou/engine/v2/evaluator"
	"slices"
	"testing"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/engine/v2/round"
	sharedlifebar "github.com/openkakutou/lifebar"
)

// callLifebar has a round call of a 3-tick animation shown 2 ticks into the
// intro, a fight call shown for 4 ticks right after it, and a KO held on
// its last frame.
func callLifebar() *sharedlifebar.Lifebar {
	anim := func(times ...int) air.Animation {
		var fs []air.Frame
		for _, t := range times {
			fs = append(fs, air.Frame{Time: t})
		}
		return air.Animation{Frames: fs}
	}
	lb := &sharedlifebar.Lifebar{Animations: map[int]air.Animation{1: anim(1, 2), 2: anim(2, -1)}}
	lb.Round.CtrlTime = 30
	lb.Round.RoundTime = 2
	lb.Round.RoundDefault = sharedlifebar.Announcement{Element: sharedlifebar.Element{HasAnim: true, Anim: 1}, DisplayTime: -2}
	lb.Round.CallFightTime = 60
	lb.Round.Fight = sharedlifebar.Announcement{Text: sharedlifebar.Text{Text: "Fight"}, DisplayTime: 4}
	lb.Round.KoTime = 5
	lb.Round.Ko = sharedlifebar.Announcement{Element: sharedlifebar.Element{HasAnim: true, Anim: 2}, DisplayTime: -2}
	// The library's defaults for the calls left out: no sound, no display time.
	none := sharedlifebar.Announcement{Sound: sharedlifebar.SoundRef{Group: -1}, DisplayTime: -2}
	lb.Round.Win, lb.Round.Draw, lb.Round.Dko, lb.Round.To = [2]sharedlifebar.Announcement{none, none}, none, none, none
	return lb
}

// fs.go:3563-3900: the round call waits round.time, plays its animation
// through, the fight call follows callfight.time after the round call
// started showing and lasts its display time.
func TestAnnouncer_RoundThenFightCalls(t *testing.T) {
	lb := callLifebar()
	var a Announcer
	var shown []Call
	for intro := 40; intro >= 0; intro-- {
		a.Tick(lb, 1, intro, round.RoundResult{Outcome: round.OutcomeNone})
		shown = append(shown, a.Showing(lb)[0].Call)
	}
	// Intro 40..31 is before ctrl.time: nothing.
	for i, c := range shown[:10] {
		if c != CallNone {
			t.Fatalf("tick %d before ctrl.time shows %v", i, c)
		}
	}
	// Intro 30, 31: round.time; then 3 ticks of round call.
	want := []Call{CallNone, CallNone, CallRound, CallRound, CallRound, CallNone}
	for i, c := range want {
		if shown[10+i] != c {
			t.Fatalf("calls from ctrl.time = %v, want %v first", shown[10:20], want)
		}
	}
	fights := 0
	b := Announcer{}
	for intro := 40; intro >= -100; intro-- {
		// The fight goes on at intro 0: callfight.time may outlast the intro.
		b.Tick(lb, 1, max(intro, 0), round.RoundResult{Outcome: round.OutcomeNone})
		if b.Showing(lb)[1].Call == CallFight {
			fights++
		}
	}
	if fights == 0 {
		t.Error("the fight call never showed")
	}
}

func TestAnnouncer_FightCallLastsItsDisplayTime(t *testing.T) {
	lb := callLifebar()
	lb.Round.CallFightTime = 0
	var a Announcer
	fights := 0
	for intro := 30; intro >= 0; intro-- {
		a.Tick(lb, 1, intro, round.RoundResult{Outcome: round.OutcomeNone})
		if a.Showing(lb)[1].Call == CallFight {
			fights++
		}
	}
	// The tick it starts, then display time 4 less the tick it ends.
	if fights != 4 {
		t.Errorf("fight call showed %d ticks, want 4", fights)
	}
}

// The KO call starts ko.time + MUGEN's 10 ticks after the decision, runs on
// frames (not slowed), and holds on a last frame of time -1.
func TestAnnouncer_KOCallHoldsAfterItsDelay(t *testing.T) {
	lb := callLifebar()
	lb.Round.CallFightTime = 0
	var a Announcer
	for intro := 30; intro >= 0; intro-- {
		a.Tick(lb, 1, intro, round.RoundResult{Outcome: round.OutcomeNone})
	}
	for range 20 {
		a.Tick(lb, 1, 0, round.RoundResult{Outcome: round.OutcomeNone})
	}
	a.Tick(lb, 1, -1, round.RoundResult{Outcome: round.OutcomeKO})
	frames := 1
	for ; frames < 40; frames++ {
		if a.Showing(lb)[2].Call == CallKO {
			break
		}
		a.Frame(lb)
	}
	if frames != 5+mugenKODelay+1 {
		t.Errorf("KO call after %d frames, want ko.time + 10 + 1", frames)
	}
	for range 100 {
		a.Frame(lb)
	}
	if a.Showing(lb)[2].Call != CallKO {
		t.Error("KO call gone after its last frame of time -1")
	}
}

func TestAnnouncer_PicksTheRoundsOwnCallAndTheOutcomesCall(t *testing.T) {
	lb := callLifebar()
	lb.Round.Round[1] = sharedlifebar.Announcement{Text: sharedlifebar.Text{Text: "Round 2"}}
	if got := RoundCall(lb, 2).Text.Text; got != "Round 2" {
		t.Errorf("round 2 call = %q", got)
	}
	if got := RoundCall(lb, 3); got.Element.Anim != 1 {
		t.Errorf("round 3 call = %+v, want round.default", got)
	}
	if RoundCall(lb, 12).Element.Anim != 1 {
		t.Error("round 12 has no own call: want round.default")
	}
	lb.Round.To = sharedlifebar.Announcement{Text: sharedlifebar.Text{Text: "Time"}}
	if KOCall(lb, round.OutcomeTimeout).Text.Text != "Time" || KOCall(lb, round.OutcomeKO).Element.Anim != 2 {
		t.Error("outcome calls mixed up")
	}
}

func TestEnded_FollowsAnimTextSndEnd(t *testing.T) {
	lb := callLifebar()
	none := sharedlifebar.Announcement{DisplayTime: -2}
	if !ended(lb, none, 1, true) {
		t.Error("nothing to show has ended")
	}
	held := lb.Round.Ko
	if ended(lb, held, 100, false) {
		t.Error("a last frame of time -1 holds without inf")
	}
	if ended(lb, held, 2, true) || !ended(lb, held, 3, true) {
		t.Error("with inf it ends on reaching the held frame")
	}
}

func TestRecordWin_KeepsTheWinnersRoundsAndTheirKind(t *testing.T) {
	m := &Match{LifeMax: [2]int{1000, 1000}}
	m.State.Fighters[0].Health, m.State.Fighters[1].Health = 1000, 400
	m.runtimes[1].Context.MoveType = 'H'
	m.recordWin(round.RoundResult{Outcome: round.OutcomeKO, Winner: 0})
	m.recordWin(round.RoundResult{Outcome: round.OutcomeTimeout, Winner: 1})
	m.recordWin(round.RoundResult{Outcome: round.OutcomeDoubleKO})
	if len(m.Wins[0]) != 1 || !m.Wins[0][0].Perfect || m.Wins[0][0].Type != WinNormal {
		t.Errorf("P1 wins = %+v, want one perfect KO", m.Wins[0])
	}
	if len(m.Wins[1]) != 1 || m.Wins[1][0].Type != WinTime || m.Wins[1][0].Perfect {
		t.Errorf("P2 wins = %+v, want one time win", m.Wins[1])
	}
}

// fs.go handleRoundIntro / handleRoundOutro: each call's sound plays at its
// sndtime: the round's own sound, else round.default's; the fight sound;
// the KO sound sndtime + 10 frames after the decision.
func TestAnnouncer_PlaysEachCallsSoundAtItsTime(t *testing.T) {
	lb := callLifebar()
	lb.Round.RoundSndTime, lb.Round.FightSndTime, lb.Round.KoSndTime = 1, 0, 2
	lb.Round.RoundDefault.Sound = sharedlifebar.SoundRef{Group: 0, Number: 9}
	lb.Round.Round[1].Sound = sharedlifebar.SoundRef{Group: 0, Number: 2}
	lb.Round.Fight.Sound = sharedlifebar.SoundRef{Group: 1, Number: 0}
	lb.Round.Ko.Sound = sharedlifebar.SoundRef{Group: 2, Number: 0}
	lb.Round.CallFightTime = 0
	var a Announcer
	var heard []sharedlifebar.SoundRef
	for intro := 30; intro >= 0; intro-- {
		a.Tick(lb, 2, intro, round.RoundResult{Outcome: round.OutcomeNone})
		heard = a.DrainSounds(heard)
	}
	a.Tick(lb, 2, -1, round.RoundResult{Outcome: round.OutcomeKO})
	for range 20 {
		a.Frame(lb)
	}
	heard = a.DrainSounds(heard)
	want := []sharedlifebar.SoundRef{{Group: 0, Number: 2}, {Group: 1, Number: 0}, {Group: 2, Number: 0}}
	if len(heard) != len(want) {
		t.Fatalf("sounds = %v, want %v", heard, want)
	}
	for i := range want {
		if heard[i] != want[i] {
			t.Fatalf("sounds = %v, want %v", heard, want)
		}
	}
}

// fs.go handleRoundOutro: over.waittime after the decision the winner's
// win call shows from win.time for its display time; a time over draw shows
// the draw call; a double KO without dko.showdraw shows none.
func TestAnnouncer_WinCallFollowsOverWaitTime(t *testing.T) {
	lb := callLifebar()
	lb.Round.CallFightTime = 0
	lb.Round.OverWaitTime, lb.Round.WinTime = 3, 2
	lb.Round.Win[1] = sharedlifebar.Announcement{Text: sharedlifebar.Text{Text: "%s Wins"}, DisplayTime: 5, Sound: sharedlifebar.SoundRef{Group: -1}}
	lb.Round.Draw = sharedlifebar.Announcement{Text: sharedlifebar.Text{Text: "Draw"}, DisplayTime: 5, Sound: sharedlifebar.SoundRef{Group: -1}}
	run := func(result round.RoundResult) (calls []Call) {
		var a Announcer
		for intro := 30; intro >= 0; intro-- {
			a.Tick(lb, 1, intro, round.RoundResult{})
		}
		for intro := -1; intro > -20; intro-- {
			a.Tick(lb, 1, intro, result)
			a.Frame(lb)
			calls = append(calls, a.Showing(lb)[3].Call)
		}
		return calls
	}
	win := run(round.RoundResult{Outcome: round.OutcomeKO, Winner: 1})
	first := slices.Index(win, CallWin)
	if first < 3+2 || slices.Contains(win[:first], CallWin) {
		t.Errorf("win calls %v: want none before over.waittime + win.time", win)
	}
	// End is checked before the frame draws: display time 5 shows 4 frames.
	if n := countCalls(win, CallWin); n != 4 {
		t.Errorf("win call shown %d frames, want 4 (display time 5)", n)
	}
	if got := CallOf(lb, CallWin, 1, round.RoundResult{Outcome: round.OutcomeKO, Winner: 1}).Text.Text; got != "%s Wins" {
		t.Errorf("P2's win call = %q", got)
	}
	if n := countCalls(run(round.RoundResult{Outcome: round.OutcomeTimeoutDraw}), CallDraw); n != 4 {
		t.Errorf("draw call shown %d frames, want 4", n)
	}
	if n := countCalls(run(round.RoundResult{Outcome: round.OutcomeDoubleKO}), CallDraw) + countCalls(run(round.RoundResult{Outcome: round.OutcomeDoubleKO}), CallWin); n != 0 {
		t.Errorf("double KO without dko.showdraw showed %d call frames", n)
	}
}

func countCalls(calls []Call, c Call) int {
	n := 0
	for _, x := range calls {
		if x == c {
			n++
		}
	}
	return n
}

// char.go:8968-8992: a KO's win type comes from the hit that knocked the
// loser out.
func TestKOWinType_FollowsTheFinishingHit(t *testing.T) {
	attr := func(text string) int32 {
		a, err := evaluator.ParseAttr(text, true)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	cases := []struct {
		ctx  evaluator.Context
		want WinType
	}{
		{evaluator.Context{}, WinSuicide},
		{evaluator.Context{MoveType: 'H', GetHit: evaluator.GetHit{Guarded: true, Attr: attr("S, SA")}}, WinCheese},
		{evaluator.Context{MoveType: 'H', GetHit: evaluator.GetHit{Attr: attr("S, HA")}}, WinHyper},
		{evaluator.Context{MoveType: 'H', GetHit: evaluator.GetHit{Attr: attr("S, SP")}}, WinSpecial},
		{evaluator.Context{MoveType: 'H', GetHit: evaluator.GetHit{Attr: attr("S, NT")}}, WinThrow},
		{evaluator.Context{MoveType: 'H', GetHit: evaluator.GetHit{Attr: attr("S, NA")}}, WinNormal},
	}
	for _, c := range cases {
		if got := koWinType(&c.ctx); got != c.want {
			t.Errorf("%+v: %v, want %v", c.ctx.GetHit, got, c.want)
		}
	}
}
