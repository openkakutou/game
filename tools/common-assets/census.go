package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Census holds the numbers CENSUS.md records: the fightfx actions real
// characters use (as spark numbers and as F-prefixed anims) and the common
// sounds they play. Negative numbers mean "none" and are not kept.
type Census struct {
	Sparks []int
	Anims  []int
	Sounds [][2]int
}

const (
	sparkHeading = "## Fightfx actions used as spark numbers"
	soundHeading = "## Common sounds"
	animHeading  = "## Fightfx actions used as F-prefixed anims"
)

// ReadCensus parses the tables of CENSUS.md, the output of census.py.
func ReadCensus(path string) (*Census, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	census := &Census{}
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## ") {
			section = line
			continue
		}
		key, ok := tableKey(line)
		if !ok {
			continue
		}
		err := census.add(section, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return census, nil
}

// tableKey returns the first cell of a numeric table row.
func tableKey(line string) (string, bool) {
	if !strings.HasPrefix(line, "| ") {
		return "", false
	}
	cells := strings.Split(line, "|")
	key := strings.TrimSpace(cells[1])
	first := strings.SplitN(key, ",", 2)[0]
	_, err := strconv.Atoi(first)
	return key, err == nil
}

func (c *Census) add(section, key string) error {
	switch {
	case strings.HasPrefix(section, sparkHeading):
		n, err := strconv.Atoi(key)
		if err != nil {
			return err
		}
		if n >= 0 {
			c.Sparks = append(c.Sparks, n)
		}
	case strings.HasPrefix(section, animHeading):
		n, err := strconv.Atoi(key)
		if err != nil {
			return err
		}
		if n >= 0 {
			c.Anims = append(c.Anims, n)
		}
	case strings.HasPrefix(section, soundHeading):
		var group, sample int
		_, err := fmt.Sscanf(key, "%d,%d", &group, &sample)
		if err != nil {
			return err
		}
		if group >= 0 && sample >= 0 {
			c.Sounds = append(c.Sounds, [2]int{group, sample})
		}
	}
	return nil
}

// Actions returns every fightfx action number the files must hold: the census
// numbers plus the numbers the sources name (see README.md).
func (c *Census) Actions() []int {
	seen := map[int]bool{}
	var out []int
	for _, list := range [][]int{requiredActions, c.Sparks, c.Anims} {
		for _, n := range list {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	slices.Sort(out)
	return out
}

// AllSounds returns every common sound the file must hold, sorted.
func (c *Census) AllSounds() [][2]int {
	seen := map[[2]int]bool{}
	var out [][2]int
	for _, list := range [][][2]int{requiredSounds, c.Sounds} {
		for _, key := range list {
			if !seen[key] {
				seen[key] = true
				out = append(out, key)
			}
		}
	}
	slices.SortFunc(out, func(a, b [2]int) int {
		return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
	})
	return out
}
