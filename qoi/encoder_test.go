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

func TestEncodeWithTestFiles(t *testing.T) {
	filenames, err := filepath.Glob("../testdata/*.qoi")
	if err != nil {
		t.Fatalf("could not find files: %v\n", err)
	}

	for _, name := range filenames {
		t.Run(filepath.Base(name), func(t *testing.T) {
			ref, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("could not read file: %v\n", err)
			}

			pngFile, err := os.Open(strings.TrimSuffix(name, filepath.Ext(name)) + ".png")
			if err != nil {
				t.Fatalf("could not read file: %v\n", err)
			}
			defer pngFile.Close()

			img, err := png.Decode(bufio.NewReader(pngFile))
			if err != nil {
				t.Fatalf("could not decode file: %v\n", err)
			}

			//qoi en-/decoder always returns an NRGBA image
			if img.ColorModel() != color.NRGBAModel {
				img = imgconv.ToNRGBA(img)
			}

			encoded := bytes.NewBuffer(nil)
			err = Encode(encoded, img)
			if err != nil {
				t.Fatalf("could not encode file: %v\n", err)
			}

			for idx, val := range ref {
				if val != encoded.Bytes()[idx] {
					if idx == 12 || idx == 13 {
						t.Log("Warning: Skipped different channel/colorspace value")
						//Skip channel and/or colorspace values
						//because Encode uses default values for both
						//and does not check for user input
						continue
					}
					t.Fatalf("\nFile:\t %s not equal encodings\nencoded: %d - %.8b\nreference: %d - %.8b\nidx: %d\n", name, len(encoded.Bytes()), encoded.Bytes()[idx], len(ref), val, idx)
				}
			}
		})
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			w io.Writer
			m image.Image
		}
		expectError  bool
		expectedData []byte
	}{
		{
			name: "should return an error if image is not in NRGBA format",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: io.Discard,
				m: image.NewRGBA(image.Rect(0, 0, 1, 1)),
			},
			expectError: true,
		},
		{
			name: "should return an error if width is zero",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: io.Discard,
				m: generateImageStub(t, qoiHeader{width: 0, height: 20}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return an error if height is zero",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: io.Discard,
				m: generateImageStub(t, qoiHeader{width: 20, height: 0}, []byte{}),
			},
			expectError: true,
		},
		{
			name: "should return encoded header",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{1, 2, 3, 4}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 1, 2, 3, 4}),
		},
		{
			name: "should return index",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{10, 20, 30, 255, 0, 0, 0, 0}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 10, 20, 30, opINDEX | hash(color.NRGBA{0, 0, 0, 0})}),
		},
		{
			name: "should return run",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{0, 0, 0, 0, 0, 0, 0, 0}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opINDEX | hash(color.NRGBA{0, 0, 0, 0}), opRUN | 0}),
		},
		{
			name: "should return diff",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{9, 1, 255, 255, 10, 255, 0, 255}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 9, 1, 255, opDIFF | (3 << 4) | (0 << 2) | (3 << 0)}),
		},
		{
			name: "should return luma",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{127, 30, 0, 200, 100, 0, 225, 200}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 127, 30, 0, 200, opLUMA | 2, (11 << 4) | (7 << 0)}),
		},
		{
			name: "should return rgb",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{10, 20, 30, 255}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 10, 20, 30}),
		},
		{
			name: "should return rgba",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{10, 20, 30, 100}),
			},
			expectedData: getEncodedHeader(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 10, 20, 30, 100}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Encode(test.args.w, test.args.m)
			if actualError := err != nil; actualError != test.expectError {
				format := fmt.Sprintf("\n") +
					fmt.Sprintf("Encode(w io.Writer, %v) = (%v)\n", test.args.m, err) +
					fmt.Sprintf("Expected error:\t %t\n", test.expectError) +
					fmt.Sprintf("Actual error:\t %t\n", actualError)

				t.Errorf(format)
			}

			if buf, ok := test.args.w.(*bytes.Buffer); ok {
				if !bytes.Equal(buf.Bytes(), test.expectedData) {
					format := fmt.Sprintf("\n") +
						fmt.Sprintf("Encode(w io.Writer, %v) = (%v)\n", test.args.m, err) +
						fmt.Sprintf("Expected data:\t %v\n", test.expectedData) +
						fmt.Sprintf("Actual data:\t %v\n", buf.Bytes())

					t.Errorf(format)
				}
			}
		})
	}
}

func getEncodedHeader(t testing.TB, h qoiHeader, data []byte) []byte {
	t.Helper()

	buf := bytes.NewBuffer(nil)
	err := writeBytes(buf, []byte(qoiMagic))
	if err != nil {
		t.Fatal(err)
	}
	err = writeBytes(buf, h.width)
	if err != nil {
		t.Fatal(err)
	}
	err = writeBytes(buf, h.height)
	if err != nil {
		t.Fatal(err)
	}
	err = writeBytes(buf, h.channels)
	if err != nil {
		t.Fatal(err)
	}
	err = writeBytes(buf, h.colorspace)
	if err != nil {
		t.Fatal(err)
	}

	if len(buf.Bytes()) != qoiHeaderSize {
		t.Fatalf("invalid header size")
	}

	_, err = buf.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	err = writeBytes(buf, qoiEndMarker)
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func BenchmarkEncodeFromFile(b *testing.B) {
	pngFile, err := os.Open("../testdata/dice.png")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer pngFile.Close()

	img, err := png.Decode(bufio.NewReader(pngFile))
	if err != nil {
		b.Fatalf("could not decode file: %v\n", err)
	}

	img = imgconv.ToNRGBA(img)
	dir := b.TempDir()
	qoiFile, err := os.Create(dir + "test.qoi")
	if err != nil {
		b.Fatalf("could not create file: %v\n", err)
	}
	defer qoiFile.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Encode(qoiFile, img)
		if err != nil {
			b.Fatalf("could not encode file: %v\n", err)
		}
		b.StopTimer()
		qoiFile.Seek(0, 0)
		b.StartTimer()
	}
}

func BenchmarkEncodeFromFileBuffered(b *testing.B) {
	pngFile, err := os.Open("../testdata/dice.png")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer pngFile.Close()

	img, err := png.Decode(bufio.NewReader(pngFile))
	if err != nil {
		b.Fatalf("could not decode file: %v\n", err)
	}

	img = imgconv.ToNRGBA(img)
	dir := b.TempDir()
	qoiFile, err := os.Create(dir + "test.qoi")
	if err != nil {
		b.Fatalf("could not create file: %v\n", err)
	}
	defer qoiFile.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Encode(bufio.NewWriter(qoiFile), img)
		if err != nil {
			b.Fatalf("could not encode file: %v\n", err)
		}
		b.StopTimer()
		qoiFile.Seek(0, 0)
		b.StartTimer()
	}
}

func BenchmarkEncodeFromMemory(b *testing.B) {
	pngFile, err := os.Open("../testdata/dice.png")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer pngFile.Close()

	img, err := png.Decode(bufio.NewReader(pngFile))
	if err != nil {
		b.Fatalf("could not decode file: %v\n", err)
	}

	img = imgconv.ToNRGBA(img)
	b.ResetTimer()
	buf := bytes.NewBuffer(nil)
	for i := 0; i < b.N; i++ {
		err := Encode(buf, img)
		if err != nil {
			b.Fatalf("could not encode file: %v\n", err)
		}
		b.StopTimer()
		buf = bytes.NewBuffer(nil)
		b.StartTimer()
	}
}
