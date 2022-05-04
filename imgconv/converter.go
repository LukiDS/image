package imgconv

import (
	"image"
	"image/color"
)

// ToNRGBA converts any image m to an *image.NRGBA image.
// Any Image may be converted, but images that are not image.NRGBA might be converted lossily.
func ToNRGBA(m image.Image) *image.NRGBA {
	if m.ColorModel() == color.NRGBAModel {
		return m.(*image.NRGBA)
	}

	img := image.NewNRGBA(m.Bounds())

	for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
		for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
			px := m.At(x, y)
			px = color.NRGBAModel.Convert(px)
			img.Set(x, y, px)
		}
	}

	return img
}

// ToRGBA converts any image m to an *image.RGBA image.
// Any Image may be converted, but images that are not image.RGBA might be converted lossily.
func ToRGBA(m image.Image) *image.RGBA {
	if m.ColorModel() == color.RGBAModel {
		return m.(*image.RGBA)
	}

	img := image.NewRGBA(m.Bounds())

	for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
		for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
			px := m.At(x, y)
			px = color.RGBAModel.Convert(px)
			img.Set(x, y, px)
		}
	}

	return img
}
