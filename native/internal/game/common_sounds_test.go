package game

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/character"
	"github.com/openkakutou/engine/v2/evaluator"
	sndfile "github.com/openkakutou/snd"
)

// shippedCommonSnd is the game's own common/common.snd (CC0, item 050).
const shippedCommonSnd = "../../../common/common.snd"

func commonFS(t *testing.T, sndFile []byte) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{"chars/.keep": {}, "stages/.keep": {}}
	if sndFile != nil {
		fsys["common/common.snd"] = &fstest.MapFile{Data: sndFile}
	}
	return fsys
}

func TestCommonSounds_FindsHitSoundInTheShippedCommonFile(t *testing.T) {
	raw, err := os.ReadFile(shippedCommonSnd)
	if err != nil {
		t.Fatal(err)
	}
	c := NewCommonSounds(commonFS(t, raw))
	s, ok := c.Find(5, 0)
	if !ok {
		t.Fatal("common sound 5,0 (light hit) not found in common/common.snd")
	}
	if s.Group != 5 || s.Sample != 0 || len(s.PCM) == 0 || s.SampleRate < 1 || s.Channels < 1 {
		t.Fatalf("sound 5,0 = group %d sample %d, %d PCM samples, %d Hz, %d channels", s.Group, s.Sample, len(s.PCM), s.SampleRate, s.Channels)
	}
}

func TestCommonSounds_ReplacingTheFileChangesTheSound(t *testing.T) {
	pcm := []int16{100, -100, 200, -200}
	set := &sndfile.V1SoundSet{Version: [4]byte{0, 1, 0, 0}, HeaderPadding: make([]byte, 488), Sounds: []sndfile.V1Sound{{Group: 5, Sample: 0, Audio: WAVBytes(character.Sound{PCM: pcm, Channels: 1, SampleRate: 22050})}}}
	custom, err := sndfile.EncodeV1(set)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := NewCommonSounds(commonFS(t, custom)).Find(5, 0)
	if !ok {
		t.Fatal("sound 5,0 of the replaced common.snd not found")
	}
	if len(s.PCM) != len(pcm) || s.PCM[2] != 200 {
		t.Fatalf("PCM = %v, want the replaced file's %v", s.PCM, pcm)
	}
}

func TestCommonSounds_UnknownKeyIsNotFound(t *testing.T) {
	raw, err := os.ReadFile(shippedCommonSnd)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := NewCommonSounds(commonFS(t, raw)).Find(999, 9); ok {
		t.Fatal("999,9 is not in common.snd: want not found")
	}
}

func TestCommonSounds_MissingFileIsSilenceWithOneWarning(t *testing.T) {
	c := NewCommonSounds(commonFS(t, nil))
	if _, ok := c.Find(5, 0); ok {
		t.Fatal("no common.snd: want not found")
	}
	if _, ok := c.Find(6, 0); ok {
		t.Fatal("no common.snd: want not found")
	}
	warns := c.Warnings()
	if len(warns) != 1 {
		t.Fatalf("warnings = %q, want exactly one for the missing file", warns)
	}
}

func TestCommonSounds_CorruptFileIsSilenceWithOneWarning(t *testing.T) {
	c := NewCommonSounds(commonFS(t, []byte("not a sound file")))
	if _, ok := c.Find(5, 0); ok {
		t.Fatal("corrupt common.snd: want not found")
	}
	c.Find(5, 1)
	if len(c.Warnings()) != 1 {
		t.Fatalf("warnings = %q, want one", c.Warnings())
	}
}

func TestQueueCues_CarriesTheCommonFlag(t *testing.T) {
	m := &Match{}
	m.queueCues([2][]evaluator.SoundEvent{{{Group: 5, Sample: 0, Common: true}}, {{Group: 5, Sample: 0}}})
	cues := m.DrainCues(nil)
	if len(cues) != 2 || !cues[0].Common || cues[1].Common {
		t.Fatalf("cues = %+v, want side 1 common, side 2 the character's own", cues)
	}
}

func TestFindCueSound_CommonCueReadsCommonFileNotTheFighter(t *testing.T) {
	raw, err := os.ReadFile(shippedCommonSnd)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFlow(commonFS(t, raw), nil, nil, DefaultLoader(commonFS(t, raw), "common"), nil)
	f.Fighters[0] = &Fighter{}
	if _, ok := f.FindCueSound(Cue{Group: 5, Sample: 0, Common: true}); !ok {
		t.Fatal("common cue 5,0 not found in common/common.snd")
	}
	if _, ok := f.FindCueSound(Cue{Group: 5, Sample: 0}); ok {
		t.Fatal("a fighter without sounds must not play 5,0 from the common file")
	}
}
