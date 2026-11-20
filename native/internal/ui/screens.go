package ui

import "github.com/openkakutou/screenpack"

// noticeLines is how many line heights apart a notice sits from the line
// above it (the hint or the previous notice).
const noticeLines = 1.5

// OptionsState is what the game shows on an options screen.
type OptionsState struct {
	Title  string
	Rows   []Row
	Cursor int
	// Controls is set on a player's bindings screen: rows have a key and a
	// button column, Column (0 key, 1 button) is the chosen one, and Waiting
	// says the cursor row's chosen slot awaits a key or button.
	Controls bool
	Column   int
	Waiting  bool
	// Prompt is the question or instruction, shown as the hint; Notices are
	// warnings stacked above it.
	Prompt    string
	Notices   []string
	ElapsedMs int
}

// Options adds an options screen to dst: background, title, the rows as the
// menu part (values in columns, bindings with the controls part's slot
// styles), the hint and notices, foreground. The format's tabs are not drawn:
// the game's options are a list of pages, each a screen of its own.
func (l *Layout) Options(dst *DrawList, view View, s OptionsState) {
	o := l.pack.resolved.Screens.Options
	screen := view.rect()
	l.elements(dst, view, l.pack.scope("options", ""), o.Background, screen, s.ElapsedMs)
	if o.Title != nil {
		l.text(dst, view, l.pack.scope("options", "title"), &o.Title.Layout, o.Title.Style, s.Title, screen, 1, "")
	}
	if o.Menu != nil {
		sl := noSlots
		if s.Controls && o.Controls != nil {
			sl = slots{sc: l.pack.scope("options", "controls"), style: o.Controls.SlotStyle, waitingStyle: o.Controls.WaitingStyle, column: s.Column, waiting: s.Waiting}
		}
		l.menu(dst, view, l.pack.scope("options", "menu"), o.Menu, s.Rows, s.Cursor, screen, s.ElapsedMs, sl)
	}
	if o.Hint != nil {
		l.hint(dst, view, o.Hint, s.Prompt, s.Notices, screen)
	}
	l.elements(dst, view, l.pack.scope("options", ""), o.Foreground, screen, s.ElapsedMs)
}

// hint draws the prompt in the hint part and each notice one step above the
// last, in the warning color.
func (l *Layout) hint(dst *DrawList, view View, part *screenpack.TextPart, prompt string, notices []string, screen rect) {
	sc := l.pack.scope("options", "hint")
	if prompt != "" {
		l.text(dst, view, sc, &part.Layout, part.Style, prompt, screen, 1, "")
	}
	style, ok := l.styled(sc, part.Style, view)
	if !ok {
		return
	}
	step := float32(style.style.Size) * noticeLines * view.unit()
	for i, n := range notices {
		above := rect{screen.x, screen.y - float32(i+1)*step, screen.w, screen.h}
		l.text(dst, view, sc, &part.Layout, part.Style, n, above, 1, warningColor)
	}
}

// PauseState is what the game shows on the pause screen, over the match.
type PauseState struct {
	Title     string
	Entries   []string
	Cursor    int
	ElapsedMs int
}

// Pause adds the pause screen to dst: the dim over the frozen match, the
// title and the menu.
func (l *Layout) Pause(dst *DrawList, view View, s PauseState) {
	p := l.pack.resolved.Screens.Pause
	screen := view.rect()
	l.elements(dst, view, l.pack.scope("pause", ""), p.Background, screen, s.ElapsedMs)
	if p.Dim != nil {
		dst.Ops = append(dst.Ops, Op{Kind: OpRect, X: screen.x, Y: screen.y, W: screen.w, H: screen.h, Color: parseColor(p.Dim.Color, 1)})
	}
	if p.Title != nil {
		l.text(dst, view, l.pack.scope("pause", "title"), &p.Title.Layout, p.Title.Style, s.Title, screen, 1, "")
	}
	if p.Menu != nil {
		l.menu(dst, view, l.pack.scope("pause", "menu"), p.Menu, l.labels(s.Entries), s.Cursor, screen, s.ElapsedMs, noSlots)
	}
	l.elements(dst, view, l.pack.scope("pause", ""), p.Foreground, screen, s.ElapsedMs)
}

// MatchSetupState is what the game shows on the match setup screen.
type MatchSetupState struct {
	Title     string
	Rows      []Row
	Cursor    int
	ElapsedMs int
}

// MatchSetup adds the match setup screen to dst: title and the rows with
// their values.
func (l *Layout) MatchSetup(dst *DrawList, view View, s MatchSetupState) {
	m := l.pack.resolved.Screens.MatchSetup
	screen := view.rect()
	l.elements(dst, view, l.pack.scope("matchSetup", ""), m.Background, screen, s.ElapsedMs)
	if m.Title != nil {
		l.text(dst, view, l.pack.scope("matchSetup", "title"), &m.Title.Layout, m.Title.Style, s.Title, screen, 1, "")
	}
	if m.Menu != nil {
		l.menu(dst, view, l.pack.scope("matchSetup", "menu"), m.Menu, s.Rows, s.Cursor, screen, s.ElapsedMs, noSlots)
	}
	l.elements(dst, view, l.pack.scope("matchSetup", ""), m.Foreground, screen, s.ElapsedMs)
}

// VersusState is what the game shows on the VS screen.
type VersusState struct {
	P1, P2, Stage string
	ElapsedMs     int
}

// Versus adds the VS screen to dst: both names and the stage.
// shortcut: the portraits are not drawn; they come with the character
// select screens, which load the characters' portrait sprites.
func (l *Layout) Versus(dst *DrawList, view View, s VersusState) {
	v := l.pack.resolved.Screens.Versus
	screen := view.rect()
	l.elements(dst, view, l.pack.scope("versus", ""), v.Background, screen, s.ElapsedMs)
	if v.Name != nil {
		sc := l.pack.scope("versus", "name")
		if v.Name.P1 != nil {
			l.text(dst, view, sc, &v.Name.P1.Layout, v.Name.P1.Style, s.P1, screen, 1, "")
		}
		if v.Name.P2 != nil {
			l.text(dst, view, sc, &v.Name.P2.Layout, v.Name.P2.Style, s.P2, screen, 1, "")
		}
	}
	if v.Stage != nil {
		l.text(dst, view, l.pack.scope("versus", "stage"), &v.Stage.Layout, v.Stage.Style, s.Stage, screen, 1, "")
	}
	l.elements(dst, view, l.pack.scope("versus", ""), v.Foreground, screen, s.ElapsedMs)
}

// ResultsState is what the game shows on the results screen.
type ResultsState struct {
	Message, Score, Winner string
	Entries                []string
	Cursor                 int
	ElapsedMs              int
}

// Results adds the results screen to dst: the winner's name, the message, the
// score and the menu.
// shortcut: the winner's portrait is not drawn, see Versus.
func (l *Layout) Results(dst *DrawList, view View, s ResultsState) {
	r := l.pack.resolved.Screens.Results
	screen := view.rect()
	l.elements(dst, view, l.pack.scope("results", ""), r.Background, screen, s.ElapsedMs)
	if r.Winner != nil && r.Winner.Name != nil {
		l.text(dst, view, l.pack.scope("results", "winner"), &r.Winner.Name.Layout, r.Winner.Name.Style, s.Winner, screen, 1, "")
	}
	if r.Message != nil {
		l.text(dst, view, l.pack.scope("results", "message"), &r.Message.Layout, r.Message.Style, s.Message, screen, 1, "")
	}
	if r.Score != nil {
		l.text(dst, view, l.pack.scope("results", "score"), &r.Score.Layout, r.Score.Style, s.Score, screen, 1, "")
	}
	if r.Menu != nil {
		l.menu(dst, view, l.pack.scope("results", "menu"), r.Menu, l.labels(s.Entries), s.Cursor, screen, s.ElapsedMs, noSlots)
	}
	l.elements(dst, view, l.pack.scope("results", ""), r.Foreground, screen, s.ElapsedMs)
}
