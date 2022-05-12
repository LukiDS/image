package qoi

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

type qoiHeader struct {
	width      int
	height     int
	channels   uint8
	colorspace uint8
}

type decoder struct {
	m   *image.NRGBA
	buf *bufio.Reader
	h   qoiHeader
	err error
}

func (d *decoder) decodeHeader() {
	h := make([]byte, qoiHeaderSize)

	_, err := d.buf.Read(h)
	if err != nil {
		d.err = err
		return
	}

	d.h.width = int(binary.BigEndian.Uint32(h[4:8]))
	d.h.height = int(binary.BigEndian.Uint32(h[8:12]))

	d.h.channels = h[12]
	d.h.colorspace = h[13]

	if d.h.channels < 3 || d.h.channels > 4 || d.h.colorspace > 1 || !bytes.Equal(h[:4], []byte(qoiMagic)) {
		d.err = fmt.Errorf("image not valid qoi file")
		return
	}

	if d.h.width <= 0 || d.h.height <= 0 || d.h.width*d.h.height > qoiMaxPixels {
		d.err = fmt.Errorf("image size invalid")
		return
	}
}

func (d *decoder) decode() {
	if d.err != nil {
		return
	}

	d.m = image.NewNRGBA(image.Rect(0, 0, d.h.width, d.h.height))

	colorBuffer := [qoiMaxBufferSize]color.NRGBA{}
	pxPrev := color.NRGBA{0, 0, 0, 255}

	run := uint8(0)
	maxPixelPos := d.h.width * d.h.height
	for pxPos := 0; pxPos < maxPixelPos; pxPos++ {
		if d.err != nil {
			return
		}

		x := pxPos % d.h.width
		y := pxPos / d.h.width

		if run > 0 {
			run--
			d.m.SetNRGBA(x, y, pxPrev)

			continue
		}

		b1, err := d.buf.ReadByte()
		if err != nil {
			d.err = err
			return
		}

		switch {
		case b1 == opRGB:
			r, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}
			g, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}
			b, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}

			pxPrev.R = r
			pxPrev.G = g
			pxPrev.B = b

		case b1 == opRGBA:
			r, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}
			g, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}
			b, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}
			a, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}

			pxPrev.R = r
			pxPrev.G = g
			pxPrev.B = b
			pxPrev.A = a

		case (b1 & maskOP) == opINDEX:
			pxPrev = colorBuffer[(b1 & mask6)]

		case (b1 & maskOP) == opDIFF:
			pxPrev.R += ((b1 >> 4) & mask2) - 2
			pxPrev.G += ((b1 >> 2) & mask2) - 2
			pxPrev.B += ((b1 >> 0) & mask2) - 2

		case (b1 & maskOP) == opLUMA:
			b2, err := d.buf.ReadByte()
			if err != nil {
				d.err = err
				return
			}

			vg := (b1 & mask6) - 32

			pxPrev.R += vg - 8 + ((b2 >> 4) & mask4)
			pxPrev.G += vg
			pxPrev.B += vg - 8 + ((b2 >> 0) & mask4)

		case (b1 & maskOP) == opRUN:
			run = b1 & mask6
		}

		colorBuffer[hash(pxPrev)] = pxPrev
		d.m.SetNRGBA(x, y, pxPrev)
	}
}

func (d *decoder) decodePadding() {
	if d.err != nil {
		return
	}

	padding := make([]byte, len(qoiEndMarker))
	_, err := d.buf.Read(padding)
	if err != nil {
		d.err = err
		return
	}

	if !bytes.Equal(padding, qoiEndMarker) {
		d.err = fmt.Errorf("unexpected EOF")
		return
	}

	_, err = d.buf.ReadByte()
	if err != io.EOF {
		d.err = fmt.Errorf("invalid qoi size")
		return
	}
}

func DecodeConfig(r io.Reader) (image.Config, error) {
	d := decoder{
		buf: bufio.NewReader(r),
	}

	d.decodeHeader()
	if d.err != nil {
		return image.Config{}, d.err
	}

	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      d.h.width,
		Height:     d.h.height,
	}, nil
}

func Decode(r io.Reader) (image.Image, error) {
	d := decoder{
		buf: bufio.NewReader(r),
	}

	d.decodeHeader()
	d.decode()
	d.decodePadding()

	if d.err != nil {
		return nil, d.err
	}

	return d.m, nil
}
