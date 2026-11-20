package mode

import (
	"encoding/json"
	"slices"
	"testing"
)

// The schema and this package state the same value lists twice; this test
// fails when one changes without the other.
func TestSchema_EnumsMatchThePackagesValues(t *testing.T) {
	var s map[string]any
	if err := json.Unmarshal(mustRead(t, "mode.schema.json"), &s); err != nil {
		t.Fatal(err)
	}
	at := func(path ...string) []string {
		var node any = s
		for _, k := range path {
			node = node.(map[string]any)[k]
		}
		var out []string
		for _, v := range node.([]any) {
			out = append(out, v.(string))
		}
		return out
	}
	props := []string{"properties"}
	flowItem := append(slices.Clone(props), "flow", "items", "properties")
	rules := append(slices.Clone(props), "rules", "properties")
	check := func(name string, got []string, want []string) {
		if !slices.Equal(got, want) {
			t.Errorf("%s: schema %v, package %v", name, got, want)
		}
	}
	check("screens", at(append(flowItem, "screen", "enum")...), strs(Screens))
	check("controls", at(append(rules, "sides", "items", "properties", "control", "enum")...), strs(Controls))
	check("opponents", at(append(rules, "opponents", "properties", "kind", "enum")...), strs(OpponentsKinds))
	check("pause", at("properties", "pause", "items", "enum"), strs(PauseEntries))
	check("onWin", at(append(rules, "onWin", "enum")...), strs(Outcomes))
	check("onLose", at(append(rules, "onLose", "enum")...), strs(Outcomes))
	sides := s["properties"].(map[string]any)["flow"].(map[string]any)["items"].(map[string]any)["allOf"].([]any)[0].(map[string]any)["then"].(map[string]any)["properties"].(map[string]any)["params"].(map[string]any)["properties"].(map[string]any)["sides"].(map[string]any)["items"].(map[string]any)["enum"]
	var got []string
	for _, v := range sides.([]any) {
		got = append(got, v.(string))
	}
	check("sides", got, Sides)
}

func strs[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// The roster rule's object keys, stated in the schema and in this package.
func TestSchema_RosterKeysMatchThePackage(t *testing.T) {
	var s map[string]any
	if err := json.Unmarshal(mustRead(t, "mode.schema.json"), &s); err != nil {
		t.Fatal(err)
	}
	keys := func(node map[string]any) []string {
		var out []string
		for k := range node["properties"].(map[string]any) {
			out = append(out, k)
		}
		slices.Sort(out)
		return out
	}
	child := func(node map[string]any, k string) map[string]any {
		return node["properties"].(map[string]any)[k].(map[string]any)
	}
	roster := child(child(s, "rules"), "roster")
	if got := keys(roster); !slices.Equal(got, []string{"characters", "stages"}) {
		t.Errorf("roster keys = %v", got)
	}
	if got := keys(child(roster, "characters")); !slices.Equal(got, []string{"opponents", "select"}) {
		t.Errorf("roster.characters keys = %v", got)
	}
	if got := keys(child(roster, "stages")); !slices.Equal(got, []string{"select"}) {
		t.Errorf("roster.stages keys = %v", got)
	}
	for _, ref := range []map[string]any{child(child(roster, "characters"), "select"), child(child(roster, "characters"), "opponents"), child(child(roster, "stages"), "select")} {
		if ref["$ref"] != "#/$defs/filter" {
			t.Errorf("a roster list refers to %v, want #/$defs/filter", ref["$ref"])
		}
	}
	filter := s["$defs"].(map[string]any)["filter"].(map[string]any)
	if got := keys(filter); !slices.Equal(got, []string{"exclude", "include"}) {
		t.Errorf("filter keys = %v", got)
	}
}

// The opponents rule's keys, stated in the schema and in this package.
func TestSchema_OpponentsKeysMatchThePackage(t *testing.T) {
	var s map[string]any
	if err := json.Unmarshal(mustRead(t, "mode.schema.json"), &s); err != nil {
		t.Fatal(err)
	}
	opponents := s["properties"].(map[string]any)["rules"].(map[string]any)["properties"].(map[string]any)["opponents"].(map[string]any)
	var got []string
	for k := range opponents["properties"].(map[string]any) {
		got = append(got, k)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"count", "final", "kind"}) {
		t.Errorf("opponents keys = %v", got)
	}
}
