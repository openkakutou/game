# Common effects and sounds

Generates OpenKakutou's own `common/fightfx.sff`, `common/fightfx.air` and `common/common.snd` (CC0, see `common/LICENSES.md`). Nothing third-party is copied: the sprites are drawn by formulas, the sounds synthesised by formulas.

## Run

```sh
cd tools/common-assets
go run . CENSUS.md ../../common    # rewrites the three files
go test ./...                      # loads them with sff, snd and character/air, checks every census number
```

The output is byte-identical between two runs on the same platform and Go toolchain.

## Where the numbers come from

`census.py` counts what real characters reference; `CENSUS.md` is its output (numbers and counts only):

```sh
python3 -I tools/common-assets/census.py /path/to/Mugen/chars engine/testdata/reference/chars native/testdata/reference/chars > tools/common-assets/CENSUS.md
```

The generator reads `CENSUS.md`, so every number a character uses gets an action in `fightfx.air` and a sound in `common.snd`. It also adds the numbers the sources name:

- Ikemen GO @ 54d4148a728153c2097a0fb899a06812b2d3e193: default `sparkno` 2 and `guard.sparkno` 40 (`src/char.go:341-342`); a HitDef starts with sparks from fightfx and no hit/guard sound (`src/char.go:758-772`); common sounds come from `common.snd` and fightfx from `fightfx.sff`/`fightfx.air`, found in the screenpack's folder, then `data/` (`src/fightscreen.go:4956`, `:4967`, `:4978`); `F` selects the fightfx prefix, `S` the character's own files (`src/char.go:6636-6661`).
- MUGEN docs, HitDef (`sparkno`, `guard.sparkno`, `hitsound`, `guardsound`): no prefix means fightfx and `common.snd`, `S` means the character's own; hit sounds `5,0` (light) to `5,4` (painful whack), guard sound `6,0`.
- Common states `common1.cns` (MUGEN 1.1) and `common1.cns.zss` (Ikemen GO) in `native/testdata/reference/common/`: ground bounce plays `Explod anim = F(60 + 0..2)` (`common1.cns:1523`, `common1.cns.zss:909`) and `PlaySnd F7, 0..2` (`common1.cns:1534`, `common1.cns.zss:736`).

Round-call and KO announcer sounds are not in `common.snd`: Ikemen GO plays them from the screenpack's own sound file (`src/fightscreen.go:3673-3817`).

## What each number plays

- Actions 0 to 3 are hit sparks from light to big, 40 to 42 guard sparks from light to heavy, 60 to 62 ground-bounce dust.
- Sounds 5,0 to 5,4 are hit thumps from light to heavy, 6,0 a guard clank, 7,0 to 7,2 light to heavy bounce thuds.
- Every other number in the census (characters' own fightfx packs, unusual values) plays an effect or a sound chosen from its value, so it resolves without claiming a meaning the sources do not give.
- Negative numbers mean "none" and are skipped.

Sprites are SFF v1 (MUGEN 1.0, MUGEN 1.1 and Ikemen GO all read v1; v2 would exclude MUGEN 1.0), 8-bit with one 256-colour palette per effect, colour 0 transparent, axis at the centre, drawn with additive blending. Sounds are 22050 Hz mono 16-bit PCM in a SND v1.
