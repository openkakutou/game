package game

import (
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/round"
)

// WinType is how a round was won (Ikemen GO's WT_*).
type WinType int

const (
	WinNormal WinType = iota
	WinSpecial
	WinHyper
	WinCheese
	WinTime
	WinThrow
	WinSuicide
)

// Win is a round won, as the lifebar's win icons show it: its type, perfect
// when the winner lost no life.
// shortcut: the teammate and clutch types are not told apart.
type Win struct {
	Type    WinType
	Perfect bool
}

// koWinType is the type of a KO win over loser, by the hit that knocked it
// out (lifeSet, char.go:8968-8992): not in a hit state, a suicide; guarded,
// a cheese KO; else by the HitDef's attr: hyper, special, throw or normal.
func koWinType(loser *evaluator.Context) WinType {
	switch {
	case loser.MoveType != 'H':
		return WinSuicide
	case loser.GetHit.Guarded:
		return WinCheese
	case evaluator.IsHyper(loser.GetHit.Attr):
		return WinHyper
	case evaluator.IsSpecial(loser.GetHit.Attr):
		return WinSpecial
	case evaluator.IsThrow(loser.GetHit.Attr):
		return WinThrow
	}
	return WinNormal
}

// recordWin adds the round just over to its winner's wins.
func (m *Match) recordWin(r round.RoundResult) {
	if r.Outcome != round.OutcomeKO && r.Outcome != round.OutcomeTimeout {
		return
	}
	w := int(r.Winner)
	if w < 0 || w > 1 {
		return
	}
	t := WinTime
	if r.Outcome == round.OutcomeKO {
		t = koWinType(&m.runtimes[1-w].Context)
	}
	m.Wins[w] = append(m.Wins[w], Win{Type: t, Perfect: m.State.Fighters[w].Health >= m.LifeMax[w]})
}
