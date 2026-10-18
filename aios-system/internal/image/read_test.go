package image

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadi/aios-system/internal/tools"
)

func TestReadImageTool_PNG(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, dir, "a.png", 16, 12)
	g := ReadImageTool{}
	res, err := g.Run(map[string]any{"path": "a.png"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("not OK: %s", res.Summary)
	}
	img := res.Output["image"].(Image)
	if img.Width != 16 || img.Height != 12 {
		t.Fatalf("dims = %dx%d", img.Width, img.Height)
	}
	if img.MIMEType != "image/png" {
		t.Fatalf("mime = %q", img.MIMEType)
	}
}

func TestReadImageTool_JPEG(t *testing.T) {
	dir := t.TempDir()
	writeJPEG(t, dir, "a.jpg", 32, 24)
	g := ReadImageTool{}
	res, err := g.Run(map[string]any{"path": "a.jpg"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	img := res.Output["image"].(Image)
	if img.Width != 32 || img.Height != 24 || img.MIMEType != "image/jpeg" {
		t.Fatalf("got %+v", img)
	}
}

func TestReadImageTool_GIF(t *testing.T) {
	dir := t.TempDir()
	writeGIF(t, dir, "a.gif", 8, 4)
	g := ReadImageTool{}
	res, err := g.Run(map[string]any{"path": "a.gif"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	img := res.Output["image"].(Image)
	if img.MIMEType != "image/gif" {
		t.Fatalf("mime = %q", img.MIMEType)
	}
}

func TestReadImageTool_WebP_VP8(t *testing.T) {
	dir := t.TempDir()
	writeVP8WebP(t, dir, "a.webp", 100, 50)
	g := ReadImageTool{}
	res, err := g.Run(map[string]any{"path": "a.webp"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	img := res.Output["image"].(Image)
	if img.Width != 100 || img.Height != 50 {
		t.Fatalf("dims = %dx%d", img.Width, img.Height)
	}
}

func TestReadImageTool_WebP_VP8L(t *testing.T) {
	dir := t.TempDir()
	writeVP8LWebP(t, dir, "a.webp", 200, 100)
	g := ReadImageTool{}
	res, err := g.Run(map[string]any{"path": "a.webp"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	img := res.Output["image"].(Image)
	if img.Width != 200 || img.Height != 100 {
		t.Fatalf("dims = %dx%d", img.Width, img.Height)
	}
}

func TestReadImageTool_WebP_VP8X(t *testing.T) {
	dir := t.TempDir()
	writeVP8XWebP(t, dir, "a.webp", 300, 150)
	g := ReadImageTool{}
	res, err := g.Run(map[string]any{"path": "a.webp"}, tools.Context{Workspace: dir})
	if err != nil {
		t.Fatal(err)
	}
	img := res.Output["image"].(Image)
	if img.Width != 300 || img.Height != 150 {
		t.Fatalf("dims = %dx%d", img.Width, img.Height)
	}
}

func TestReadImageTool_TooLarge(t *testing.T) {
	dir := t.TempDir()
	body := make([]byte, 200_000)
	body[0] = 0x89
	body[1] = 'P'
	body[2] = 'N'
	body[3] = 'G'
	if err := os.WriteFile(filepath.Join(dir, "big.png"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	g := ReadImageTool{MaxSize: 1000}
	_, err := g.Run(map[string]any{"path": "big.png"}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadImageTool_Missing(t *testing.T) {
	dir := t.TempDir()
	g := ReadImageTool{}
	_, err := g.Run(map[string]any{"path": "nope.png"}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadImageTool_MissingPath(t *testing.T) {
	g := ReadImageTool{}
	_, err := g.Run(map[string]any{}, tools.Context{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadImageTool_UnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := ReadImageTool{}
	_, err := g.Run(map[string]any{"path": "a.txt"}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadImageTool_BadMagic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.png"), []byte("not really png"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := ReadImageTool{}
	_, err := g.Run(map[string]any{"path": "a.png"}, tools.Context{Workspace: dir})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadImageTool_PathOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writePNG(t, outside, "a.png", 1, 1)
	g := ReadImageTool{}
	_, err := g.Run(map[string]any{"path": outside}, tools.Context{Workspace: dir})
	if err == nil {
		t.Fatal("expected path-rejected error")
	}
}

func TestReadImageTool_NoWorkspace(t *testing.T) {
	// When ctx.Workspace is empty and tools.Workspace()
	// is also empty, the tool uses the current working
	// directory. We use a directory that doesn't have the
	// file so the call still errors cleanly.
	g := ReadImageTool{}
	_, err := g.Run(map[string]any{"path": "nope.png"}, tools.Context{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadImageTool_Name(t *testing.T) {
	if (ReadImageTool{}).Name() != "read_image" {
		t.Fatal("name")
	}
}

func TestReadImageFile_NotFound(t *testing.T) {
	_, err := readImageFile("/no/such/file.png", 1024)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDetectMIME_BadExt(t *testing.T) {
	_, err := detectMIME("a.bin", []byte("xxx"))
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v", err)
	}
}

func TestDetectMIME_ShortData(t *testing.T) {
	_, err := detectMIME("a.png", []byte{1, 2, 3})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v", err)
	}
}

func TestMatchesMagic(t *testing.T) {
	cases := []struct {
		mime  string
		data  []byte
		match bool
	}{
		{"image/png", []byte{0x89, 'P', 'N', 'G', 0, 0, 0, 0}, true},
		{"image/png", []byte{0, 0, 0, 0, 0, 0, 0, 0}, false},
		{"image/jpeg", []byte{0xFF, 0xD8, 0xFF, 0, 0, 0, 0, 0}, true},
		{"image/jpeg", []byte{0, 0, 0, 0, 0, 0, 0, 0}, false},
		{"image/gif", []byte("GIF89a..."), true},
		{"image/gif", []byte("GIF87a..."), true},
		{"image/gif", []byte("GIX89a..."), false},
		{"image/webp", []byte("RIFF\x00\x00\x00\x00WEBP"), true},
		{"image/webp", []byte("RIFF\x00\x00\x00\x00JUNK"), false},
		{"unknown", []byte{1, 2, 3, 4, 5, 6, 7, 8}, false},
		{"image/png", []byte{1, 2}, false},
	}
	for _, c := range cases {
		if got := matchesMagic(c.mime, c.data); got != c.match {
			t.Fatalf("matchesMagic(%q, %v) = %v, want %v", c.mime, c.data, got, c.match)
		}
	}
}

func TestWebPDimensions_BadContainer(t *testing.T) {
	_, _, err := webpDimensions([]byte("not a webp file at all"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebPDimensions_TooShort(t *testing.T) {
	_, _, err := webpDimensions([]byte{1, 2, 3, 4})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebPDimensions_UnknownChunk(t *testing.T) {
	data := []byte("RIFF\x00\x00\x00\x00WEBPJUNK")
	_, _, err := webpDimensions(data)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebPDimensions_VP8_BadFrameTag(t *testing.T) {
	buf := make([]byte, 30)
	copy(buf[0:4], "RIFF")
	copy(buf[8:12], "WEBP")
	copy(buf[12:16], "VP8 ")
	buf[16] = 10
	// start code at offset 20 is zero (not 0x9D 0x01 0x2A)
	_, _, err := webpDimensions(buf)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebPDimensions_VP8L_BadSignature(t *testing.T) {
	buf := make([]byte, 30)
	copy(buf[0:4], "RIFF")
	copy(buf[8:12], "WEBP")
	copy(buf[12:16], "VP8L")
	buf[20] = 0
	_, _, err := webpDimensions(buf)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebPDimensions_VP8X_TooShort(t *testing.T) {
	buf := make([]byte, 20)
	copy(buf[0:4], "RIFF")
	copy(buf[8:12], "WEBP")
	copy(buf[12:16], "VP8X")
	_, _, err := webpDimensions(buf)
	if err == nil {
		t.Fatal("expected error")
	}
}
