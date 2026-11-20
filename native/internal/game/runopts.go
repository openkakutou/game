package game

import (
	"fmt"
	"strconv"
)

// RunOptions are the Ikemen GO command-line options that change the whole
// run rather than one quick versus match (main.lua:153-215, commit 54d4148a).
// They hold for this run only and are never saved.
type RunOptions struct {
	// Volume is -setvolume, the master volume 0-100; -1 when absent or when
	// -nosound is given.
	Volume int
	// HideLifebars is -togglelifebars: no life or power bars in a match.
	HideLifebars bool
	// MaxPowerMode is -maxpowermode: fighters start round 1 with full power
	// (system.go:4426-4428).
	MaxPowerMode bool
	// NoJoy is -nojoy: gamepads are not opened.
	NoJoy bool
	// Fight is -fight, a fight.def used instead of the selected lifebar.
	Fight string
	// AILevel is -ailevel, the CPU level of this run clamped to 1-8
	// (main.lua:154-156); 0 when absent.
	AILevel int
	// Screenpack is -r: Ikemen GO names a motif folder or system.def; here
	// it names a screenpack folder, used instead of the selected one.
	Screenpack string
}

// runFlags are the flags RunOptionsFrom reads.
var runFlags = []string{"-setvolume", "-togglelifebars", "-maxpowermode", "-nojoy", "-fight", "-r", "-ailevel"}

// RunOptionsFrom reads the run options of cl. err names an option whose
// value cannot be used.
func RunOptionsFrom(cl CommandLine) (RunOptions, error) {
	f := cl.Flags
	o := RunOptions{Volume: -1, Fight: f["-fight"], Screenpack: f["-r"]}
	_, o.HideLifebars = f["-togglelifebars"]
	_, o.MaxPowerMode = f["-maxpowermode"]
	_, o.NoJoy = f["-nojoy"]
	if raw, given := f["-ailevel"]; given {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return o, fmt.Errorf("-ailevel %q: must be a level from 1 to 8", raw)
		}
		o.AILevel = min(max(n, 1), 8)
	}
	raw, given := f["-setvolume"]
	if !given {
		return o, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return o, fmt.Errorf("-setvolume %q: must be a volume from 0 to 100", raw)
	}
	if _, mute := f["-nosound"]; mute {
		return o, nil
	}
	o.Volume = min(max(n, 0), 100)
	return o, nil
}
