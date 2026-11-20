package ui

import "github.com/openkakutou/game/native/internal/game"

// Arrange sets the pack's choices on f: the character grid's width and
// random cell, the move
// list's visible lines, how long the victory screen shows, and the order of
// the character and stage lists by the pack's sort rules (the picks follow
// their entries).
func (p *Pack) Arrange(f *game.Flow) {
	charBy, charDescending := p.Sort("characterSelect")
	stageBy, stageDescending := p.Sort("stageSelect")
	f.SortRosters(charBy, charDescending, stageBy, stageDescending)
	f.GridColumns = p.SelectColumns()
	f.RandomCell = p.HasRandomCell()
	f.MoveListWindow = p.MoveListWindow()
	f.VictoryTicks = p.VictoryDurationMs() * game.TickRateHz / 1000
}
