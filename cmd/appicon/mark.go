package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
)

type geometry struct {
	corner     float64
	stemLeft   float64
	stemRight  float64
	stemTop    float64
	stemBottom float64
	bowlRadius float64
	bowlInner  float64
}

var mark = geometry{
	corner:     0.20,
	stemLeft:   0.34,
	stemRight:  0.44,
	stemTop:    0.22,
	stemBottom: 0.78,
	bowlRadius: 0.22,
	bowlInner:  0.12,
}

var (
	plate = color.NRGBA{R: 0x0e, G: 0x10, B: 0x16, A: 0xff}
	ink   = color.NRGBA{R: 0xe8, G: 0x81, B: 0x3a, A: 0xff}
)

const samples = 4

func (g geometry) bowlCentre() (x, y float64) {
	return g.stemRight, g.stemTop + g.bowlRadius
}

func (g geometry) onPlate(x, y float64) bool {
	half := 0.5 - g.corner
	dx := math.Abs(x-0.5) - half
	dy := math.Abs(y-0.5) - half
	return math.Hypot(math.Max(dx, 0), math.Max(dy, 0))-g.corner <= 0
}

func (g geometry) onMark(x, y float64) bool {
	if x >= g.stemLeft && x <= g.stemRight && y >= g.stemTop && y <= g.stemBottom {
		return true
	}

	centreX, centreY := g.bowlCentre()
	if x < centreX {
		return false
	}

	distance := math.Hypot(x-centreX, y-centreY)
	return distance <= g.bowlRadius && distance >= g.bowlInner
}

func blend(from, to uint8, share float64) uint8 {
	return uint8(math.Round(float64(from)*(1-share) + float64(to)*share))
}

func render(size int) *image.NRGBA {
	canvas := image.NewNRGBA(image.Rect(0, 0, size, size))
	side := float64(size)
	grid := float64(samples)
	total := grid * grid

	for row := range size {
		for column := range size {
			onPlate := 0
			onMark := 0

			for down := range samples {
				for across := range samples {
					x := (float64(column) + (float64(across)+0.5)/grid) / side
					y := (float64(row) + (float64(down)+0.5)/grid) / side
					if !mark.onPlate(x, y) {
						continue
					}
					onPlate++
					if mark.onMark(x, y) {
						onMark++
					}
				}
			}

			if onPlate == 0 {
				continue
			}

			share := float64(onMark) / float64(onPlate)
			canvas.SetNRGBA(column, row, color.NRGBA{
				R: blend(plate.R, ink.R, share),
				G: blend(plate.G, ink.G, share),
				B: blend(plate.B, ink.B, share),
				A: uint8(math.Round(float64(onPlate) / total * 255)),
			})
		}
	}
	return canvas
}

func hex(of color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", of.R, of.G, of.B)
}

func vector() []byte {
	const side = 100.0

	g := mark
	_, centreY := g.bowlCentre()
	radius := g.bowlRadius * side
	inner := g.bowlInner * side

	var out bytes.Buffer
	out.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100">`)
	fmt.Fprintf(&out, `<rect width="100" height="100" rx="%g" ry="%g" fill="%s"/>`, g.corner*side, g.corner*side, hex(plate))
	fmt.Fprintf(
		&out,
		`<path fill="%s" fill-rule="evenodd" d="M%g %gL%g %gL%g %gA%g %g 0 0 1 %g %gL%g %gZM%g %gA%g %g 0 0 1 %g %gZ"/>`,
		hex(ink),
		g.stemLeft*side, g.stemBottom*side,
		g.stemLeft*side, g.stemTop*side,
		g.stemRight*side, g.stemTop*side,
		radius, radius,
		g.stemRight*side, (centreY*side)+radius,
		g.stemRight*side, g.stemBottom*side,
		g.stemRight*side, (centreY*side)-inner,
		inner, inner,
		g.stemRight*side, (centreY*side)+inner,
	)
	out.WriteString("</svg>\n")
	return out.Bytes()
}
