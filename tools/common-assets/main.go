// Command common-assets generates OpenKakutou's own common effects and sounds
// (fightfx.sff, fightfx.air, common.snd) into a folder. See README.md.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openkakutou/sff"
	"github.com/openkakutou/snd"
)

// Files are the bytes of the three generated files.
type Files struct {
	Sff, Air, Snd []byte
}

// Build makes the files holding every number of the census.
func Build(census *Census) (*Files, error) {
	var sprites []sff.V1WriteSprite
	for group, e := range effects {
		for frame := 0; frame < e.frames; frame++ {
			sprite, err := spriteFor(group, frame)
			if err != nil {
				return nil, err
			}
			sprites = append(sprites, sprite)
		}
	}
	var sffData bytes.Buffer
	err := sff.SerializeV1(&sffData, [4]byte{0, 1, 0, 1}, false, sprites)
	if err != nil {
		return nil, err
	}

	set := &snd.V1SoundSet{Version: [4]byte{1, 0, 1, 0}}
	for _, key := range census.AllSounds() {
		set.Sounds = append(set.Sounds, snd.V1Sound{Group: key[0], Sample: key[1], Audio: wav(soundFor(key))})
	}
	sndData, err := snd.EncodeV1(set)
	if err != nil {
		return nil, err
	}

	return &Files{Sff: sffData.Bytes(), Air: []byte(airText(census.Actions())), Snd: sndData}, nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: common-assets CENSUS.md OUTPUT_DIR")
		os.Exit(2)
	}
	census, err := ReadCensus(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	files, err := Build(census)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outputs := map[string][]byte{"fightfx.sff": files.Sff, "fightfx.air": files.Air, "common.snd": files.Snd}
	for name, data := range outputs {
		err := os.WriteFile(filepath.Join(os.Args[2], name), data, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
