package game

import (
	"encoding/binary"

	"github.com/openkakutou/character"
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/match"
	sharedlifebar "github.com/openkakutou/lifebar"
)

// Cue is one sound effect the engine asked a fighter to play (PlaySnd).
type Cue struct {
	Side          match.Side
	Group, Sample int
	// Common is true for an F-prefixed PlaySnd: the sample is read from
	// common/common.snd, not from the character's own sounds.
	Common bool
	// Channel is PlaySnd's channel, -1 when absent; LowPriority keeps a busy
	// channel playing instead of cutting it (see RequestSlot).
	Channel     int
	LowPriority bool
	// Lifebar is a lifebar call's sound, read from its fight.snd.
	Lifebar bool
}

// QueueLifebarSound queues a lifebar call's sound (group, sample).
func (m *Match) QueueLifebarSound(ref sharedlifebar.SoundRef) {
	m.cues = append(m.cues, Cue{Group: ref.Group, Sample: ref.Number, Channel: -1, Lifebar: true})
}

// queueCues appends one tick's PlaySnd events, side 0 first.
func (m *Match) queueCues(sounds [2][]evaluator.SoundEvent) {
	for side, evs := range sounds {
		for _, e := range evs {
			m.cues = append(m.cues, Cue{Side: match.Side(side), Group: e.Group, Sample: e.Sample, Common: e.Common, Channel: e.Channel, LowPriority: e.LowPriority})
		}
	}
}

// DrainCues returns dst truncated to length 0 followed by every pending cue,
// in the order they were produced, and empties the queue. It reuses dst's and
// the queue's capacity, so it does not allocate when nothing is pending.
func (m *Match) DrainCues(dst []Cue) []Cue {
	dst = append(dst[:0], m.cues...)
	m.cues = m.cues[:0]
	return dst
}

type soundKey struct{ group, sample int }

// FindSound returns the decoded sample for a PlaySnd (group, sample). It
// reports false for an unknown key, an empty PCM, or a sound that failed to decode.
func (f *Fighter) FindSound(group, sample int) (character.Sound, bool) {
	if f.sounds == nil {
		f.sounds = map[soundKey]character.Sound{}
		if f.Char != nil {
			for _, g := range f.Char.Sounds {
				for _, s := range g.Sounds {
					f.sounds[soundKey{s.Group, s.Sample}] = s
				}
			}
		}
	}
	s, ok := f.sounds[soundKey{group, sample}]
	if !ok || len(s.PCM) == 0 || s.Error != "" {
		return character.Sound{}, false
	}
	return s, true
}

// WAVBytes wraps a sound's 16-bit PCM in a canonical 44-byte RIFF/WAVE header.
// It returns nil when the sound has no PCM, no channel or no sample rate.
func WAVBytes(s character.Sound) []byte {
	if len(s.PCM) == 0 || s.Channels < 1 || s.SampleRate < 1 {
		return nil
	}
	dataLen := uint32(len(s.PCM) * 2)
	out := make([]byte, 44, 44+int(dataLen))
	le := binary.LittleEndian
	copy(out[0:], "RIFF")
	le.PutUint32(out[4:], 36+dataLen)
	copy(out[8:], "WAVEfmt ")
	le.PutUint32(out[16:], 16)
	le.PutUint16(out[20:], 1)
	le.PutUint16(out[22:], uint16(s.Channels))
	le.PutUint32(out[24:], uint32(s.SampleRate))
	le.PutUint32(out[28:], uint32(s.SampleRate*s.Channels*2))
	le.PutUint16(out[32:], uint16(s.Channels*2))
	le.PutUint16(out[34:], 16)
	copy(out[36:], "data")
	le.PutUint32(out[40:], dataLen)
	for _, v := range s.PCM {
		out = le.AppendUint16(out, uint16(v))
	}
	return out
}

// Music reports who owns the music. The stage music belongs to the current
// match while the match or result screen is shown, or Options is open over
// the paused match: owner is the match and path the stage's music file
// (relative paths resolved beside the stage .def, then from the content root;
// "" when rejected or when there is none). On any other screen with a
// screenpack screen, menu names that screen, whose music (or silence) the
// pack decides; the error screen has none. Only one of owner and menu is set.
func (f *Flow) Music() (path string, owner *Match, menu string) {
	overPausedMatch := f.inOptions() && f.options.fromPause
	if (f.Screen != ScreenMatch && f.Screen != ScreenResult && !overPausedMatch) || f.Match == nil {
		if f.Screen == ScreenMatch || f.Screen == ScreenResult {
			return "", nil, ""
		}
		return "", nil, f.PackScreen()
	}
	if f.Stage == nil || f.Stage.MusicFile == "" {
		return "", f.Match, ""
	}
	// Memoized: Music runs every frame and path.Join allocates.
	if f.musicFrom != [2]string{f.StageDefPath, f.Stage.MusicFile} {
		f.musicFrom = [2]string{f.StageDefPath, f.Stage.MusicFile}
		f.musicPath = joinResolved(f.FS, stageBases(f.StageDefPath), f.Stage.MusicFile)
	}
	return f.musicPath, f.Match, ""
}

// MusicChange compares the current music owner with prev (pointer identity):
// changed means the player must stop and, if path is non-empty, start path.
func MusicChange(prev *Match, f *Flow) (next *Match, path string, changed bool) {
	path, next, _ = f.Music()
	return next, path, next != prev
}
