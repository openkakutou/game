package game

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openkakutou/character"
	"github.com/openkakutou/engine/v2/input"
)

func text(s string) MoveRun            { return MoveRun{Text: s} }
func icon(id, fallback string) MoveRun { return MoveRun{Text: fallback, Icon: id} }

func TestParseRuns_GlyphTokensOfIkemenGoBecomeIconsOrText(t *testing.T) {
	// Token set: defaultMotif.ini [Glyphs] at 54d4148a, lines 6478-6559.
	cases := []struct {
		in   string
		want []MoveRun
	}{
		{"_D_DF_F^LP", []MoveRun{icon("down", "D"), text("DF"), icon("forward", "F"), icon("x", "LP")}},
		{"^A ^B ^C ^X ^Y ^Z ^S", []MoveRun{
			icon("a", "A"), text(" "), icon("b", "B"), text(" "), icon("c", "C"), text(" "),
			icon("x", "X"), text(" "), icon("y", "Y"), text(" "), icon("z", "Z"), text(" "), icon("start", "S")}},
		{"_U_B^MK^HK^MP^HP^LK", []MoveRun{icon("up", "U"), icon("back", "B"), icon("b", "MK"), icon("c", "HK"), icon("y", "MP"), icon("z", "HP"), icon("a", "LK")}},
		{"_QCF_+^2P", []MoveRun{text("QCF"), text("+"), text("2P")}},
		{"~DF~D", []MoveRun{text("~DF"), text("~D")}},
		{"Hold _CHARGE, _AIROK", []MoveRun{text("Hold "), text("CHARGE"), text(", "), text("AIROK")}},
		{"_^ _= _. _-", []MoveRun{text("^"), text(" "), text("="), text(" "), text("."), text(" "), text("-")}},
		{"^Q _Z plain", []MoveRun{text("^Q _Z plain")}},
	}
	for _, c := range cases {
		if got := parseRuns(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseRuns(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseRuns_TagsAreGlyphsOrColorsAsIkemenGoReadsThem(t *testing.T) {
	// menu.lua:736-747: <#rrggbb>, <#> and </> are colors (dropped here), any
	// other tag is a glyph, drawn as its token when the table lacks it;
	// &lt; &gt; &amp; are unescaped inside a tag.
	got := parseRuns("a<#ff0000>b</>c<_D>d<foo>e<&lt;>f<x y>")
	want := []MoveRun{text("a"), text("b"), text("c"), icon("down", "D"), text("d"), text("foo"), text("e"), text("<"), text("f<x y>")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("runs = %v, want %v", got, want)
	}
}

func TestParseMovelist_TabsSetTheAlignmentOfEachCell(t *testing.T) {
	// menu.lua:797-808: no tab left, one center, two or more right; trailing
	// whitespace dropped; the blank line of the last newline dropped.
	got := parseMovelist("Move\t\t_D^A  \r\n\n\tmid\tleft?\n\t\t\tfar\n")
	want := []MoveLine{
		{{MoveLeft, []MoveRun{text("Move")}}, {MoveRight, []MoveRun{icon("down", "D"), icon("a", "A")}}},
		nil,
		{{MoveCenter, []MoveRun{text("mid")}}, {MoveCenter, []MoveRun{text("left?")}}},
		{{MoveRight, []MoveRun{text("far")}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %#v, want %#v", got, want)
	}
	if n := len(parseMovelist("a\nb")); n != 2 {
		t.Errorf("a text without a final newline has %d lines, want 2", n)
	}
	if n := len(parseMovelist("a\n\n")); n != 2 {
		t.Errorf("a text ending with a blank line has %d lines, want 2 (a and the blank)", n)
	}
}

const kfmLikeCmd = `
[Command]
name = "QCF_a"
command = ~D, DF, F, a
time = 15

[Command]
name = "FF"
command = F, F

[Command]
name = "holdfwd"
command = /$F

[Command]
name = "FF"
command = F, F, F

[Command]
name = "combo"
command = $D, x+y, ~30s
`

func cmdFighter(t *testing.T, raw string, lists ...character.Movelist) *Fighter {
	t.Helper()
	parsed, err := character.ParseCmd([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return &Fighter{Char: &character.Character{Movelists: lists}, Cmd: parsed}
}

func TestMoveLinesOf_WithoutAMovelistTheCommandsOfTheCmdAreListed(t *testing.T) {
	got := moveLinesOf(cmdFighter(t, kfmLikeCmd))
	want := []MoveLine{
		{{MoveLeft, []MoveRun{text("QCF_a")}}, {MoveRight, []MoveRun{
			text("~"), icon("down", "D"), text(" "), text("DF"), text(" "), icon("forward", "F"), text(" "), icon("a", "a")}}},
		{{MoveLeft, []MoveRun{text("FF")}}, {MoveRight, []MoveRun{icon("forward", "F"), text(" "), icon("forward", "F")}}},
		{{MoveLeft, []MoveRun{text("combo")}}, {MoveRight, []MoveRun{
			text("$"), icon("down", "D"), text(" "), icon("x", "x"), text("+"), icon("y", "y"), text(" "), text("~30"), icon("start", "s")}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %#v\nwant %#v", got, want)
	}
}

func TestMoveLinesOf_TheFirstReadableMovelistOfIndexZeroWins(t *testing.T) {
	lists := []character.Movelist{
		{Key: "movelist", Index: 0, Error: "unreadable"},
		{Key: "movelist1", Index: 1, Text: "other"},
		{Key: "movelist0", Index: 0, Text: "Punch\t\t^X"},
		{Key: "movelist", Index: 0, Text: "later"},
	}
	got := moveLinesOf(cmdFighter(t, kfmLikeCmd, lists...))
	want := []MoveLine{{{MoveLeft, []MoveRun{text("Punch")}}, {MoveRight, []MoveRun{icon("x", "X")}}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %#v, want %#v", got, want)
	}
}

func TestMoveLinesOf_AnEmptyMovelistFallsBackToTheCommands(t *testing.T) {
	got := moveLinesOf(cmdFighter(t, kfmLikeCmd, character.Movelist{Index: 0}))
	if len(got) != 3 {
		t.Errorf("%d lines, want the 3 command lines", len(got))
	}
}

func TestMoveLinesOf_ACharacterWithNeitherShowsNoMoveList(t *testing.T) {
	got := moveLinesOf(cmdFighter(t, "[Command]\nname = \"x\"\ncommand = a\n"))
	want := []MoveLine{{{MoveLeft, []MoveRun{text("No move list")}}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %#v, want %#v", got, want)
	}
}

// moveListFlow is a paused match whose fighters have the given lines.
func moveListFlow(t *testing.T) *Flow {
	t.Helper()
	f := pausedMatch(t)
	rows := make([]string, 10)
	for i := range rows {
		rows[i] = "line" + string(rune('0'+i))
	}
	f.Fighters[0] = cmdFighter(t, "", character.Movelist{Text: strings.Join(rows, "\n")})
	f.Fighters[1] = cmdFighter(t, "", character.Movelist{Text: "only"})
	f.lastChars = [2]Entry{{Name: "a", Display: "Alice"}, {Name: "bob"}}
	f.MoveListWindow = 4
	return f
}

// openMoveListBy pauses as player and opens Move List (4th entry).
func openMoveListBy(f *Flow, player int) {
	f.NavigateBy(NavPause, player)
	for range 3 {
		f.Navigate(NavDown)
	}
	f.Navigate(NavConfirm)
}

func TestMoveList_PauseMenuOffersItBeforeOptions(t *testing.T) {
	f := pausedMatch(t)
	var labels []string
	for _, item := range f.PauseItems() {
		labels = append(labels, item.Label())
	}
	if got := strings.Join(labels, ","); got != "Continue,Restart,Character Change,Move List,Options,Exit" {
		t.Errorf("items = %s", got)
	}
	f.quick = &Setup{}
	labels = labels[:0]
	for _, item := range f.PauseItems() {
		labels = append(labels, item.Label())
	}
	if got := strings.Join(labels, ","); got != "Continue,Restart,Move List,Options,Exit" {
		t.Errorf("quick versus items = %s", got)
	}
}

func TestMoveList_ShowsTheCharacterOfThePlayerWhoPaused(t *testing.T) {
	f := moveListFlow(t)
	openMoveListBy(f, 1)
	if !f.MoveListOpen() || f.MoveListTitle() != "bob" || len(f.MoveListLines()) != 1 || f.PackScreen() != "moveList" {
		t.Fatalf("pad of player 2: open=%v title=%q lines=%d screen=%q", f.MoveListOpen(), f.MoveListTitle(), len(f.MoveListLines()), f.PackScreen())
	}
	g := moveListFlow(t)
	openMoveListBy(g, 0)
	if g.MoveListTitle() != "Alice" || len(g.MoveListLines()) != 10 {
		t.Errorf("keyboard: title=%q lines=%d, want player 1's Alice with 10", g.MoveListTitle(), len(g.MoveListLines()))
	}
}

func TestMoveList_LeftAndRightSwitchToTheOtherSide(t *testing.T) {
	f := moveListFlow(t)
	openMoveListBy(f, 0)
	f.Navigate(NavRight)
	if f.MoveListTitle() != "bob" {
		t.Fatalf("right: title %q", f.MoveListTitle())
	}
	f.Navigate(NavLeft)
	if f.MoveListTitle() != "Alice" {
		t.Fatalf("left: title %q", f.MoveListTitle())
	}
}

func TestMoveList_ScrollsOneLineAtATimeWithinTheWindow(t *testing.T) {
	f := moveListFlow(t) // 10 lines, window 4: the first line shown goes from 0 to 6
	openMoveListBy(f, 0)
	f.Navigate(NavUp)
	if f.MoveListTop() != 0 {
		t.Fatalf("up at the top: %d", f.MoveListTop())
	}
	for range 20 {
		f.Navigate(NavDown)
	}
	if f.MoveListTop() != 6 {
		t.Fatalf("down past the end: top = %d, want 6", f.MoveListTop())
	}
	f.Navigate(NavUp)
	if f.MoveListTop() != 5 {
		t.Fatalf("up: top = %d, want 5", f.MoveListTop())
	}
	f.Navigate(NavRight) // the other side keeps its own position, and is shorter than the window
	f.Navigate(NavDown)
	if f.MoveListTop() != 0 {
		t.Fatalf("a list shorter than the window scrolled to %d", f.MoveListTop())
	}
	f.Navigate(NavLeft)
	if f.MoveListTop() != 5 {
		t.Fatalf("the first side lost its position: %d", f.MoveListTop())
	}
}

func TestMoveList_CancelReturnsToThePauseMenuAndOptionsStillWork(t *testing.T) {
	f := moveListFlow(t)
	openMoveListBy(f, 0)
	f.Navigate(NavCancel)
	if f.MoveListOpen() || !f.Paused || f.PackScreen() != "pause" || f.PauseCursor != 3 {
		t.Fatalf("cancel: open=%v paused=%v screen=%q cursor=%d", f.MoveListOpen(), f.Paused, f.PackScreen(), f.PauseCursor)
	}
	f.Navigate(NavConfirm)
	f.Navigate(NavPause) // Start closes the list too, not the whole menu
	if f.MoveListOpen() || !f.Paused {
		t.Fatalf("start: open=%v paused=%v", f.MoveListOpen(), f.Paused)
	}
	f.Navigate(NavDown)
	f.Navigate(NavConfirm)
	if f.Screen != ScreenOptions {
		t.Errorf("Options after Move List: screen %v", f.Screen)
	}
	f.Tick([2]input.TickInput{})
}

func TestMoveList_QueuesMenuSoundsOnItsOwnScreen(t *testing.T) {
	f := moveListFlow(t)
	openMoveListBy(f, 0)
	f.DrainMenuCues(nil)
	f.Navigate(NavDown)
	f.Navigate(NavUp)
	f.Navigate(NavUp) // at the top: nothing moved, no sound
	f.Navigate(NavCancel)
	got := f.DrainMenuCues(nil)
	want := []MenuCue{{SoundMove, "moveList"}, {SoundMove, "moveList"}, {SoundCancel, "moveList"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cues = %v, want %v", got, want)
	}
}

func TestMoveList_QuickVersusTitleIsTheCharactersNameNotTheCommandLineArgument(t *testing.T) {
	f := moveListFlow(t)
	f.lastChars = [2]Entry{{Name: "kfm"}, {Name: "bob", Display: "Bob the Rival"}}
	f.Fighters[0].Char.Name = "Kung Fu Man"
	f.Fighters[1].Char.Name = "ignored"
	openMoveListBy(f, 0)
	if f.MoveListTitle() != "Kung Fu Man" {
		t.Errorf("title = %q, want the character's name", f.MoveListTitle())
	}
	f.Navigate(NavRight)
	if f.MoveListTitle() != "Bob the Rival" {
		t.Errorf("title = %q, want the entry's display name first", f.MoveListTitle())
	}
}

func TestCommandRuns_SpacesAroundModifiersAndPlusKeepTheIcons(t *testing.T) {
	got := commandRuns("~ D")
	want := []MoveRun{text("~"), icon("down", "D")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("~ D = %v, want %v", got, want)
	}
	got = commandRuns("x + y")
	want = []MoveRun{icon("x", "x"), text("+"), icon("y", "y")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("x + y = %v, want %v", got, want)
	}
}
