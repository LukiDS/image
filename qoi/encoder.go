package qoi

import (
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/LukiDS/image/imgconv"
)

type encoder struct {
	m      image.Image
	err    error
	buf    []byte
	width  int
	height int
}

// Encode writes the Image m to w in QOI format. Any Image may be
// encoded, but images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	width := m.Bounds().Dx()
	height := m.Bounds().Dy()
	if width <= 0 || height <= 0 || width*height > qoiMaxPixels {
		return fmt.Errorf("invalid image size")
	}

	maxSize := qoiHeaderSize + (width * height * int(qoiDefaultChannel+1)) + len(qoiEndMarker) //worst case -> [header-size + (op--r--g--b--{a} * pixels) + padding-size]
	e := encoder{
		m:      m,
		buf:    make([]byte, 0, maxSize),
		width:  width,
		height: height,
	}

	e.encodeHeader()
	e.encode()
	e.encodePadding()

	if e.err != nil {
		return e.err
	}

	_, err := w.Write(e.buf)
	if err != nil {
		return err
	}

	return nil
}

func (e *encoder) encodeHeader() {
	e.buf = append(e.buf, qoiMagic...)
	e.buf = append(e.buf, byte(e.width>>24), byte(e.width>>16), byte(e.width>>8), byte(e.width))
	e.buf = append(e.buf, byte(e.height>>24), byte(e.height>>16), byte(e.height>>8), byte(e.height))
	e.buf = append(e.buf, qoiDefaultChannel, qoiDefaultColorSpace)
}

func (e *encoder) encode() {
	if e.err != nil {
		return
	}

	img := imgconv.ToNRGBA(e.m)

	colorBuffer := [64]color.NRGBA{}
	pxPrev := color.NRGBA{0, 0, 0, 255}

	run := uint8(0)
	maxPixelPos := e.width * e.height
	for pxPos := 0; pxPos < maxPixelPos; pxPos++ {
		x := pxPos % e.width
		y := pxPos / e.width
		px := img.NRGBAAt(x, y)

		if px == pxPrev {
			run++
			if run == qoiMaxRunSize || pxPos == maxPixelPos-1 {
				e.buf = append(e.buf, opRUN|run-1)
				run = 0
			}
			continue
		}

		if run > 0 {
			e.buf = append(e.buf, opRUN|run-1)
			run = 0
		}

		idx := hash(px)
		if colorBuffer[idx] == px {
			e.buf = append(e.buf, opINDEX|idx)
			pxPrev = px
			continue
		}
		colorBuffer[idx] = px

		if px.A != pxPrev.A {
			e.buf = append(e.buf, opRGBA, px.R, px.G, px.B, px.A)
			pxPrev = px
			continue
		}

		vr := int8(px.R - pxPrev.R)
		vg := int8(px.G - pxPrev.G)
		vb := int8(px.B - pxPrev.B)

		if isValidDiff(vr, vg, vb) {
			chunk := opDIFF | (uint8(vr+2) << 4) | (uint8(vg+2) << 2) | uint8(vb+2)
			e.buf = append(e.buf, chunk)
			pxPrev = px
			continue
		}

		vgR := vr - vg
		vgB := vb - vg

		if isValidLuma(vgR, vg, vgB) {
			e.buf = append(e.buf, opLUMA|uint8(vg+32), (uint8(vgR+8)<<4)|uint8(vgB+8))
			pxPrev = px
			continue
		}

		e.buf = append(e.buf, opRGB, px.R, px.G, px.B)
		pxPrev = px
	}
}

func (e *encoder) encodePadding() {
	e.buf = append(e.buf, qoiEndMarker...)
}

func isValidDiff(vr, vg, vb int8) bool {
	return vr > -3 && vr < 2 &&
		vg > -3 && vg < 2 &&
		vb > -3 && vb < 2
}

func isValidLuma(vgR, vg, vgB int8) bool {
	return vgR > -9 && vgR < 8 &&
		vg > -33 && vg < 32 &&
		vgB > -9 && vgB < 8
}
