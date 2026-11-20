// Package mode loads and validates a game mode definition: a JSONC file
// listing a flow of built-in screens and the match rules (roadmap decision
// 030). The format is described in docs/modes.md and mode.schema.json. The
// native game imports this package; cmd/mode-wasm exposes the same Parse to
// the web build.
package mode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// FormatVersion is the only formatVersion this loader reads.
const FormatVersion = 1

// Screen is a built-in screen type a mode's flow can list.
type Screen string

// The built-in screen types a flow can use. Title, main menu, options,
// pause and move list are the runtime's own, outside any mode's flow.
const (
	ScreenCharacterSelect Screen = "characterSelect"
	ScreenStageSelect     Screen = "stageSelect"
	ScreenMatchSetup      Screen = "matchSetup"
	ScreenVersus          Screen = "versus"
	ScreenMatch           Screen = "match"
	ScreenResults         Screen = "results"
	ScreenContinue        Screen = "continue"
	ScreenVictory         Screen = "victory"
)

// Screens lists every Screen in the order the spec documents them.
var Screens = []Screen{ScreenCharacterSelect, ScreenStageSelect, ScreenMatchSetup, ScreenVersus, ScreenMatch, ScreenResults, ScreenContinue, ScreenVictory}

// Control is who drives a side.
type Control string

// The controls a side can have.
const (
	ControlHuman      Control = "human"
	ControlCPU        Control = "cpu"
	ControlHumanOrCPU Control = "humanOrCpu" // chosen on the match setup screen
)

// Controls lists every Control.
var Controls = []Control{ControlHuman, ControlCPU, ControlHumanOrCPU}

// OpponentsKind is how the opponents of a run are picked.
type OpponentsKind string

// The opponent kinds.
const (
	OpponentsChosen OpponentsKind = "chosen" // picked on the select screen
	OpponentsLadder OpponentsKind = "ladder" // Count CPU opponents in a row
)

// OpponentsKinds lists every OpponentsKind.
var OpponentsKinds = []OpponentsKind{OpponentsChosen, OpponentsLadder}

// Outcome is what follows a won or lost match.
type Outcome string

// The outcomes.
const (
	OutcomeResults      Outcome = "results"      // the flow's results screen
	OutcomeNextOpponent Outcome = "nextOpponent" // next ladder opponent, victory after the last
	OutcomeContinue     Outcome = "continue"     // continue screen, then retry or game over
	OutcomeGameOver     Outcome = "gameOver"
)

// Outcomes lists every Outcome.
var Outcomes = []Outcome{OutcomeResults, OutcomeNextOpponent, OutcomeContinue, OutcomeGameOver}

// PauseEntry is a pause menu entry a mode can ask the game to add.
type PauseEntry string

// The pause entries the game offers a mode.
const (
	PauseResetPositions PauseEntry = "resetPositions"
	PauseDummy          PauseEntry = "dummy"
)

// PauseEntries lists every PauseEntry.
var PauseEntries = []PauseEntry{PauseResetPositions, PauseDummy}

// Sides are the side names a characterSelect screen can pick for.
var Sides = []string{"p1", "p2"}

// Mode is a validated mode definition.
type Mode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Flow  []Step `json:"flow"`
	Rules Rules  `json:"rules"`
	// Pause lists the entries the pause menu gets besides the game's own, in
	// this order.
	Pause []PauseEntry `json:"pause,omitempty"`
}

// Step is one screen of the flow. Sides is characterSelect's "sides"
// parameter: the sides that pick there.
type Step struct {
	Screen Screen   `json:"screen"`
	Sides  []string `json:"sides,omitempty"`
}

// Rules are the match rules.
type Rules struct {
	Sides            [2]Side   `json:"sides"`
	Rounds           int       `json:"rounds"`
	TimeLimitSeconds int       `json:"timeLimitSeconds"` // 0: no time limit
	Opponents        Opponents `json:"opponents"`
	OnWin            Outcome   `json:"onWin"`
	OnLose           Outcome   `json:"onLose"`
	Roster           Roster    `json:"roster"` // the zero Roster shows everything
	// Training turns on the training behaviour: life refills and no round
	// ends by KO.
	Training bool `json:"training,omitempty"`
}

// Roster says which characters and stages the mode shows (roadmap decision
// 034).
type Roster struct {
	Characters CharacterRoster `json:"characters"`
	Stages     StageRoster     `json:"stages"`
}

// CharacterRoster filters the characters: Select is what the select screens
// show, Opponents the pool a CPU opponent can come from.
type CharacterRoster struct {
	Select    Filter `json:"select"`
	Opponents Filter `json:"opponents"`
}

// StageRoster filters the stages the select screens show.
type StageRoster struct {
	Select Filter `json:"select"`
}

// Filter keeps the names in Include (every name when Include is nil, none
// when it is empty), less the names in Exclude. Names are folder or file
// names, matched ignoring case.
type Filter struct {
	Include []string `json:"include,omitzero"` // omitzero: an empty list stays, only nil is dropped
	Exclude []string `json:"exclude,omitempty"`
}

// Allows is whether name passes the filter.
func (f Filter) Allows(name string) bool {
	named := func(list []string) bool {
		return slices.ContainsFunc(list, func(n string) bool { return strings.EqualFold(n, name) })
	}
	if f.Include != nil && !named(f.Include) {
		return false
	}
	return !named(f.Exclude)
}

// Side is one side's rule.
type Side struct {
	Control    Control `json:"control"`
	Characters int     `json:"characters"`
}

// Opponents is the opponents rule; Count is set for a ladder, and Final, the
// folder name of a character fought after the Count ladder opponents, may be.
type Opponents struct {
	Kind  OpponentsKind `json:"kind"`
	Count int           `json:"count,omitempty"`
	Final string        `json:"final,omitempty"`
}

// Error is a mode that cannot be used: File, the JSON path of the Field
// ("" for the whole file) and the Reason.
type Error struct {
	File   string `json:"file"`
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (e *Error) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.File, e.Reason)
	}
	return fmt.Sprintf("%s: %s: %s", e.File, e.Field, e.Reason)
}

// Load reads and validates the mode file at path.
func Load(path string) (Mode, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Mode{}, &Error{File: path, Reason: err.Error()}
	}
	return Parse(path, src)
}

// Parse validates src, the JSONC text of the mode file named file.
func Parse(file string, src []byte) (Mode, error) {
	dec := json.NewDecoder(bytes.NewReader(stripJSONC(src)))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return Mode{}, &Error{File: file, Reason: "not valid JSONC: " + err.Error()}
	}
	v := validator{file: file}
	m := v.mode(root)
	if v.err != nil {
		return Mode{}, v.err
	}
	return m, nil
}

// validator walks the decoded JSON, keeping the first error.
type validator struct {
	file string
	err  *Error
}

func (v *validator) fail(field, reason string) {
	if v.err == nil {
		v.err = &Error{File: v.file, Field: field, Reason: reason}
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func index(path string, i int) string { return path + "[" + strconv.Itoa(i) + "]" }

// object returns x as an object holding only allowed keys, each of required
// present.
func (v *validator) object(path string, x any, required []string, allowed ...string) map[string]any {
	obj, ok := x.(map[string]any)
	if !ok {
		v.fail(path, "must be an object")
		return nil
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if !slices.Contains(allowed, k) && !slices.Contains(required, k) {
			v.fail(join(path, k), "unknown field")
		}
	}
	for _, k := range required {
		if _, ok := obj[k]; !ok {
			v.fail(join(path, k), "required field is missing")
		}
	}
	return obj
}

func (v *validator) str(path string, x any) string {
	s, ok := x.(string)
	if !ok || s == "" {
		v.fail(path, "must be a non-empty string")
	}
	return s
}

func (v *validator) integer(path string, x any, minimum int) int {
	n, ok := x.(json.Number)
	if !ok {
		v.fail(path, "must be an integer")
		return 0
	}
	i, err := strconv.Atoi(n.String())
	if err != nil {
		v.fail(path, "must be an integer")
		return 0
	}
	if i < minimum {
		v.fail(path, fmt.Sprintf("must be at least %d", minimum))
	}
	return i
}

func (v *validator) boolean(path string, x any) bool {
	b, ok := x.(bool)
	if !ok {
		v.fail(path, "must be a boolean")
	}
	return b
}

func oneOf[T ~string](v *validator, path string, x any, values []T) T {
	s, _ := x.(string)
	if slices.Contains(values, T(s)) {
		return T(s)
	}
	v.fail(path, fmt.Sprintf("%q is not one of %v", s, values))
	return ""
}

func (v *validator) mode(x any) Mode {
	obj := v.object("", x, []string{"formatVersion", "id", "name", "flow", "rules"}, "$schema", "pause")
	if obj == nil {
		return Mode{}
	}
	if fv, ok := obj["formatVersion"]; ok && v.integer("formatVersion", fv, 0) != FormatVersion {
		v.fail("formatVersion", fmt.Sprintf("must be %d", FormatVersion))
	}
	m := Mode{}
	if id, ok := obj["id"]; ok {
		m.ID = v.str("id", id)
	}
	if name, ok := obj["name"]; ok {
		m.Name = v.str("name", name)
	}
	if flow, ok := obj["flow"]; ok {
		m.Flow = v.flow(flow)
	}
	if rules, ok := obj["rules"]; ok {
		m.Rules = v.rules(rules)
	}
	if pause, ok := obj["pause"]; ok {
		m.Pause = v.pause(pause)
	}
	return m
}

// pause validates the pause entries: only the ones the game offers, each once.
func (v *validator) pause(x any) []PauseEntry {
	list, ok := x.([]any)
	if !ok {
		v.fail("pause", "must be an array of pause entries")
		return nil
	}
	entries := make([]PauseEntry, 0, len(list))
	for i, item := range list {
		path := index("pause", i)
		entry := oneOf(v, path, item, PauseEntries)
		if slices.Contains(entries, entry) {
			v.fail(path, fmt.Sprintf("%q is listed twice", entry))
		}
		entries = append(entries, entry)
	}
	return entries
}

func (v *validator) flow(x any) []Step {
	list, ok := x.([]any)
	if !ok || len(list) == 0 {
		v.fail("flow", "must be a non-empty array of screens")
		return nil
	}
	steps := make([]Step, 0, len(list))
	for i, item := range list {
		path := index("flow", i)
		obj := v.object(path, item, []string{"screen"}, "params")
		if obj == nil {
			continue
		}
		step := Step{Screen: oneOf(v, join(path, "screen"), obj["screen"], Screens)}
		if params, ok := obj["params"]; ok {
			step.Sides = v.params(join(path, "params"), step.Screen, params)
		}
		if step.Screen == ScreenCharacterSelect && step.Sides == nil {
			v.fail(join(path, "params.sides"), "is required on a characterSelect screen")
		}
		// The results screen shows the match just played.
		if step.Screen == ScreenResults && !slices.ContainsFunc(steps, func(s Step) bool { return s.Screen == ScreenMatch }) {
			v.fail(join(path, "screen"), `"results" must come after a "match" screen`)
		}
		steps = append(steps, step)
	}
	if !slices.ContainsFunc(steps, func(s Step) bool { return s.Screen == ScreenMatch }) {
		v.fail("flow", `must include a "match" screen`)
	}
	return steps
}

// params validates a step's parameters: only characterSelect takes one,
// "sides".
func (v *validator) params(path string, screen Screen, x any) []string {
	var allowed []string
	if screen == ScreenCharacterSelect {
		allowed = []string{"sides"}
	}
	obj := v.object(path, x, nil, allowed...)
	raw, ok := obj["sides"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		v.fail(join(path, "sides"), "must be a non-empty array")
		return nil
	}
	sides := make([]string, 0, len(list))
	for i, s := range list {
		sides = append(sides, oneOf(v, index(join(path, "sides"), i), s, Sides))
	}
	return sides
}

func (v *validator) rules(x any) Rules {
	obj := v.object("rules", x, []string{"sides", "rounds", "timeLimitSeconds", "opponents", "onWin", "onLose"}, "roster", "training")
	if obj == nil {
		return Rules{}
	}
	r := Rules{}
	if list, ok := obj["sides"].([]any); ok && len(list) == 2 {
		for i, s := range list {
			path := index("rules.sides", i)
			side := v.object(path, s, []string{"control", "characters"})
			if side == nil {
				continue
			}
			r.Sides[i].Control = oneOf(v, join(path, "control"), side["control"], Controls)
			r.Sides[i].Characters = v.integer(join(path, "characters"), side["characters"], 1)
		}
	} else if _, present := obj["sides"]; present {
		v.fail("rules.sides", "must be an array of exactly 2 sides")
	}
	if rounds, ok := obj["rounds"]; ok {
		r.Rounds = v.integer("rules.rounds", rounds, 1)
		if r.Rounds%2 == 0 {
			v.fail("rules.rounds", "must be odd (best of N)")
		}
	}
	if limit, ok := obj["timeLimitSeconds"]; ok {
		r.TimeLimitSeconds = v.integer("rules.timeLimitSeconds", limit, 0)
	}
	if opp, ok := obj["opponents"]; ok {
		r.Opponents = v.opponents(opp)
	}
	if win, ok := obj["onWin"]; ok {
		r.OnWin = oneOf(v, "rules.onWin", win, Outcomes)
	}
	if lose, ok := obj["onLose"]; ok {
		r.OnLose = oneOf(v, "rules.onLose", lose, Outcomes)
	}
	if roster, ok := obj["roster"]; ok {
		r.Roster = v.roster(roster)
	}
	if training, ok := obj["training"]; ok {
		r.Training = v.boolean("rules.training", training)
	}
	return r
}

func (v *validator) roster(x any) Roster {
	const path = "rules.roster"
	obj := v.object(path, x, nil, "characters", "stages")
	r := Roster{}
	if chars, ok := obj["characters"]; ok {
		cpath := join(path, "characters")
		cobj := v.object(cpath, chars, nil, "select", "opponents")
		r.Characters.Select = v.filter(join(cpath, "select"), cobj, "select")
		r.Characters.Opponents = v.filter(join(cpath, "opponents"), cobj, "opponents")
	}
	if stages, ok := obj["stages"]; ok {
		spath := join(path, "stages")
		sobj := v.object(spath, stages, nil, "select")
		r.Stages.Select = v.filter(join(spath, "select"), sobj, "select")
	}
	return r
}

// filter reads the filter under key of parent, the zero Filter when absent.
func (v *validator) filter(path string, parent map[string]any, key string) Filter {
	raw, ok := parent[key]
	if !ok {
		return Filter{}
	}
	obj := v.object(path, raw, nil, "include", "exclude")
	f := Filter{}
	if list, ok := obj["include"]; ok {
		f.Include = v.names(join(path, "include"), list)
	}
	if list, ok := obj["exclude"]; ok {
		f.Exclude = v.names(join(path, "exclude"), list)
	}
	return f
}

// names reads an array of non-empty strings; an empty array is an empty,
// non-nil list.
func (v *validator) names(path string, x any) []string {
	list, ok := x.([]any)
	if !ok {
		v.fail(path, "must be an array of strings")
		return nil
	}
	out := make([]string, 0, len(list))
	for i, item := range list {
		out = append(out, v.str(index(path, i), item))
	}
	return out
}

func (v *validator) opponents(x any) Opponents {
	obj := v.object("rules.opponents", x, []string{"kind"}, "count", "final")
	if obj == nil {
		return Opponents{}
	}
	o := Opponents{Kind: oneOf(v, "rules.opponents.kind", obj["kind"], OpponentsKinds)}
	count, hasCount := obj["count"]
	switch {
	case o.Kind == OpponentsLadder && !hasCount:
		v.fail("rules.opponents.count", "required field is missing for a ladder")
	case o.Kind == OpponentsLadder:
		o.Count = v.integer("rules.opponents.count", count, 1)
	case hasCount:
		v.fail("rules.opponents.count", `only a "ladder" takes a count`)
	}
	final, hasFinal := obj["final"]
	switch {
	case hasFinal && o.Kind != OpponentsLadder:
		v.fail("rules.opponents.final", `only a "ladder" takes a final`)
	case hasFinal:
		o.Final = v.str("rules.opponents.final", final)
	}
	return o
}
