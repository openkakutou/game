package lifebar

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// DefaultName is the lifebar used when none is selected or the selected one
// is gone (roadmap decision 031).
const DefaultName = "default"

// Entry is one usable folder under lifebars/.
type Entry struct {
	Name string
	// Def is the name, in the content file system, of the folder's fight.def.
	Def string
}

// Discover lists the folders of dir (a name in fsys) holding a fight.def that parses, sorted
// by name. A folder that does not is left out and reported in problems; a
// missing dir is no lifebar at all, not an error.
func Discover(fsys fs.FS, dir string) (entries []Entry, problems []error) {
	items, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, nil
	}
	for _, it := range items {
		if !it.IsDir() {
			continue
		}
		folder := path.Join(dir, it.Name())
		def, ferr := findDef(fsys, folder)
		if ferr != nil {
			problems = append(problems, fmt.Errorf("lifebar %q skipped: %w", it.Name(), ferr))
			continue
		}
		src, rerr := fs.ReadFile(fsys, def)
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
func findDef(fsys fs.FS, folder string) (string, error) {
	items, err := fs.ReadDir(fsys, folder)
	if err != nil {
		return "", err
	}
	for _, it := range items {
		if it.Type().IsRegular() && strings.EqualFold(it.Name(), "fight.def") {
			return path.Join(folder, it.Name()), nil
		}
	}
	return "", errors.New("no fight.def")
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
