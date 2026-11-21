package game

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"

	"github.com/openkakutou/character"
	sndfile "github.com/openkakutou/snd"
)

// CommonSounds is the common sound file (common/common.snd), where a PlaySnd
// with the F prefix reads its sample: Ikemen GO's fight.def common.snd
// (src/char.go:6653-6663), moved to common/ by roadmap decision 031.
// It reads the file on the first lookup and decodes each sample once.
type CommonSounds struct {
	fsys fs.FS
	// name is the file read; "" is common/common.snd.
	name     string
	read     bool
	data     []byte
	sounds   map[soundKey]character.Sound
	warnings []string
}

// NewCommonSounds returns the common sounds of fsys's common/ folder.
func NewCommonSounds(fsys fs.FS) *CommonSounds {
	return &CommonSounds{fsys: fsys, sounds: map[soundKey]character.Sound{}}
}

// NewSoundFile returns the sounds of the .snd file name of fsys, read the
// same way (the lifebar's fight.snd).
func NewSoundFile(fsys fs.FS, name string) *CommonSounds {
	return &CommonSounds{fsys: fsys, name: name, sounds: map[soundKey]character.Sound{}}
}

// Find returns the decoded common sample (group, sample). A missing or
// unreadable file is silence: false, and one warning for the whole file.
func (c *CommonSounds) Find(group, sample int) (character.Sound, bool) {
	key := soundKey{group, sample}
	if s, seen := c.sounds[key]; seen {
		return s, len(s.PCM) > 0
	}
	c.load()
	var s character.Sound
	if c.data != nil {
		d, err := sndfile.DecodeSound(bytes.NewReader(c.data), group, sample, nil)
		if err == nil && len(d.PCM) > 0 {
			s = character.Sound{Group: group, Sample: sample, SampleRate: d.SampleRate, Channels: d.Channels, BitsPerSample: d.BitsPerSample, PCM: d.PCM}
		}
	}
	c.sounds[key] = s
	return s, len(s.PCM) > 0
}

// Warnings lists what went wrong reading the file, at most one entry.
func (c *CommonSounds) Warnings() []string { return c.warnings }

func (c *CommonSounds) load() {
	if c.read {
		return
	}
	c.read = true
	name := c.name
	if name == "" {
		name = path.Join(CommonFolder(c.fsys), "common.snd")
	}
	data, err := fs.ReadFile(c.fsys, name)
	if err != nil {
		c.warnings = append(c.warnings, fmt.Sprintf("common sounds: cannot read %s: %v", name, err))
		return
	}
	if err := checkSoundFile(data); err != nil {
		c.warnings = append(c.warnings, fmt.Sprintf("common sounds: %s is not a sound file: %v", name, err))
		return
	}
	c.data = data
}

// checkSoundFile reports whether data's header and table parse as .snd v1 or v2.
func checkSoundFile(data []byte) error {
	r := bytes.NewReader(data)
	if _, err := sndfile.ParseV1(r); err == nil {
		return nil
	}
	_, err := sndfile.ParseV2(r)
	return err
}

// FindCueSound returns the sample a cue plays: from common/common.snd for a
// common cue, from the cue's fighter otherwise.
func (f *Flow) FindCueSound(c Cue) (character.Sound, bool) {
	if c.Lifebar {
		if f.Lifebar == nil || f.Lifebar.SoundFile == nil {
			return character.Sound{}, false
		}
		return f.Lifebar.SoundFile.Find(c.Group, c.Sample)
	}
	if c.Common {
		return f.CommonSounds.Find(c.Group, c.Sample)
	}
	return f.Fighters[c.Side].FindSound(c.Group, c.Sample)
}
