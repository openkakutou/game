package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// CommandLine is the program's arguments, parsed as Ikemen GO parses them
// (processCommandLine, src/main.go:241-361, commit 54d4148a): Flags maps each
// "-flag" to its value ("true" for a flag given no value), the first two
// names before any flag become -p1 and -p2, and Help is set by -h, -? or -%.
type CommandLine struct {
	Flags map[string]string
	Help  bool
}

// boolFlags take no value (main.go:260-269).
var boolFlags = []string{"-windowed", "-togglelifebars", "-maxpowermode", "-debug", "-debugdump", "-nojoy", "-nomusic", "-nosound"}

var (
	playerFlag = regexp.MustCompile(`^-p[0-9]+$`)
	dotSuffix  = regexp.MustCompile(`^\.[A-Za-z][A-Za-z0-9]*$`)
	helpFlag   = regexp.MustCompile(`^-[h%?]$`)
)

// ParseArgs parses the program's arguments, without the program name.
func ParseArgs(raw []string) CommandLine {
	cl := CommandLine{Flags: map[string]string{}}
	// PowerShell can split "-p1.ai" into "-p1" ".ai" (main.go:244-259).
	args := make([]string, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if playerFlag.MatchString(raw[i]) && i+1 < len(raw) && dotSuffix.MatchString(raw[i+1]) {
			args = append(args, raw[i]+raw[i+1])
			i++
			continue
		}
		args = append(args, raw[i])
	}
	key := ""
	player := 1
	flagsSeen := false
	for _, a := range args {
		_, numErr := strconv.ParseFloat(a, 64)
		isFlag := strings.HasPrefix(a, "-")
		switch {
		case key != "" && (numErr == nil || !isFlag):
			cl.Flags[key] = a
			key = ""
		case isFlag:
			flagsSeen = true
			if helpFlag.MatchString(a) {
				cl.Help = true
				continue
			}
			if slices.Contains(boolFlags, a) {
				cl.Flags[a] = "true"
				continue
			}
			cl.Flags[a] = ""
			key = a
		case !flagsSeen && player <= 2:
			cl.Flags[fmt.Sprintf("-p%d", player)] = a
			player++
		}
	}
	if key != "" {
		cl.Flags[key] = "true"
	}
	return cl
}

// HelpText lists the options, in Ikemen GO's layout (main.go:281-322):
// first the ones this game supports, then OpenKakutou's own.
const HelpText = `OpenKakutou command line options (case sensitive):
-h -?                   Help

Quick VS options (start one match without menus, then quit):
<p1> <p2>               The first two names are player 1 and player 2
-p1 <char> -p2 <char>   Loads player 1 / player 2, eg. -p1 kfm
-p<n>.ai <level>        Player n (1 or 2) is played by the CPU when level > 0
-p<n>.life <life>       Sets player n's life to <life> (1-1000) for round 1
-p<n>.power <power>     Sets player n's power to <power> (0-3000) for round 1
-time <num>             Round time in seconds (-1 to disable)
-rounds <num>           Rounds to win the match
-s <stagename>          Loads stage <stagename>

Other options:
-windowed               Accepted: the game always runs in a window
-width <num>            Sets the window width
-height <num>           Sets the window height
-nomusic                Disables music
-nosound                Disables all sound effects and music

OpenKakutou only:
-content <folder>       Folder holding chars/, stages/ and common/
-frames <num>           Exits after this many frames (smoke tests)
-capture <file.png>     Saves the last frame (with -frames)

Characters are found as Ikemen GO finds them: "kfm" is chars/kfm/kfm.def,
"kfm/kfm2.def" is chars/kfm/kfm2.def; stages likewise under stages/.
See docs/command-line.md for the Ikemen GO options not supported yet.`

// QuickVersus is a match the command line starts directly (Ikemen GO's
// quick VS, main.f_commandLine, main.lua:1023-1256).
type QuickVersus struct {
	P1, P2 string // character declarations, resolved by ResolveCharDef
	Stage  string // stage declaration; "" for the default stage
	Wins   int    // rounds to win (-rounds)
	Time   int    // round time in seconds, -1 for none (-time)
	CPU    [2]bool
	Start  [2]StartValues
}

// Ikemen GO's defaults: Options.Match.Wins 2 and Options.Time 99 (its
// default config); quick VS reads them when -rounds and -time are absent.
const (
	defaultQuickWins = 2
	defaultQuickTime = 99
)

// Setup is the match setup quick versus plays: best of 2×Wins-1 rounds.
func (q QuickVersus) Setup() Setup {
	limit := q.Time
	if limit < 0 {
		limit = 0
	}
	return Setup{Rounds: 2*q.Wins - 1, TimeLimitSeconds: limit, CPU: q.CPU, Start: q.Start}
}

// handledFlags are the flags this game reads.
var handledFlags = []string{
	"-p1", "-p2", "-p1.ai", "-p2.ai", "-p1.life", "-p2.life", "-p1.power", "-p2.power",
	"-time", "-rounds", "-s", "-windowed", "-width", "-height", "-nomusic", "-nosound",
	"-content", "-frames", "-capture",
}

// QuickVersusFrom reads the quick versus match from cl. ok is false when cl
// does not ask for one (Ikemen GO needs -p1 and -p2 and no -loadmotif,
// main.lua:1255). notices name the options given that this game ignores; err
// names an option whose value cannot be used.
func QuickVersusFrom(cl CommandLine) (q QuickVersus, ok bool, notices []string, err error) {
	f := cl.Flags
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if !slices.Contains(handledFlags, k) {
			notices = append(notices, fmt.Sprintf("%s is not supported yet, ignored", k))
		}
	}
	_, hasP1 := f["-p1"]
	_, hasP2 := f["-p2"]
	_, loadMotif := f["-loadmotif"]
	if !hasP1 || !hasP2 || loadMotif {
		return QuickVersus{}, false, notices, nil
	}
	q = QuickVersus{P1: f["-p1"], P2: f["-p2"], Stage: f["-s"], Wins: defaultQuickWins, Time: defaultQuickTime}
	var errs []error
	intFlag := func(name string, valid func(int) bool, want string) (int, bool) {
		raw, given := f[name]
		if !given {
			return 0, false
		}
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || !valid(n) {
			errs = append(errs, fmt.Errorf("%s %q: must be %s", name, raw, want))
			return 0, false
		}
		return n, true
	}
	if n, given := intFlag("-rounds", func(n int) bool { return n >= 1 }, "a number of rounds to win, 1 or more"); given {
		q.Wins = n
	}
	if n, given := intFlag("-time", func(n int) bool { return n == -1 || n >= 1 }, "a number of seconds, or -1 for no limit"); given {
		q.Time = n
	}
	for i := range 2 {
		p := fmt.Sprintf("-p%d", i+1)
		if level, given := intFlag(p+".ai", func(n int) bool { return n >= 0 }, "an AI level, 0 for a human player"); given {
			q.CPU[i] = level > 0
			if level > 0 {
				notices = append(notices, fmt.Sprintf("%s.ai %d: the CPU opponent has no levels yet, the level is not used", p, level))
			}
		}
		if life, given := intFlag(p+".life", func(n int) bool { return n >= 1 && n <= DefaultHealth }, fmt.Sprintf("between 1 and %d", DefaultHealth)); given {
			q.Start[i].Life = life
		}
		if power, given := intFlag(p+".power", func(n int) bool { return n >= 0 && n <= MaxPower }, fmt.Sprintf("between 0 and %d", MaxPower)); given {
			q.Start[i].Power = power
		}
	}
	if len(errs) > 0 {
		return QuickVersus{}, false, notices, errors.Join(errs...)
	}
	return q, true, notices, nil
}

// ResolveCharDef finds the character decl names in fsys, as Ikemen GO's
// resolveCharDef (system.go:5370-5433) does: a name without ".def" is a
// folder holding <name>.def or a .def without its extension (the file first
// when decl has a folder in it), searched as given, then under chars/ and
// common/ (Ikemen GO's data/, roadmap decision 031). An absolute path names
// nothing in fsys: the host maps the ones inside the game folder first.
func ResolveCharDef(fsys fs.FS, decl string) (string, error) {
	return resolveDef(fsys, decl, "chars", "character")
}

// ResolveStageDef finds the stage decl names in fsys, as Ikemen GO's
// resolveStageDef (system.go:5436-5497): the .def, or a folder holding
// <name>.def, searched as given, then under stages/ and common/.
func ResolveStageDef(fsys fs.FS, decl string) (string, error) {
	return resolveDef(fsys, decl, "stages", "stage")
}

func resolveDef(fsys fs.FS, decl, folder, what string) (string, error) {
	p := strings.TrimSuffix(strings.ReplaceAll(strings.TrimSpace(decl), `\`, "/"), "/")
	if p == "" {
		return "", fmt.Errorf("no %s given", what)
	}
	candidates := []string{p}
	if !strings.HasSuffix(strings.ToLower(p), ".def") {
		inFolder := p + "/" + path.Base(p) + ".def"
		file := p + ".def"
		candidates = []string{file, inFolder}
		if folder == "chars" && !strings.Contains(p, "/") {
			candidates = []string{inFolder, file}
		}
	}
	for _, c := range candidates {
		if strings.HasPrefix(c, "/") || hasDriveLetter(c) {
			continue
		}
		for _, base := range []string{"", "common"} {
			for _, rel := range []string{c, folder + "/" + c} {
				if base != "" {
					rel = base + "/" + rel
				}
				if found, err := ResolvePath(fsys, ".", rel); err == nil && isFile(fsys, found) {
					return found, nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s %q not found in %s", what, decl, folder)
}

func isFile(fsys fs.FS, p string) bool {
	st, err := fs.Stat(fsys, p)
	return err == nil && st.Mode().IsRegular()
}
