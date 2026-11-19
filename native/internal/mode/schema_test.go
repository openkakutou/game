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
