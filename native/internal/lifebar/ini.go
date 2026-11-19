package lifebar

import (
	"strconv"
	"strings"
)

// section is one "[Name]" block of a fight.def: lower-cased keys, the first
// occurrence of a key wins (Ikemen GO common.go:1071-1091).
type section map[string]string

// sections splits a fight.def into its sections by lower-cased name. A name
// that appears twice keeps its first block, as Ikemen GO's loader only acts
// on the first [Files] (fightscreen.go:4917).
func sections(src string) map[string]section {
	out := map[string]section{}
	var cur section
	for _, raw := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			name := sectionName(line)
			if name == "" {
				cur = nil
				continue
			}
			if _, seen := out[name]; seen {
				cur = nil
				continue
			}
			cur = section{}
			out[name] = cur
			continue
		}
		if cur == nil {
			continue
		}
		cur.add(line)
	}
	return out
}

// sectionName is Ikemen GO's SectionName (common.go:862) lower-cased, with
// the subname kept: "[Simul Lifebar]" is "simul lifebar".
func sectionName(line string) string {
	line = strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
	if !strings.HasSuffix(line, "]") {
		return ""
	}
	end := strings.Index(line, "]")
	return strings.ToLower(strings.TrimSpace(line[1:end]))
}

func (s section) add(line string) {
	line = strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
	cut := strings.IndexAny(line, "= \t")
	if cut <= 0 {
		return
	}
	key := strings.ToLower(line[:cut])
	value := ""
	if eq := strings.Index(line, "="); eq >= 0 {
		value = strings.TrimSpace(line[eq+1:])
	}
	if _, seen := s[key]; seen {
		return
	}
	s[key] = value
}

// ints reads up to len(out) comma-separated integers the way Ikemen GO's
// ReadI32 does: an empty item keeps its default, digits stop at the first
// non-digit. It reports whether the key is present and not empty.
func (s section) ints(key string, out ...*int) bool {
	value := s[key]
	if value == "" {
		return false
	}
	for i, item := range strings.Split(value, ",") {
		if i >= len(out) {
			break
		}
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		*out[i] = atoi(item)
	}
	return true
}

// floats is ints for decimal values (Ikemen GO ReadF32).
func (s section) floats(key string, out ...*float64) bool {
	value := s[key]
	if value == "" {
		return false
	}
	for i, item := range strings.Split(value, ",") {
		if i >= len(out) {
			break
		}
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		*out[i] = atof(item)
	}
	return true
}

// atoi is Ikemen GO's lenient Atoi (common.go:191): an optional sign, then
// digits up to the first other character.
func atoi(s string) int {
	sign := 1
	if s != "" && (s[0] == '-' || s[0] == '+') {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return sign * n
}

// atof reads the longest leading decimal number of s, 0 when none.
func atof(s string) float64 {
	end := 0
	for end < len(s) && strings.ContainsRune("+-.0123456789", rune(s[end])) {
		end++
	}
	for end > 0 {
		v, err := strconv.ParseFloat(s[:end], 64)
		if err == nil {
			return v
		}
		end--
	}
	return 0
}
