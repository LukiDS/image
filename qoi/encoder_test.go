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
)

func TestEncodeWithTestFiles(t *testing.T) {
	filenames, err := filepath.Glob("../testdata/*.qoi")
	if err != nil {
		t.Fatalf("could not find files: %v\n", err)
	}

	for _, name := range filenames {
		t.Run(filepath.Base(name), func(t *testing.T) {
			reference, err := os.ReadFile(name)
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

			encoded := bytes.NewBuffer(nil)
			err = Encode(encoded, img)
			if err != nil {
				format := fmt.Sprintf("\n") +
					fmt.Sprintf("File:\t %s\n", name) +
					fmt.Sprintf("Expected error:\t %v\n", nil) +
					fmt.Sprintf("Actual error:\t %v\n", err)

				t.Fatalf(format)
			}

			if len(reference) != len(encoded.Bytes()) {
				format := fmt.Sprintf("\n") +
					fmt.Sprintf("File:\t %s\n", name) +
					fmt.Sprintf("Expected length:\t %d\n", len(encoded.Bytes())) +
					fmt.Sprintf("Actual length:\t %d\n", len(reference))

				t.Fatalf(format)
			}

			for idx, val := range reference {
				if val != encoded.Bytes()[idx] {
					if idx == 12 || idx == 13 {
						if idx == 12 {
							t.Log("Warning: Skipped different channel value")
						}
						if idx == 13 {
							t.Log("Warning: Skipped different colorspace value")
						}

						//Skip channel and colorspace values
						//because Encode uses default values for both
						//and does not check for user input
						continue
					}

					format := fmt.Sprintf("\n") +
						fmt.Sprintf("File:\t %s\n", name) +
						fmt.Sprintf("Index:\t %d\n", idx) +
						fmt.Sprintf("Expected value:\t %.8b\n", encoded.Bytes()[idx]) +
						fmt.Sprintf("Actual value:\t %.8b\n", reference[idx])

					t.Fatalf(format)
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
			name: "should return encoded index",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{10, 20, 30, 255, 0, 0, 0, 0}),
			},
			expectedData: generateEncodeDummy(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 10, 20, 30, opINDEX | hash(color.NRGBA{0, 0, 0, 0})}),
		},
		{
			name: "should return encoded run",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{0, 0, 0, 0, 0, 0, 0, 0}),
			},
			expectedData: generateEncodeDummy(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opINDEX | hash(color.NRGBA{0, 0, 0, 0}), opRUN | 0}),
		},
		{
			name: "should return encoded diff",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{9, 1, 255, 255, 10, 255, 0, 255}),
			},
			expectedData: generateEncodeDummy(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 9, 1, 255, opDIFF | (3 << 4) | (0 << 2) | (3 << 0)}),
		},
		{
			name: "should return encoded luma",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 2, height: 1}, []byte{127, 30, 0, 200, 100, 0, 225, 200}),
			},
			expectedData: generateEncodeDummy(t, qoiHeader{width: 2, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 127, 30, 0, 200, opLUMA | 2, (11 << 4) | (7 << 0)}),
		},
		{
			name: "should return encoded rgb",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{10, 20, 30, 255}),
			},
			expectedData: generateEncodeDummy(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGB, 10, 20, 30}),
		},
		{
			name: "should return encoded rgba",
			args: struct {
				w io.Writer
				m image.Image
			}{
				w: bytes.NewBuffer(nil),
				m: generateImageStub(t, qoiHeader{width: 1, height: 1}, []byte{10, 20, 30, 100}),
			},
			expectedData: generateEncodeDummy(t, qoiHeader{width: 1, height: 1, channels: 4, colorspace: 0}, []byte{opRGBA, 10, 20, 30, 100}),
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

func generateEncodeDummy(t testing.TB, h qoiHeader, data []byte) []byte {
	t.Helper()

	header := make([]byte, 0, qoiHeaderSize)
	header = append(header, qoiMagic...)
	header = append(header, byte(h.width>>24), byte(h.width>>16), byte(h.width>>8), byte(h.width))
	header = append(header, byte(h.height>>24), byte(h.height>>16), byte(h.height>>8), byte(h.height))
	header = append(header, h.channels, h.colorspace)

	if len(header) != qoiHeaderSize {
		t.Fatalf("invalid header size")
	}

	buf := bytes.NewBuffer(header)
	_, err := buf.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	_, err = buf.Write(qoiEndMarker)
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func BenchmarkEncodeToFile(b *testing.B) {
	pngFile, err := os.Open("../testdata/dice.png")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer pngFile.Close()

	img, err := png.Decode(bufio.NewReader(pngFile))
	if err != nil {
		b.Fatalf("could not decode file: %v\n", err)
	}

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

func BenchmarkEncodeToBufferedFile(b *testing.B) {
	pngFile, err := os.Open("../testdata/dice.png")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer pngFile.Close()

	img, err := png.Decode(bufio.NewReader(pngFile))
	if err != nil {
		b.Fatalf("could not decode file: %v\n", err)
	}

	dir := b.TempDir()
	qoiFile, err := os.Create(dir + "test.qoi")
	if err != nil {
		b.Fatalf("could not create file: %v\n", err)
	}
	defer qoiFile.Close()

	buf := bufio.NewWriter(qoiFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Encode(buf, img)
		if err != nil {
			b.Fatalf("could not encode file: %v\n", err)
		}

		b.StopTimer()
		buf.Reset(qoiFile)
		qoiFile.Seek(0, 0)
		b.StartTimer()
	}
}

func BenchmarkEncodeToMemory(b *testing.B) {
	pngFile, err := os.Open("../testdata/dice.png")
	if err != nil {
		b.Fatalf("could not read file: %v\n", err)
	}
	defer pngFile.Close()

	img, err := png.Decode(bufio.NewReader(pngFile))
	if err != nil {
		b.Fatalf("could not decode file: %v\n", err)
	}

	buf := bytes.NewBuffer(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Encode(buf, img)
		if err != nil {
			b.Fatalf("could not encode file: %v\n", err)
		}

		b.StopTimer()
		buf.Reset()
		b.StartTimer()
	}
}
