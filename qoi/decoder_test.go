package qoi

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukiDS/image/imgconv"
)

func TestDecodeWithTestFiles(t *testing.T) {
	filenames, err := filepath.Glob("../testdata/*.qoi")
	if err != nil {
		t.Fatalf("could not find files: %v\n", err)
	}

	for _, name := range filenames {
		t.Run(filepath.Base(name), func(t *testing.T) {
			qoiFile, err := os.Open(name)
			if err != nil {
				t.Fatalf("could not read file: %v\n", err)
			}
			defer qoiFile.Close()

			img, err := Decode(bufio.NewReader(qoiFile))
			if err != nil {
				t.Fatalf("could not decode file: %v\n", err)
			}

			pngFile, err := os.Open(strings.TrimSuffix(name, filepath.Ext(name)) + ".png")
			if err != nil {
				t.Fatalf("could not read file: %v\n", err)
			}
			defer pngFile.Close()

			ref, err := png.Decode(bufio.NewReader(pngFile))
			if err != nil {
				t.Fatalf("could not decode file: %v\n", err)
			}

			//qoi en-/decoder always returns an NRGBA image
			if ref.ColorModel() != color.NRGBAModel {
				ref = imgconv.ToNRGBA(ref)
			}

			format := fmt.Sprintf("\nFile:\t %s\n", name)
			assertEqualImage(t, ref, img, format)
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return an error if width is zero",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 0, height: 1, channels: 4, colorspace: 0}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if height is zero",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 0, channels: 3, colorspace: 1}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if channel is less than 3",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 0, colorspace: 1}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if channel is greater than 4",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 5, colorspace: 1}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if colorspace is greater than 1",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 2}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if reader does not start with qoiMagic",
			args: struct{ r io.Reader }{
				r: bytes.NewBuffer(nil),
			},
			expectError: true,
		},
		{
			name: "should return an error if reader does not contain enough bytes specified with height and width",
			args: struct{ r io.Reader }{
				r: generateReaderStubWithoutPadding(t, qoiHeader{width: 1, height: 5, channels: 4, colorspace: 0}, []byte{opRGB, 255, 255, 255}),
			},
			expectError: true,
		},
		{
			name: "should return an error if reader does not contain qoiEndMarker",
			args: struct{ r io.Reader }{
				r: generateReaderStubWithoutPadding(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 255, 255, 255}),
			},
			expectError: true,
		},
		{
			name: "should return an error if reader does not end with valid qoiEndMarker",
			args: struct{ r io.Reader }{
				r: generateReaderStubWithoutPadding(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{
					/* OP        */ opRGB, 255, 255, 255,
					/* EndMarker */ 0, 0, 0, 0, 0, 0, 0, 4,
				}),
			},
			expectError: true,
		},
		{
			name: "should return an error if reader does not end with qoiEndMarker",
			args: struct{ r io.Reader }{
				r: generateReaderStubWithoutPadding(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{
					/* OP           */ opRGB, 255, 255, 255,
					/* EndMarker    */ 0, 0, 0, 0, 0, 0, 0, 1,
					/* TrailingByte */ 255,
				}),
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)
		})
	}
}

func TestDecodeConfig(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError    bool
		expectedConfig image.Config
	}{
		{
			name: "should return an error if width is zero",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 0, height: 1, channels: 4, colorspace: 0}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if height is zero",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 0, channels: 3, colorspace: 1}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if channel is less than 3",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 0, colorspace: 1}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if channel is greater than 4",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 5, colorspace: 1}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if colorspace is greater than 1",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 2}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if reader does not start with qoiMagic",
			args: struct{ r io.Reader }{
				r: bytes.NewBuffer(nil),
			},
			expectError: true,
		},
		{
			name: "should return decoded config",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 255, 255, 255}),
			},
			expectedConfig: image.Config{ColorModel: color.NRGBAModel, Width: 1, Height: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualConfig, err := DecodeConfig(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := fmt.Sprintf("\n") +
					fmt.Sprintf("DecodeConfig(r io.Reader) = (%+v,%v)\n", actualConfig, err) +
					fmt.Sprintf("Expected error:\t %t\n", test.expectError) +
					fmt.Sprintf("Actual error:\t %t\n", actualError)
				t.Errorf(format)
			}

			if actualConfig != test.expectedConfig {
				format := fmt.Sprintf("\n") +
					fmt.Sprintf("DecodeConfig(r io.Reader) = (%+v,%v)\n", actualConfig, err) +
					fmt.Sprintf("Expected config:\t %+v\n", test.expectedConfig) +
					fmt.Sprintf("Actual config:\t %+v\n", actualConfig)
				t.Errorf(format)
			}

		})
	}
}

func TestDecodeIndex(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return default pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{(opINDEX | 10)}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{0, 0, 0, 0}),
		},
		{
			name: "should return pixel at index",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 1, 2, 3, 200, (opINDEX | hash(color.NRGBA{1, 2, 3, 200}))}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{1, 2, 3, 200, 1, 2, 3, 200}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)

		})
	}
}

func TestDecodeRun(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return default pixels",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 3, height: 1, channels: 4, colorspace: 0}, []byte{opRUN | 2}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 3, height: 1}, []byte{0, 0, 0, 255, 0, 0, 0, 255, 0, 0, 0, 255}),
		},
		{
			name: "should return at least one pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRUN | 0}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{0, 0, 0, 255}),
		},
		{
			name: "should return last pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 4, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 1, 2, 3, opRUN | 2}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 4, height: 1}, []byte{1, 2, 3, 255, 1, 2, 3, 255, 1, 2, 3, 255, 1, 2, 3, 255}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)

		})
	}
}

func TestDecodeDiff(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return diff of default pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opDIFF | 0}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{254, 254, 254, 255}),
		},
		{
			name: "should return diff of last pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 100, 200, 50, opDIFF | 0b00_10_11_00}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{100, 200, 50, 255, 100, 201, 48, 255}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)

		})
	}
}

func TestDecodeLuma(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return luma of default pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opLUMA | 0, 0b0000_0000}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{216, 224, 216, 255}),
		},
		{
			name: "should return luma of last pixel",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 100, 200, 50, opLUMA | 0b00_100000, 0b1001_0101}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{100, 200, 50, 255, 101, 200, 47, 255}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)

		})
	}
}

func TestDecodeRGB(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return rgb",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 1, 2, 3}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{1, 2, 3, 255}),
		},
		{
			name: "should return multiple rgb's",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 1, 2, 3, opRGB, 10, 20, 30}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{1, 2, 3, 255, 10, 20, 30, 255}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)

		})
	}
}

func TestDecodeRGBA(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			r io.Reader
		}
		expectError   bool
		expectedImage image.Image
	}{
		{
			name: "should return rgba",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 1, 2, 3, 4}),
			},
			expectedImage: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{1, 2, 3, 4}),
		},
		{
			name: "should return multiple rgba's",
			args: struct{ r io.Reader }{
				r: generateReaderStub(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 1, 2, 3, 4, opRGBA, 10, 20, 30, 40}),
			},
			expectError:   false,
			expectedImage: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{1, 2, 3, 4, 10, 20, 30, 40}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualImage, err := Decode(test.args.r)
			if actualError := err != nil; actualError != test.expectError {
				format := getErrorFormatMsg(test.expectError, actualError, actualImage, err)
				t.Errorf(format)
			}

			format := getImageFormatMsg(test.expectedImage, actualImage, err)
			assertEqualImage(t, test.expectedImage, actualImage, format)

		})
	}
}

/*
	Utils, Stubs, Asserts
*/

func getImageFormatMsg(expected, actual image.Image, err error) string {
	return fmt.Sprintf("\n") +
		fmt.Sprintf("Decode(r io.Reader) = (%+v,%v)\n", actual, err) +
		fmt.Sprintf("Expected image:\t %+v\n", expected) +
		fmt.Sprintf("Actual image:\t %+v\n", actual)
}

func getErrorFormatMsg(expected, actual bool, actualImage image.Image, actualError error) string {
	return fmt.Sprintf("\n") +
		fmt.Sprintf("Decode(r io.Reader) = (%+v,%v)\n", actualImage, actualError) +
		fmt.Sprintf("Expected error:\t %t\n", expected) +
		fmt.Sprintf("Actual error:\t %t\n", actual)
}

func generateHeader(t testing.TB, h qoiHeader) []byte {
	t.Helper()

	buf := bytes.NewBuffer(nil)

	if err := writeBytes(buf, []byte(qoiMagic)); err != nil {
		t.Fatal(err)
	}
	if err := writeBytes(buf, h.width); err != nil {
		t.Fatal(err)
	}
	if err := writeBytes(buf, h.height); err != nil {
		t.Fatal(err)
	}
	if err := writeBytes(buf, h.channels); err != nil {
		t.Fatal(err)
	}
	if err := writeBytes(buf, h.colorspace); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func generateImageStub(t testing.TB, h qoiHeader, testdata []byte) image.Image {
	t.Helper()

	m := image.NewNRGBA(image.Rect(0, 0, int(h.width), int(h.height)))
	m.Pix = testdata

	return m
}

func generateReaderStub(t testing.TB, h qoiHeader, testdata []byte) io.Reader {
	t.Helper()
	buf := bytes.NewBuffer(nil)

	if err := writeBytes(buf, generateHeader(t, h)); err != nil {
		t.Fatal(err)
	}

	if err := writeBytes(buf, testdata); err != nil {
		t.Fatal(err)
	}

	if err := writeBytes(buf, qoiEndMarker); err != nil {
		t.Fatal(err)
	}

	return buf
}
func generateReaderStubWithoutPadding(t testing.TB, h qoiHeader, testdata []byte) io.Reader {
	t.Helper()
	buf := bytes.NewBuffer(nil)

	if err := writeBytes(buf, generateHeader(t, h)); err != nil {
		t.Fatal(err)
	}

	if err := writeBytes(buf, testdata); err != nil {
		t.Fatal(err)
	}

	return buf
}

func assertEqualImage(t testing.TB, expected, actual image.Image, format string) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}

	if expected == nil {
		f := fmt.Sprintf("%s", format) +
			fmt.Sprintf("Assert image:\t unexpected image")
		t.Fatalf(f)
	}

	if actual == nil {
		f := fmt.Sprintf("%s", format) +
			fmt.Sprintf("Assert image:\t unexpected nil image")
		t.Fatalf(f)
	}

	if expected.Bounds() != actual.Bounds() {
		f := fmt.Sprintf("%s", format) +
			fmt.Sprintf("Assert image:\t different image dimensions: Expected: %+v - Actual: %+v\n", expected.Bounds(), actual.Bounds())
		t.Fatalf(f)
	}

	if expected.ColorModel() != actual.ColorModel() {
		f := fmt.Sprintf("%s", format) +
			fmt.Sprintf("Assert image:\t different color model\n")
		t.Fatalf(f)
	}

	for x := expected.Bounds().Min.X; x < expected.Bounds().Max.X; x++ {
		for y := expected.Bounds().Min.Y; y < expected.Bounds().Max.Y; y++ {
			if expected.At(x, y) != actual.At(x, y) {
				f := fmt.Sprintf("%s", format) +
					fmt.Sprintf("Assert image:\t different pixel at x=%d, y=%d: Expected: %+v - Actual: %+v\n", x, y, expected.At(x, y), actual.At(x, y))
				t.Fatalf(f)
			}
		}
	}
}

func BenchmarkDecodeFromFile(b *testing.B) {
	qoiFile, err := os.Open("../testdata/dice.qoi")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer qoiFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decode(qoiFile)
		if err != nil {
			b.Fatalf("could not decode file: %v\n", err)
		}
		b.StopTimer()
		qoiFile.Seek(0, 0)
		b.StartTimer()
	}
}

func BenchmarkDecodeFromFileBuffered(b *testing.B) {
	qoiFile, err := os.Open("../testdata/dice.qoi")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer qoiFile.Close()

	buf := bufio.NewReader(qoiFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decode(buf)
		if err != nil {
			b.Fatalf("could not decode file: %v\n", err)
		}
		b.StopTimer()
		qoiFile.Seek(0, 0)
		b.StartTimer()
	}
}

func BenchmarkDecodeFromMemory(b *testing.B) {
	qoiData, err := os.ReadFile("../testdata/dice.qoi")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := bytes.NewBuffer(qoiData)
		b.StartTimer()
		_, err := Decode(buf)
		if err != nil {
			b.Fatalf("could not decode file: %v\n", err)
		}
	}
}
