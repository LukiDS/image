package qoi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

type qoiHeader struct {
	width      uint32
	height     uint32
	channels   uint8
	colorspace uint8
}

type decoder struct {
	h   qoiHeader
	r   io.Reader
	m   *image.NRGBA
	err error
}

func (d *decoder) decodeHeader() {
	h := make([]byte, qoiHeaderSize)
	if d.readBytes(&h); d.err != nil {
		return
	}

	d.h.width = binary.BigEndian.Uint32(h[4:8])
	d.h.height = binary.BigEndian.Uint32(h[8:12])

	d.h.channels = h[12]
	d.h.colorspace = h[13]

	if d.h.channels < 3 || d.h.channels > 4 || d.h.colorspace > 1 || !bytes.Equal(h[:4], []byte(qoiMagic)) {
		d.err = fmt.Errorf("image not valid qoi file")
		return
	}

	if int(d.h.width) <= 0 || int(d.h.height) <= 0 || d.h.width*d.h.height > qoiMaxPixels {
		d.err = fmt.Errorf("image size invalid")
		return
	}
}

func (d *decoder) decode() {
	if d.err != nil {
		return
	}
	d.m = image.NewNRGBA(image.Rect(0, 0, int(d.h.width), int(d.h.height)))

	pixelBuffer := [qoiMaxBufferSize]color.NRGBA{}
	prevPixel := color.NRGBA{0, 0, 0, 255}

	width := d.m.Bounds().Dx()
	height := d.m.Bounds().Dy()

	maxPixel := width * height
	run := 0

	for pxPos := 0; pxPos < maxPixel; pxPos++ {
		if d.err != nil {
			return
		}

		x := pxPos % width
		y := pxPos / width

		if run > 0 {
			run--
			d.m.SetNRGBA(x, y, prevPixel)

			continue
		}

		var b1 byte
		d.readBytes(&b1)

		if b1 == opRGB {
			var rgb [3]byte
			d.readBytes(&rgb)

			prevPixel.R = rgb[0]
			prevPixel.G = rgb[1]
			prevPixel.B = rgb[2]

		} else if b1 == opRGBA {
			var rgba [4]byte
			d.readBytes(&rgba)

			prevPixel.R = rgba[0]
			prevPixel.G = rgba[1]
			prevPixel.B = rgba[2]
			prevPixel.A = rgba[3]

		} else if (b1 & maskOP) == opINDEX {
			prevPixel = pixelBuffer[(b1 & mask6)]

		} else if (b1 & maskOP) == opDIFF {
			prevPixel.R += ((b1 >> 4) & mask2) - 2
			prevPixel.G += ((b1 >> 2) & mask2) - 2
			prevPixel.B += ((b1 >> 0) & mask2) - 2

		} else if (b1 & maskOP) == opLUMA {
			var b2 byte
			d.readBytes(&b2)

			vg := (b1 & mask6) - 32

			prevPixel.R += vg - 8 + ((b2 >> 4) & mask4)
			prevPixel.G += vg
			prevPixel.B += vg - 8 + ((b2 >> 0) & mask4)

		} else if (b1 & maskOP) == opRUN {
			run = int(b1 & mask6)
		}

		pixelBuffer[hash(prevPixel)] = prevPixel
		d.m.SetNRGBA(x, y, prevPixel)
	}
}

func (d *decoder) decodePadding() {
	if d.err != nil {
		return
	}

	padding := make([]byte, len(qoiEndMarker))
	if d.readBytes(&padding); d.err != nil {
		return
	}

	if !bytes.Equal(padding, qoiEndMarker) {
		d.err = fmt.Errorf("unexpected EOF")
		return
	}

	var lastByte byte
	if d.readBytes(&lastByte); d.err != io.EOF {
		d.err = fmt.Errorf("invalid qoi size")
		return
	}

	d.err = nil
}

func (d *decoder) readBytes(data any) {
	d.err = binary.Read(d.r, binary.BigEndian, data)
}

func DecodeConfig(r io.Reader) (image.Config, error) {
	d := decoder{
		r: r,
	}

	d.decodeHeader()
	if d.err != nil {
		return image.Config{}, d.err
	}

	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(d.h.width),
		Height:     int(d.h.height),
	}, nil
}

func Decode(r io.Reader) (image.Image, error) {
	d := decoder{
		r: r,
	}

	d.decodeHeader()
	d.decode()
	d.decodePadding()

	if d.err != nil {
		return nil, d.err
	}

	return d.m, nil
}
