package lifebar

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `; test lifebar
[Info]
localcoord = 1280,720 ; HD

[Files]
sff = fight.sff

[Lifebar]
p1.pos = 10,20
p1.bg0.anim = 170
p1.bg1.spr = 11,4
p1.bg1.layerno = 5
p1.mid.spr = 1,1
p1.front.spr = 1,2
p1.front.facing = -1
p1.range.x = 563,135;428
p1.pos = 99,99
p2.range.x = 717,1145

[Powerbar]
p1.range.x = 0,99

[Name]
p1.pos = 30,40
p1.name.font = 3,0,1
p1.name.offset = 2.5,0

[Time]
pos = 160,23
counter.font = 5

[Begin Action 170]
11,3, 0,0, 10
11,4, 0,0, 5
`

func TestParseReadsBarsAsIkemenGo(t *testing.T) {
	lb, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if lb.Localcoord != [2]int{1280, 720} || lb.Sprites != "fight.sff" {
		t.Fatalf("info/files = %v %q", lb.Localcoord, lb.Sprites)
	}
	p1 := lb.Life[0]
	if p1.Pos != [2]int{10, 20} {
		t.Errorf("first occurrence of a key wins: pos = %v", p1.Pos)
	}
	if p1.RangeX != [2]int{563, 135} {
		t.Errorf("comment after value is ignored: range = %v", p1.RangeX)
	}
	if p1.Bg[0].Anim != 170 || !p1.Bg[0].Shown() {
		t.Errorf("bg0 anim = %+v", p1.Bg[0])
	}
	if p1.Bg[1].Sprite != [2]int{11, 4} || p1.Bg[1].Layer != 2 {
		t.Errorf("bg1 = %+v (layerno is capped at 2)", p1.Bg[1])
	}
	if p1.Front.Facing != -1 || p1.Mid.Facing != 1 {
		t.Errorf("facing front %d mid %d", p1.Front.Facing, p1.Mid.Facing)
	}
	if p1.Top.Shown() || p1.Bg[2].Shown() {
		t.Error("undeclared elements must not be shown")
	}
	if lb.Life[1].RangeX != [2]int{717, 1145} || lb.Power[0].RangeX != [2]int{0, 99} {
		t.Errorf("p2/power range = %v %v", lb.Life[1].RangeX, lb.Power[0].RangeX)
	}
	if lb.Name[0].Pos != [2]int{30, 40} || lb.Name[0].Offset != [2]float64{2.5, 0} || lb.Name[0].Font != 3 || lb.Name[0].Align != 1 {
		t.Errorf("name = %+v", lb.Name[0])
	}
	if !lb.Time.Shown() || lb.Time.Pos != [2]int{160, 23} {
		t.Errorf("time = %+v", lb.Time)
	}
}

func TestParseDefaultsAndMissingAnim(t *testing.T) {
	lb, err := Parse("[lifebar]\np1.front.anim = 999\n")
	if err != nil {
		t.Fatal(err)
	}
	if lb.Localcoord != DefaultLocalcoord {
		t.Errorf("localcoord = %v", lb.Localcoord)
	}
	if lb.Life[0].Front.Shown() {
		t.Error("an anim missing from the def shows nothing (common.go:1537)")
	}
	if lb.Power[0].Front.Shown() || lb.Time.Shown() {
		t.Error("missing sections give empty bars")
	}
}

func TestParseRejectsTextWithoutLifebar(t *testing.T) {
	if _, err := Parse("[Info]\nname = x\n"); err == nil {
		t.Fatal("want an error without [Lifebar]")
	}
}

func TestFrameFollowsTheAnimation(t *testing.T) {
	lb, _ := Parse(sample)
	bg := lb.Life[0].Bg[0]
	cases := map[int][2]int{0: {11, 3}, 9: {11, 3}, 10: {11, 4}, 15: {11, 3}, 27: {11, 4}}
	for tick, want := range cases {
		fr, ok := lb.Frame(bg, tick)
		if !ok || [2]int{fr.Group, fr.Image} != want {
			t.Errorf("tick %d: %v %v, want %v", tick, fr, ok, want)
		}
	}
	if fr, ok := lb.Frame(lb.Life[0].Bg[1], 500); !ok || fr.Group != 11 || fr.Image != 4 {
		t.Errorf("spr element = %v %v", fr, ok)
	}
	if _, ok := lb.Frame(lb.Life[0].Top, 0); ok {
		t.Error("an unset element has no frame")
	}
}

func TestFillSpanMatchesCalcBarFillRect(t *testing.T) {
	// Ascending: starts at pos+r0, grows right.
	if s, n := FillSpan(0, [2]int{0, 99}, 1, 1); s != 0 || n != 100 {
		t.Errorf("full ascending = %d,%d", s, n)
	}
	if s, n := FillSpan(10, [2]int{0, 99}, 1, 0.5); s != 10 || n != 50 {
		t.Errorf("half ascending = %d,%d", s, n)
	}
	// Descending (P1 drains toward its outer edge): ends at pos+r0+1.
	if s, n := FillSpan(0, [2]int{563, 135}, 0.25, 0.5); s != 87 || n != 54 {
		t.Errorf("half descending = %d,%d", s, n)
	}
	if _, n := FillSpan(0, [2]int{0, 99}, 1, 0); n != 0 {
		t.Errorf("empty = %d", n)
	}
}

func TestScaleFromLocalcoord(t *testing.T) {
	lb := &Lifebar{Localcoord: [2]int{1280, 720}}
	if s := lb.Scale(320); s != 0.25 {
		t.Errorf("scale = %v", s)
	}
	lb.Localcoord = [2]int{0, 0}
	if s := lb.Scale(320); s != 1 {
		t.Errorf("zero localcoord scale = %v", s)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverListsValidFoldersAndReportsBrokenOnes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "zeta", "FIGHT.DEF"), "[Lifebar]\n")
	write(t, filepath.Join(dir, "default", "fight.def"), "[Lifebar]\n")
	write(t, filepath.Join(dir, "broken", "fight.def"), "[Info]\n")
	write(t, filepath.Join(dir, "empty", "readme.txt"), "x")
	write(t, filepath.Join(dir, "loose.def"), "[Lifebar]\n")

	entries, problems := Discover(os.DirFS(dir), ".")
	if len(entries) != 2 || entries[0].Name != "default" || entries[1].Name != "zeta" {
		t.Fatalf("entries = %+v", entries)
	}
	if len(problems) != 2 {
		t.Fatalf("problems = %v", problems)
	}
	if entries, problems := Discover(os.DirFS(dir), "nope"); entries != nil || problems != nil {
		t.Error("a missing lifebars/ folder is simply empty")
	}
}

func TestPick(t *testing.T) {
	entries := []Entry{{Name: "default"}, {Name: "vhd"}}
	if e, ok := Pick(entries, "VHD"); !ok || e.Name != "vhd" {
		t.Errorf("pick = %v %v", e, ok)
	}
	if e, ok := Pick(entries, "gone"); !ok || e.Name != "default" {
		t.Errorf("fallback = %v %v", e, ok)
	}
	if _, ok := Pick(entries[1:], "gone"); ok {
		t.Error("no selection and no default: nothing")
	}
}

func TestMeterTrailsThenCatchesUp(t *testing.T) {
	m := NewMeter()
	m.Step(1, false, true)
	m.Step(0.5, true, true) // hit: the front halves its gap, mid holds
	if m.Top != 0.75 || m.Mid != 1 {
		t.Fatalf("on hit top %v mid %v", m.Top, m.Mid)
	}
	for range 29 {
		m.Step(0.5, false, true)
	}
	if m.Mid != 1 {
		t.Fatalf("mid must wait the 30-tick delay, got %v", m.Mid)
	}
	for range 200 {
		m.Step(0.5, false, true)
	}
	if m.Top != 0.5 || m.Mid < 0.5 || m.Mid > 0.51 {
		t.Fatalf("after catching up top %v mid %v", m.Top, m.Mid)
	}
}

func TestMeterRefillFollowsAtOnce(t *testing.T) {
	m := NewMeter()
	m.Top, m.Mid = 0.2, 0.2
	m.Step(0.6, false, false)
	if m.Top != 0.6 || m.Mid <= 0.2 {
		t.Fatalf("refill top %v mid %v", m.Top, m.Mid)
	}
}
