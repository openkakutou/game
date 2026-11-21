package game

import "github.com/openkakutou/engine/v2/round"

// Win is a round won, as the lifebar's win icons show it (Ikemen GO's win
// types, WT_*): by KO or by time, perfect when the winner lost no life.
// shortcut: the special, hyper, cheese, throw, suicide, teammate and clutch
// types are not told apart: a KO win is a normal one.
type Win struct {
	Time, Perfect bool
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
	m.Wins[w] = append(m.Wins[w], Win{Time: r.Outcome == round.OutcomeTimeout, Perfect: m.State.Fighters[w].Health >= m.LifeMax[w]})
}
