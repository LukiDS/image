package qoi

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/LukiDS/image/imgconv"
)

type encoder struct {
	w   io.Writer
	m   image.Image
	err error
}

// Encode writes the Image m to w in QOI format. Any Image may be
// encoded, but images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	if m.ColorModel() != color.NRGBAModel {
		return fmt.Errorf("image not in image.NRGBA fromat")
	}

	e := encoder{
		w: w,
		m: m,
	}

	e.encodeHeader()
	e.encodeBody()
	e.encodeEndMarker()

	if e.err != nil {
		return e.err
	}

	return nil
}

func (e *encoder) encodeHeader() {
	mw, mh := e.m.Bounds().Dx(), e.m.Bounds().Dy()
	if mw <= 0 || mh <= 0 || mw*mh > qoiMaxPixels {
		e.err = fmt.Errorf("invalid image size")
		return
	}

	e.writeBytes([]byte(qoiMagic))
	e.writeBytes(uint32(mw))
	e.writeBytes(uint32(mh))
	e.writeBytes(qoiDefaultChannel)
	e.writeBytes(qoiDefaultColorSpace)
}

func (e *encoder) encodeEndMarker() {
	if e.err != nil {
		return
	}

	e.writeBytes(qoiEndMarker)
}

func (e *encoder) encodeBody() {
	if e.err != nil {
		return
	}

	index := [64]color.NRGBA{}

	width := e.m.Bounds().Dx()
	height := e.m.Bounds().Dy()

	run := 0
	pxPrev := color.NRGBA{0, 0, 0, 255}
	px := pxPrev

	pxLen := width * height

	img := imgconv.ToNRGBA(e.m)

	for pxPos := 0; pxPos < pxLen; pxPos++ {
		if e.err != nil {
			return
		}

		x := pxPos % width
		y := pxPos / width
		px = img.NRGBAAt(x, y)

		if px == pxPrev {
			run++
			if run == qoiMaxRunSize || pxPos == pxLen-1 {
				chunk := opRUN | uint8(run-1)
				e.writeBytes(chunk)
				run = 0
			}
		} else {
			if run > 0 {
				chunk := opRUN | uint8(run-1)
				e.writeBytes(chunk)
				run = 0
			}

			indexPos := hash(px)
			if index[indexPos] == px {
				chunk := (opINDEX | indexPos)
				e.writeBytes(chunk)
			} else {
				index[indexPos] = px

				if px.A == pxPrev.A {
					var vr int8 = int8(px.R) - int8(pxPrev.R)
					var vg int8 = int8(px.G) - int8(pxPrev.G)
					var vb int8 = int8(px.B) - int8(pxPrev.B)

					vgR := vr - vg
					vgB := vb - vg

					if vr > -3 && vr < 2 && vg > -3 && vg < 2 && vb > -3 && vb < 2 {
						chunk := opDIFF | (uint8(vr+2) << 4) | (uint8(vg+2) << 2) | uint8(vb+2)
						e.writeBytes(chunk)
					} else if vgR > -9 && vgR < 8 &&
						vg > -33 && vg < 32 &&
						vgB > -9 && vgB < 8 {
						chunk := make([]byte, 2)
						chunk[0] = opLUMA | uint8(vg+32)
						chunk[1] = (uint8(vgR+8) << 4) | uint8(vgB+8)
						e.writeBytes(chunk)
					} else {
						rgb := make([]byte, 4)
						rgb[0] = opRGB
						rgb[1] = px.R
						rgb[2] = px.G
						rgb[3] = px.B
						e.writeBytes(rgb)
					}
				} else {
					rgba := make([]byte, 5)
					rgba[0] = opRGBA
					rgba[1] = px.R
					rgba[2] = px.G
					rgba[3] = px.B
					rgba[4] = px.A
					e.writeBytes(rgba)
				}
			}
		}
		pxPrev = px
	}
}

func (e *encoder) writeBytes(data any) {
	err := binary.Write(e.w, binary.BigEndian, data)
	if err != nil {
		e.err = err
	}
}
