package image

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writePNG writes a 1x1 PNG to dir/name. Returns the bytes.
func writePNG(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{0xff, 0x00, 0x00, 0xff})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeJPEG(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{0x00, 0xff, 0x00, 0xff})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeGIF(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.RGBA{0, 0, 0xff, 0xff}})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// writeVP8WebP writes a minimal VP8-keyframe WebP to
// dir/name with the given width and height. The frame
// uses the VP8 bitstream layout (not VP8L/VP8X).
func writeVP8WebP(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	buf := makeVP8(w, h)
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return buf
}

// makeVP8 builds a minimal valid RIFF/WEBP container with
// a VP8 chunk. The dimensions are encoded in the VP8 frame
// tag. The rest of the chunk is zero-padded.
func makeVP8(w, h int) []byte {
	// VP8 chunk payload starts with a 3-byte frame tag
	// marker (0x9D 0x01 0x2A), then 2-byte width + 2-byte
	// height (little-endian, 14 bits each).
	payload := make([]byte, 10)
	payload[0] = 0x9D
	payload[1] = 0x01
	payload[2] = 0x2A
	payload[3] = byte(w & 0xFF)
	payload[4] = byte((w >> 8) & 0x3F)
	payload[5] = byte(h & 0xFF)
	payload[6] = byte((h >> 8) & 0x3F)
	vp8 := make([]byte, 8+len(payload))
	copy(vp8[0:4], "VP8 ")
	vp8[4] = byte(len(payload) & 0xFF)
	vp8[5] = byte((len(payload) >> 8) & 0xFF)
	vp8[6] = byte((len(payload) >> 16) & 0xFF)
	vp8[7] = byte((len(payload) >> 24) & 0xFF)
	copy(vp8[8:], payload)
	chunkSize := uint32(len(vp8))
	riff := make([]byte, 12+len(vp8))
	copy(riff[0:4], "RIFF")
	riff[4] = byte(chunkSize & 0xFF)
	riff[5] = byte((chunkSize >> 8) & 0xFF)
	riff[6] = byte((chunkSize >> 16) & 0xFF)
	riff[7] = byte((chunkSize >> 24) & 0xFF)
	copy(riff[8:12], "WEBP")
	copy(riff[12:], vp8)
	return riff
}

// writeVP8LWebP writes a minimal VP8L-lossless WebP.
func writeVP8LWebP(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	buf := makeVP8L(w, h)
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return buf
}

func makeVP8L(w, h int) []byte {
	// VP8L: 4 bytes "VP8L" + 4 bytes chunk size + 0x2F signature +
	// 14-bit width-1, 14-bit height-1, then alpha + version
	vp8l := make([]byte, 8+5)
	copy(vp8l[0:4], "VP8L")
	vp8l[7] = 5
	vp8l[8] = 0x2F
	bits := uint32(w-1) | uint32(h-1)<<14
	vp8l[9] = byte(bits & 0xFF)
	vp8l[10] = byte((bits >> 8) & 0xFF)
	vp8l[11] = byte((bits >> 16) & 0xFF)
	vp8l[12] = byte((bits >> 24) & 0xFF)
	chunkSize := uint32(len(vp8l))
	riff := make([]byte, 12+len(vp8l))
	copy(riff[0:4], "RIFF")
	riff[4] = byte(chunkSize & 0xFF)
	riff[5] = byte((chunkSize >> 8) & 0xFF)
	riff[6] = byte((chunkSize >> 16) & 0xFF)
	riff[7] = byte((chunkSize >> 24) & 0xFF)
	copy(riff[8:12], "WEBP")
	copy(riff[12:], vp8l)
	return riff
}

// writeVP8XWebP writes a minimal VP8X (extended) WebP.
func writeVP8XWebP(t *testing.T, dir, name string, w, h int) []byte {
	t.Helper()
	buf := makeVP8X(w, h)
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return buf
}

func makeVP8X(w, h int) []byte {
	vp8x := make([]byte, 8+10)
	copy(vp8x[0:4], "VP8X")
	vp8x[4] = 10
	vp8x[5] = 0
	vp8x[6] = 0
	vp8x[7] = 0
	// VP8X payload: 4 bytes Rsv/I/E/X/A/R/R, then 3 bytes
	// width-1 and 3 bytes height-1 (LE).
	vp8x[8] = 0 // Rsv
	vp8x[9] = 0
	vp8x[10] = 0
	vp8x[11] = 0
	vp8x[12] = byte((w - 1) & 0xFF)
	vp8x[13] = byte(((w - 1) >> 8) & 0xFF)
	vp8x[14] = byte(((w - 1) >> 16) & 0xFF)
	vp8x[15] = byte((h - 1) & 0xFF)
	vp8x[16] = byte(((h - 1) >> 8) & 0xFF)
	vp8x[17] = byte(((h - 1) >> 16) & 0xFF)
	chunkSize := uint32(len(vp8x))
	riff := make([]byte, 12+len(vp8x))
	copy(riff[0:4], "RIFF")
	riff[4] = byte(chunkSize & 0xFF)
	riff[5] = byte((chunkSize >> 8) & 0xFF)
	riff[6] = byte((chunkSize >> 16) & 0xFF)
	riff[7] = byte((chunkSize >> 24) & 0xFF)
	copy(riff[8:12], "WEBP")
	copy(riff[12:], vp8x)
	return riff
}
