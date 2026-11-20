package game

import (
	"fmt"
	"slices"
	"strings"

	"github.com/openkakutou/game/native/internal/mode"
)

// VisibleChars are the roster indexes of the characters the mode's select
// screens show, in roster order. The character select's cursor is a position
// in this list; Picks hold the roster indexes it maps to.
func (f *Flow) VisibleChars() []int { return f.charView }

// VisibleStages are the roster indexes of the stages the mode's select
// screen shows, in roster order. The stage select's cursor is a position in
// this list; StageNo holds the roster index it maps to.
func (f *Flow) VisibleStages() []int { return f.stageView }

// OpponentPool are the roster indexes of the characters a CPU opponent can
// come from in the mode, in roster order: the mode's roster.characters
// opponents filter, independent of what the select screens show.
func (f *Flow) OpponentPool() []int {
	return allowedIndexes(f.Chars, f.Mode.Rules.Roster.Characters.Opponents)
}

// applyRoster computes the visible characters and stages of the mode being
// played, again after the rosters change order.
func (f *Flow) applyRoster() {
	r := f.Mode.Rules.Roster
	f.charView = allowedIndexes(f.Chars, r.Characters.Select)
	f.stageView = allowedIndexes(f.Stages, r.Stages.Select)
}

// allowedIndexes are the indexes of the entries filter allows.
func allowedIndexes(entries []Entry, filter mode.Filter) []int {
	out := make([]int, 0, len(entries))
	for i, e := range entries {
		if filter.Allows(e.Name) {
			out = append(out, i)
		}
	}
	return out
}

// viewPos is the position in view of the roster index i; 0 when i is not
// shown (a side that has not picked yet points at the first roster entry).
func viewPos(view []int, i int) int {
	at := slices.Index(view, i)
	if at >= 0 {
		return at
	}
	return 0
}

// emptyRosterMessage is the error text of a mode whose filter shows no
// character or no stage, "" when both lists have something to show.
func (f *Flow) emptyRosterMessage() string {
	name := f.Mode.Name
	if len(f.charView) == 0 {
		return fmt.Sprintf("The %q mode shows no character: roster.characters.select leaves none of the installed characters.", name)
	}
	if len(f.stageView) == 0 {
		return fmt.Sprintf("The %q mode shows no stage: roster.stages.select leaves none of the installed stages.", name)
	}
	return ""
}

// RosterNotices lists, for each mode, the names its roster filters give that
// no installed character or stage has: not fatal, the name just matches
// nothing. The host prints them once at startup.
func RosterNotices(modes []mode.Mode, chars, stages []Entry) []string {
	var notices []string
	for _, m := range modes {
		r := m.Rules.Roster
		lists := []struct {
			path    string
			names   []string
			noun    string
			entries []Entry
		}{
			{"rules.roster.characters.select.include", r.Characters.Select.Include, "character", chars},
			{"rules.roster.characters.select.exclude", r.Characters.Select.Exclude, "character", chars},
			{"rules.roster.characters.opponents.include", r.Characters.Opponents.Include, "character", chars},
			{"rules.roster.characters.opponents.exclude", r.Characters.Opponents.Exclude, "character", chars},
			{"rules.roster.stages.select.include", r.Stages.Select.Include, "stage", stages},
			{"rules.roster.stages.select.exclude", r.Stages.Select.Exclude, "stage", stages},
		}
		for _, l := range lists {
			for _, name := range l.names {
				if !slices.ContainsFunc(l.entries, func(e Entry) bool { return strings.EqualFold(e.Name, name) }) {
					notices = append(notices, fmt.Sprintf("mode %q: %s names %q, which is not an installed %s", m.Name, l.path, name, l.noun))
				}
			}
		}
	}
	return notices
}
