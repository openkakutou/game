package main

import (
	"math"

	"github.com/openkakutou/sff"
)

// A drawing function returns the brightness (0 to 1) of the effect at a point
// (x, y in -1..1, centred on the sprite axis) and a time t in 0..1.

func radius(x, y float64) float64 { return math.Hypot(x, y) }

func clamp(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// burst is a bright core with n rays; ringWeight adds a shock ring.
func burst(rays int, ringWeight float64) func(x, y, t float64) float64 {
	return func(x, y, t float64) float64 {
		r := radius(x, y)
		angle := math.Atan2(y, x)
		core := math.Exp(-math.Pow(r/(0.18+0.22*t), 2))
		spike := math.Pow(math.Abs(math.Cos(float64(rays)*angle/2)), 6)
		reach := 0.25 + 0.7*t
		rayLight := spike * math.Max(0, 1-r/reach) * 0.9
		ringLight := ringWeight * ring(x, y, t)
		return clamp((core*1.2 + rayLight + ringLight*0.6) * (1 - t*t))
	}
}

// ring is a thin expanding circle.
func ring(x, y, t float64) float64 {
	r := radius(x, y)
	radiusNow := 0.2 + 0.7*t
	width := 0.12 * (1 - 0.6*t)
	return clamp(math.Exp(-math.Pow((r-radiusNow)/width, 2)) * (1 - t))
}

// shield is a bright rimmed disc that swells then fades, the guard flash.
func shield(scale float64) func(x, y, t float64) float64 {
	return func(x, y, t float64) float64 {
		r := radius(x, y)
		edge := scale * (0.35 + 0.55*math.Sqrt(t))
		rim := math.Exp(-math.Pow((r-edge)/0.07, 2))
		fill := 0.35 * clamp(1-r/edge)
		return clamp((rim + fill) * (1 - t*t))
	}
}

// dust is a few puffs that spread sideways and rise.
func dust(scale float64) func(x, y, t float64) float64 {
	puffs := [][3]float64{{-0.5, 0.25, 0.3}, {0.5, 0.25, 0.3}, {-0.2, 0.05, 0.38}, {0.25, 0.1, 0.34}, {0, 0.3, 0.28}}
	return func(x, y, t float64) float64 {
		light := 0.0
		for _, p := range puffs {
			cx := p[0] * scale * (0.4 + 0.9*t)
			cy := 0.5 - p[1]*scale*(0.3+0.9*t)
			size := p[2] * scale * (0.5 + 0.7*t)
			d := math.Hypot(x-cx, y-cy)
			light += math.Exp(-math.Pow(d/size, 2))
		}
		return clamp(light * 0.7 * (1 - t))
	}
}

// flash is a large radial flash with many rays, for the heavy effects.
func flash(x, y, t float64) float64 {
	r := radius(x, y)
	angle := math.Atan2(y, x)
	glow := math.Exp(-math.Pow(r/(0.3+0.5*t), 2))
	rays := math.Pow(math.Abs(math.Cos(8*angle)), 8) * math.Max(0, 1-r/(0.4+0.6*t))
	return clamp((glow + rays) * (1 - t*t*t))
}

// spriteFor renders frame number frame of effect e as an 8-bit indexed sprite
// (index 0 is transparent) with the effect's own palette, axis at its centre.
func spriteFor(group int, frame int) (sff.V1WriteSprite, error) {
	e := effects[group]
	t := float64(frame) / float64(e.frames)
	pixels := make([]byte, e.size*e.size)
	for py := 0; py < e.size; py++ {
		for px := 0; px < e.size; px++ {
			x := (float64(px)+0.5)/float64(e.size)*2 - 1
			y := (float64(py)+0.5)/float64(e.size)*2 - 1
			v := e.draw(x, y, t)
			pixels[py*e.size+px] = byte(math.Round(v * 255))
		}
	}
	pcx, err := sff.EncodePCX(&sff.PCXImage{Width: e.size, Height: e.size, BytesPerPixel: 1, Pixels: pixels})
	if err != nil {
		return sff.V1WriteSprite{}, err
	}
	return sff.V1WriteSprite{
		Group: group, Image: frame, AxisX: e.size / 2, AxisY: e.size / 2,
		PixelData: pcx, Palette: palette(e.color),
	}, nil
}

// palette is a 256-colour ramp: black, then the effect colour at 60% brightness,
// then white. The pixel value is the brightness, so index 0 (transparent)
// is the only fully dark entry.
func palette(base [3]uint8) []byte {
	out := make([]byte, 0, 768)
	for i := 0; i < 256; i++ {
		v := float64(i) / 255
		for _, c := range base {
			if v < 0.6 {
				out = append(out, byte(math.Round(float64(c)*v/0.6)))
			} else {
				out = append(out, byte(math.Round(float64(c)+(255-float64(c))*(v-0.6)/0.4)))
			}
		}
	}
	return out
}
