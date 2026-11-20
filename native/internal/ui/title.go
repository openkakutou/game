package ui

// TitlePhase is which part of the title screen is shown.
type TitlePhase uint8

const (
	// PhasePressStart shows the "press start" prompt, before the menu opens.
	PhasePressStart TitlePhase = iota
	// PhaseMenu shows the main menu.
	PhaseMenu
)

// TitleState is what the game shows on the title screen.
type TitleState struct {
	Phase   TitlePhase
	Entries []string // the menu's labels, in the game's order
	Cursor  int      // the active entry
	// ElapsedMs is the time since the screen appeared: it drives the
	// transitions, the blinking and the sprite animations.
	ElapsedMs int
	Version   string // the game version, shown when not empty
}

// Title adds the title screen to dst: background, logo, the prompt or the
// menu, the version, foreground.
func (l *Layout) Title(dst *DrawList, view View, s TitleState) {
	t := l.pack.resolved.Screens.Title
	screen := view.rect()
	l.elements(dst, view, l.pack.scope("title", ""), t.Background, screen, s.ElapsedMs)
	if t.Logo != nil {
		l.element(dst, view, l.pack.scope("title", "logo"), t.Logo, screen, s.ElapsedMs)
	}
	switch s.Phase {
	case PhasePressStart:
		l.pressStart(dst, view, s, screen)
	case PhaseMenu:
		if t.Menu != nil {
			l.menu(dst, view, l.pack.scope("title", "menu"), t.Menu, l.labels(s.Entries), s.Cursor, screen, s.ElapsedMs, noSlots)
		}
	}
	if t.Version != nil && s.Version != "" {
		l.text(dst, view, l.pack.scope("title", "version"), &t.Version.Layout, t.Version.Style, s.Version, screen, 1, "")
	}
	l.elements(dst, view, l.pack.scope("title", ""), t.Foreground, screen, s.ElapsedMs)
}

// pressStart draws the prompt, hidden for blinkMs out of every 2*blinkMs.
func (l *Layout) pressStart(dst *DrawList, view View, s TitleState, screen rect) {
	ps := l.pack.resolved.Screens.Title.PressStart
	if ps == nil {
		return
	}
	if ps.BlinkMs != nil && *ps.BlinkMs > 0 && (s.ElapsedMs / *ps.BlinkMs)%2 == 1 {
		return
	}
	l.text(dst, view, l.pack.scope("title", "pressStart"), &ps.Layout, ps.Style, ps.Text, screen, 1, "")
}
