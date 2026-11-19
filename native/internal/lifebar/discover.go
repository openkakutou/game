package lifebar

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultName is the lifebar used when none is selected or the selected one
// is gone (roadmap decision 031).
const DefaultName = "default"

// Entry is one usable folder under lifebars/.
type Entry struct {
	Name string
	// Def is the path of the folder's fight.def.
	Def string
}

// Discover lists the folders of dir holding a fight.def that parses, sorted
// by name. A folder that does not is left out and reported in problems; a
// missing dir is no lifebar at all, not an error.
func Discover(dir string) (entries []Entry, problems []error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	for _, it := range items {
		if !it.IsDir() {
			continue
		}
		folder := filepath.Join(dir, it.Name())
		def, ferr := findDef(folder)
		if ferr != nil {
			problems = append(problems, fmt.Errorf("lifebar %q skipped: %w", it.Name(), ferr))
			continue
		}
		src, rerr := os.ReadFile(def)
		if rerr != nil {
			problems = append(problems, fmt.Errorf("lifebar %q skipped: %w", it.Name(), rerr))
			continue
		}
		if _, perr := Parse(string(src)); perr != nil {
			problems = append(problems, fmt.Errorf("lifebar %q skipped: %s: %w", it.Name(), def, perr))
			continue
		}
		entries = append(entries, Entry{Name: it.Name(), Def: def})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, problems
}

// findDef is the folder's fight.def, whatever its case.
func findDef(folder string) (string, error) {
	items, err := os.ReadDir(folder)
	if err != nil {
		return "", err
	}
	for _, it := range items {
		if it.Type().IsRegular() && strings.EqualFold(it.Name(), "fight.def") {
			return filepath.Join(folder, it.Name()), nil
		}
	}
	return "", errors.New("no fight.def")
}

// appearance is config/appearance.json (roadmap decision 032); only the
// lifebar selection is read here.
type appearance struct {
	Lifebar string `json:"lifebar"`
}

// Selected reads the lifebar name chosen in the appearance file at path. A
// missing file or no choice is DefaultName; a corrupt file is DefaultName
// plus the error to report, as decision 032 falls back per concern.
func Selected(path string) (string, error) {
	src, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultName, nil
	}
	if err != nil {
		return DefaultName, err
	}
	var a appearance
	if jerr := json.Unmarshal(src, &a); jerr != nil {
		return DefaultName, fmt.Errorf("%s: %w", path, jerr)
	}
	if a.Lifebar == "" {
		return DefaultName, nil
	}
	return a.Lifebar, nil
}

// Pick is the entry named name, else DefaultName's, matching names without
// case as the folders come from any OS (decision 033). It reports false when
// neither exists.
func Pick(entries []Entry, name string) (Entry, bool) {
	for _, want := range []string{name, DefaultName} {
		for _, e := range entries {
			if strings.EqualFold(e.Name, want) {
				return e, true
			}
		}
	}
	return Entry{}, false
}
