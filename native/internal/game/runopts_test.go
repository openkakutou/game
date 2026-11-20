package game

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestRunOptionsFrom_ReadsIkemenGoGlobalFlags(t *testing.T) {
	o, err := RunOptionsFrom(ParseArgs([]string{"-setvolume", "40", "-togglelifebars", "-maxpowermode", "-nojoy", "-fight", "data/fight.def", "-r", "mypack"}))
	if err != nil {
		t.Fatal(err)
	}
	if o.Volume != 40 || !o.HideLifebars || !o.MaxPowerMode || !o.NoJoy || o.Fight != "data/fight.def" || o.Screenpack != "mypack" {
		t.Fatalf("options = %+v", o)
	}
}

func TestRunOptionsFrom_DefaultsAreOff(t *testing.T) {
	o, err := RunOptionsFrom(ParseArgs(nil))
	if err != nil || o.Volume != -1 || o.HideLifebars || o.MaxPowerMode || o.NoJoy || o.Fight != "" || o.Screenpack != "" {
		t.Fatalf("options = %+v, err %v", o, err)
	}
}

func TestRunOptionsFrom_VolumeIsClampedAndNosoundWins(t *testing.T) {
	o, _ := RunOptionsFrom(ParseArgs([]string{"-setvolume", "150"}))
	if o.Volume != 100 {
		t.Errorf("-setvolume 150 = %d, want 100 (Ikemen GO clamps 0-100)", o.Volume)
	}
	o, _ = RunOptionsFrom(ParseArgs([]string{"-setvolume", "50", "-nosound"}))
	if o.Volume != -1 {
		t.Errorf("-nosound: -setvolume ignored, got %d", o.Volume)
	}
	if _, err := RunOptionsFrom(ParseArgs([]string{"-setvolume", "loud"})); err == nil || !strings.Contains(err.Error(), "-setvolume") {
		t.Errorf("err = %v, want one naming -setvolume", err)
	}
}

func TestQuickVersusFrom_GlobalFlagsAreNoLongerUnsupported(t *testing.T) {
	_, _, notices, _ := QuickVersusFrom(ParseArgs([]string{"kfm", "kfm", "-setvolume", "10", "-togglelifebars", "-maxpowermode", "-nojoy", "-fight", "f.def", "-r", "p"}))
	if len(notices) != 0 {
		t.Fatalf("notices = %v, want none", notices)
	}
}

func TestMaxPowerMode_FightersStartRoundOneWithFullPower(t *testing.T) {
	p1, p2 := loadPair(t)
	m, err := NewMatch(p1, p2, closeStage(), Setup{Rounds: 3, MaxPowerMode: true, Start: [2]StartValues{{Power: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if got := m.State.Fighters[i].Power; got != MaxPower {
			t.Errorf("side %d power = %d, want %d: max power mode wins over -p1.power", i, got, MaxPower)
		}
	}
}

func TestLoadLifebarDef_ReadsTheGivenFightDef(t *testing.T) {
	fsys := fstest.MapFS{
		"data/fight.def":         {Data: []byte("[Lifebar]\n")},
		"lifebars/vhd/fight.def": {Data: []byte("[Lifebar]\n")},
	}
	hud, notices := LoadLifebarDef(fsys, "data/fight.def")
	if hud == nil || hud.Def != "data/fight.def" || len(notices) != 0 {
		t.Fatalf("hud = %+v, notices %v", hud, notices)
	}
	hud, _ = LoadLifebarDef(fsys, "vhd/fight.def")
	if hud == nil || hud.Def != "lifebars/vhd/fight.def" {
		t.Fatalf("a path under lifebars/ is found too: %+v", hud)
	}
	hud, notices = LoadLifebarDef(fsys, "gone.def")
	if hud != nil || len(notices) != 1 || !strings.Contains(notices[0], "gone.def") {
		t.Fatalf("missing -fight: hud %+v, notices %v", hud, notices)
	}
}
