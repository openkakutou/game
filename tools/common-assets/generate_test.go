package main

import (
	"bytes"
	"testing"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/sff"
	"github.com/openkakutou/snd"
)

func buildFromCensus(t *testing.T) (*Census, *Files) {
	t.Helper()
	census, err := ReadCensus("CENSUS.md")
	if err != nil {
		t.Fatal(err)
	}
	files, err := Build(census)
	if err != nil {
		t.Fatal(err)
	}
	return census, files
}

func TestCensusIsRead(t *testing.T) {
	census, _ := buildFromCensus(t)
	if len(census.Sparks) == 0 || len(census.Anims) == 0 || len(census.Sounds) == 0 {
		t.Fatalf("empty census: %d sparks, %d anims, %d sounds", len(census.Sparks), len(census.Anims), len(census.Sounds))
	}
}

func TestEveryCensusActionPlaysASprite(t *testing.T) {
	census, files := buildFromCensus(t)
	groups, err := sff.Load(bytes.NewReader(files.Sff))
	if err != nil {
		t.Fatal(err)
	}
	sprites := map[[2]int]sff.Sprite{}
	for _, group := range groups {
		for _, sprite := range group.Sprites {
			sprites[[2]int{sprite.Group, sprite.Image}] = sprite
		}
	}
	animations, err := air.Parse(bytes.NewReader(files.Air))
	if err != nil {
		t.Fatal(err)
	}
	actions := map[int]air.Animation{}
	for _, animation := range animations {
		actions[animation.Number] = animation
	}

	var wanted []int
	wanted = append(wanted, census.Sparks...)
	wanted = append(wanted, census.Anims...)
	wanted = append(wanted, requiredActions...)
	for _, number := range wanted {
		animation, ok := actions[number]
		if !ok {
			t.Errorf("action %d is missing from fightfx.air", number)
			continue
		}
		if len(animation.Frames) == 0 {
			t.Errorf("action %d has no frame", number)
		}
		for _, frame := range animation.Frames {
			sprite, ok := sprites[[2]int{frame.Group, frame.Image}]
			if !ok {
				t.Errorf("action %d shows sprite %d,%d, absent from fightfx.sff", number, frame.Group, frame.Image)
				continue
			}
			if sprite.AxisX != sprite.Width/2 || sprite.AxisY != sprite.Height/2 {
				t.Errorf("sprite %d,%d axis (%d,%d) is not the centre of %dx%d", frame.Group, frame.Image, sprite.AxisX, sprite.AxisY, sprite.Width, sprite.Height)
			}
		}
	}
}

func TestEveryCensusSoundDecodes(t *testing.T) {
	census, files := buildFromCensus(t)
	wanted := append([][2]int{}, census.Sounds...)
	wanted = append(wanted, requiredSounds...)
	for _, key := range wanted {
		if key[0] < 0 || key[1] < 0 {
			continue
		}
		sound, err := snd.DecodeSound(bytes.NewReader(files.Snd), key[0], key[1], nil)
		if err != nil {
			t.Errorf("sound %d,%d: %v", key[0], key[1], err)
			continue
		}
		if sound.Channels != 1 || sound.SampleRate != sampleRate || len(sound.PCM) == 0 {
			t.Errorf("sound %d,%d: %d channel(s) at %d Hz with %d samples, want mono at %d Hz and not empty", key[0], key[1], sound.Channels, sound.SampleRate, len(sound.PCM), sampleRate)
		}
	}
}

func TestGenerationIsDeterministic(t *testing.T) {
	_, first := buildFromCensus(t)
	_, second := buildFromCensus(t)
	if !bytes.Equal(first.Sff, second.Sff) || !bytes.Equal(first.Air, second.Air) || !bytes.Equal(first.Snd, second.Snd) {
		t.Error("two runs produced different bytes")
	}
}
