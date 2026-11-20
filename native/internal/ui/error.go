package ui

import "unicode/utf8"

// errorLineSpacing is the distance between two lines of the error message,
// in lines of the message style's size.
// shortcut: the format has no line spacing for a wrapped text; this is
// OpenKakutou's. Upgrade when the format gets one.
const errorLineSpacing = 1.25

// ErrorState is what the game shows on the error screen.
type ErrorState struct {
	Title, Message, Hint string
	ElapsedMs            int
}

// Error adds the error screen to dst: title, the message wrapped inside the
// message part's width, hint.
func (l *Layout) Error(dst *DrawList, view View, s ErrorState) {
	e := l.pack.resolved.Screens.Error
	screen, from := view.rect(), len(dst.Ops)
	l.elements(dst, view, l.pack.scope("error", ""), e.Background, screen, s.ElapsedMs)
	if e.Title != nil {
		l.text(dst, view, l.pack.scope("error", "title"), &e.Title.Layout, e.Title.Style, s.Title, screen, 1, "")
	}
	if e.Message != nil {
		l.message(dst, view, s.Message, screen)
	}
	if e.Hint != nil {
		l.text(dst, view, l.pack.scope("error", "hint"), &e.Hint.Layout, e.Hint.Style, s.Hint, screen, 1, "")
	}
	l.elements(dst, view, l.pack.scope("error", ""), e.Foreground, screen, s.ElapsedMs)
	fadeIn(dst, from, &e.Screen, s.ElapsedMs)
}

// message draws the error text as lines, a block as wide as the part's width
// placed by its layout, each line aligned inside it by the style's align.
func (l *Layout) message(dst *DrawList, view View, text string, screen rect) {
	part := l.pack.resolved.Screens.Error.Message
	style, ok := l.styled(l.pack.scope("error", "message"), part.Style, view)
	if !ok {
		return
	}
	unit := view.unit()
	width := float32(part.Width) * unit
	lines := l.wrap(style, text, width)
	lineH := float32(style.style.Size) * unit * errorLineSpacing
	box := place(screen, &part.Layout, width, lineH*float32(len(lines)), unit)
	af := alignFactor(style.style.Align)
	for i, line := range lines {
		style.emit(dst, line, box.x+(width-style.width(line))*af, box.y+(float32(i)+0.5)*lineH, 1)
	}
}

// wrapCache is the last wrapped text: the lines are substrings of it, so
// wrapping the same text again at the same width and size costs nothing.
type wrapCache struct {
	text  string
	width float32
	font  FontRef
	px    int
	lines []string
}

// wrap breaks text into lines no wider than width pixels, at spaces, using
// the style's measured advances; a word wider than a line is broken where
// the next letter would not fit (always at least one letter per line).
func (l *Layout) wrap(style styled, text string, width float32) []string {
	c := &l.wrapped
	if c.lines != nil && c.text == text && c.width == width && c.font == style.font && c.px == style.px {
		return c.lines
	}
	c.text, c.width, c.font, c.px = text, width, style.font, style.px
	c.lines = c.lines[:0]
	space := style.width(" ")
	for i := 0; i < len(text); {
		for i < len(text) && isBreak(text[i]) {
			i++
		}
		if i == len(text) {
			break
		}
		start, end, used := i, i, float32(0)
		for i < len(text) {
			word := i
			for i < len(text) && !isBreak(text[i]) {
				i++
			}
			wordW := style.width(text[word:i])
			if end > start && used+space+wordW > width {
				i = word // the word starts the next line
				break
			}
			if end == start && wordW > width {
				i = word + style.fit(text[word:i], width)
				end = i
				break
			}
			if end > start {
				used += space
			}
			used += wordW
			end = i
			newline := false
			for i < len(text) && isBreak(text[i]) {
				newline = newline || text[i] == '\n'
				i++
			}
			if newline {
				break // a newline in the message ends the line
			}
		}
		c.lines = append(c.lines, text[start:end])
	}
	if c.lines == nil {
		c.lines = []string{}
	}
	return c.lines
}

func isBreak(b byte) bool { return b == ' ' || b == '\n' }

// fit is how many bytes of word, from its start, fit in width pixels: at
// least the first letter.
func (s styled) fit(word string, width float32) int {
	var used float32
	for i, r := range word {
		adv, _ := s.face.GlyphAdvance(s.rune(r))
		used += float32(adv) / 64
		if used > width {
			if i == 0 {
				return utf8.RuneLen(r)
			}
			return i
		}
	}
	return len(word)
}
