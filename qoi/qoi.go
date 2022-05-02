package qoi

import (
	"image"
	"image/color"
)

const (
	/*
		2GB is the max file size that this implementation can safely handle.
		We guard against anything larger than that, assuming the worst case with 5 bytes per pixel,
		rounded down to a nice clean value.

		400 million pixels ought to be enough for anybody.
	*/
	qoiMaxPixels = 400_000_000
	qoiMagic     = "qoif"

	qoiDefaultChannel    uint8 = 4
	qoiDefaultColorSpace uint8 = 0

	qoiHeaderSize    = 14 //size in bytes
	qoiMaxBufferSize = 64
	qoiMaxRunSize    = 62
)

var qoiEndMarker = []byte{0, 0, 0, 0, 0, 0, 0, 1}

const (
	opINDEX uint8 = 0b00000000
	opDIFF  uint8 = 0b01000000
	opLUMA  uint8 = 0b10000000
	opRUN   uint8 = 0b11000000
	opRGB   uint8 = 0b11111110
	opRGBA  uint8 = 0b11111111
)

const (
	maskOP uint8 = 0b11000000
	mask6  uint8 = 0b00111111
	mask4  uint8 = 0b00001111
	mask2  uint8 = 0b00000011
)

func init() {
	image.RegisterFormat("qoi", qoiMagic, Decode, DecodeConfig)
}

func hash(c color.NRGBA) uint8 {
	return (3*c.R + 5*c.G + 7*c.B + 11*c.A) % qoiMaxBufferSize
}
