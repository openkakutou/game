package game

import (
	"bytes"
	"errors"
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
	name   string
	read   bool
	data   []byte
	sounds map[soundKey]character.Sound
	// err is why the file cannot be used, nil until a lookup finds out.
	err error
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
// unreadable file is silence: false, Err telling why.
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

// Err is why the file cannot be used (missing, not a sound file), nil while
// it can or before the first lookup.
func (c *CommonSounds) Err() error { return c.err }

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
		c.err = fmt.Errorf("cannot read %s: %w", name, err)
		return
	}
	if err := checkSoundFile(data); err != nil {
		c.err = fmt.Errorf("%s is not a sound file: %w", name, err)
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

// errNoSample is a cue whose sound file has no such sample.
var errNoSample = errors.New("no such sample")

// FindCueSound returns the sample a cue plays: from the lifebar's sound file
// for a lifebar cue, from common/common.snd for a common cue, from the cue's
// fighter otherwise. Not found, the error says why: the sound file cannot
// be used, or it lacks the sample.
func (f *Flow) FindCueSound(c Cue) (character.Sound, error) {
	if c.Lifebar {
		if f.Lifebar == nil || f.Lifebar.SoundFile == nil {
			return character.Sound{}, errors.New("the lifebar has no sound file")
		}
		return findIn(f.Lifebar.SoundFile, c)
	}
	if c.Common {
		return findIn(f.CommonSounds, c)
	}
	if s, ok := f.Fighters[c.Side].FindSound(c.Group, c.Sample); ok {
		return s, nil
	}
	return character.Sound{}, errNoSample
}

// findIn looks c up in file, its error when the file cannot be used.
func findIn(file *CommonSounds, c Cue) (character.Sound, error) {
	s, ok := file.Find(c.Group, c.Sample)
	if ok {
		return s, nil
	}
	if file.Err() != nil {
		return s, file.Err()
	}
	return s, errNoSample
}
