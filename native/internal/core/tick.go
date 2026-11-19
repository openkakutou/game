package core

import (
	"strconv"

	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/engine/v2/round"
)

// Tick steps match id by one tick with both players' inputs, as
// {"p1": <input>, "p2": <input>} where each input has the shape of the
// engine's input.TickInput JSON: {"up","down","left","right","buttons":
// {"a":true,...}}, every key optional. Player 2's inputs are ignored when the
// match was started with p2cpu. It replies with the state to draw. A tick
// after a round ended is an error: call NextRound.
func (c *Core) Tick(id int, inputs string) string {
	s, ok := c.matches[id]
	if !ok {
		return fail("no match %d", id)
	}
	if err := parseInputs(inputs, &s.in); err != nil {
		return fail("invalid inputs: %v", err)
	}
	if s.match.RoundEnded() {
		return fail("the round is over: call nextRound")
	}
	if s.cpu != nil {
		self, other := s.match.State.Fighters[1].Position, s.match.State.Fighters[0].Position
		s.cpu.Decide(self, other, &s.in[1])
	}
	if err := s.match.Step(s.in); err != nil {
		return fail("the match stopped: %v", err)
	}
	s.followCamera(false)
	return s.snapshot(0)
}

// snapshot is the JSON reply carrying the state to draw, built in the
// session's reused buffer; the returned string is the call's only allocation.
// A non-zero id (a match start) adds the id and the once-per-match HUD values.
func (s *session) snapshot(id int) string {
	m := s.match
	b := append(s.buf[:0], `{"ok":true`...)
	if id != 0 {
		b = appendInt(b, "id", id)
		b = appendInt(b, "roundTimerTicks", s.roundTimerTicks)
		b = appendInt(b, "roundsToWin", s.roundsToWin)
	}
	b = appendInt(b, "tick", m.Ticks())
	b = appendInt(b, "round", m.State.Round)
	b = appendInt(b, "timer", m.State.RoundTimer)
	b = append(b, `,"roundEnded":`...)
	b = strconv.AppendBool(b, m.RoundEnded())
	b = append(b, `,"outcome":"`...)
	b = append(b, outcomeName(m.LastRound.Outcome)...)
	// Who won the round just ended; -1 while fighting and for a draw.
	roundWinner := -1
	if m.LastRound.Outcome == round.OutcomeKO || m.LastRound.Outcome == round.OutcomeTimeout {
		roundWinner = int(m.LastRound.Winner)
	}
	b = append(b, `","over":`...)
	b = strconv.AppendBool(b, m.Over)
	winner := -1
	if m.Over {
		winner = int(m.Winner)
	}
	b = appendInt(b, "winner", winner)
	b = appendInt(b, "roundWinner", roundWinner)
	b = append(b, `,"wins":[`...)
	b = strconv.AppendInt(b, int64(m.Progress.Wins[0]), 10)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(m.Progress.Wins[1]), 10)
	b = append(b, `],"camX":`...)
	b = appendFloat(b, s.camX)
	b = append(b, `,"fighters":[`...)
	for i := range m.State.Fighters {
		if i > 0 {
			b = append(b, ',')
		}
		b = s.appendFighter(b, i)
	}
	b = append(b, `],"cues":[`...)
	s.cues = m.DrainCues(s.cues)
	for i, cue := range s.cues {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '{')
		b = appendKey(b, "side")
		b = strconv.AppendInt(b, int64(cue.Side), 10)
		b = appendInt(b, "group", cue.Group)
		b = appendInt(b, "sample", cue.Sample)
		b = append(b, '}')
	}
	b = append(b, `]}`...)
	s.buf = b
	return string(b)
}

func (s *session) appendFighter(b []byte, i int) []byte {
	f := s.match.State.Fighters[i]
	facing := 1
	if f.Facing == match.FacingLeft {
		facing = -1
	}
	b = append(b, `{"x":`...)
	b = appendFloat(b, f.Position.X)
	b = append(b, `,"y":`...)
	b = appendFloat(b, f.Position.Y)
	b = appendInt(b, "facing", facing)
	b = appendInt(b, "anim", s.match.AnimNo[i])
	b = appendInt(b, "animTime", s.match.AnimTime[i])
	b = appendInt(b, "stateNo", f.StateNo)
	b = appendInt(b, "health", f.Health)
	b = appendInt(b, "power", f.Power)
	return append(b, '}')
}

// appendKey writes `"key":` with no leading comma.
func appendKey(b []byte, key string) []byte {
	b = append(b, '"')
	b = append(b, key...)
	return append(b, '"', ':')
}

// appendInt writes `,"key":v`.
func appendInt(b []byte, key string, v int) []byte {
	b = append(b, ',')
	b = appendKey(b, key)
	return strconv.AppendInt(b, int64(v), 10)
}

// appendFloat writes v as a JSON number; a non-finite value (which JSON
// cannot carry) is written as 0.
func appendFloat(b []byte, v float64) []byte {
	if v != v || v > 1.7976931348623157e308 || v < -1.7976931348623157e308 {
		v = 0
	}
	return strconv.AppendFloat(b, v, 'f', -1, 64)
}

func outcomeName(o round.Outcome) string {
	switch o {
	case round.OutcomeKO:
		return "ko"
	case round.OutcomeDoubleKO:
		return "doubleKo"
	case round.OutcomeTimeout:
		return "timeout"
	case round.OutcomeTimeoutDraw:
		return "draw"
	default:
		return "none"
	}
}
