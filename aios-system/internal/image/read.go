package image

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// MaxImageSize is the maximum image size in bytes
// (REQ-IMG-RDT-004). 20 MB.
const MaxImageSize = 20 * 1024 * 1024

// ReadImageTool reads an image file (REQ-IMG-RDT-001 to 005).
// The tool supports PNG, JPEG, GIF, and WebP. The WebP
// path parses the RIFF/VP8/VP8L/VP8X headers to extract
// the dimensions without bringing in a WebP codec.
type ReadImageTool struct {
	// MaxSize is the maximum image size in bytes. 0 means
	// MaxImageSize.
	MaxSize int64
}

func (ReadImageTool) Name() string { return "read_image" }

func (r ReadImageTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	rawPath, _ := args["path"].(string)
	if strings.TrimSpace(rawPath) == "" {
		return tools.Result{OK: false, Summary: "missing path"},
			fmt.Errorf("image: read: path is required")
	}
	ws := ctx.Workspace
	if ws == "" {
		ws, _ = tools.Workspace()
	}
	path, err := tools.ResolvePath(ws, rawPath)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad path"}, err
	}
	maxSize := r.MaxSize
	if maxSize <= 0 {
		maxSize = MaxImageSize
	}
	img, err := readImageFile(path, maxSize)
	if err != nil {
		return tools.Result{OK: false, Summary: "read failed"}, err
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"image": img},
		Summary: fmt.Sprintf("%s %dx%d", img.MIMEType, img.Width, img.Height),
	}, nil
}

// readImageFile reads path, decodes the dimensions, and
// returns the Image. The file is read in full and the
// bytes are returned alongside the decoded dims.
func readImageFile(path string, maxSize int64) (Image, error) {
	info, err := os.Stat(path)
	if err != nil {
		if stderrors.Is(err, fs.ErrNotExist) {
			return Image{}, fmt.Errorf("image: %s: not found", path)
		}
		return Image{}, fmt.Errorf("image: %s: %w", path, err)
	}
	if info.Size() > maxSize {
		return Image{}, &ImageTooLargeError{Path: path, Size: info.Size(), Max: maxSize}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Image{}, fmt.Errorf("image: %s: %w", path, err)
	}
	mime, err := detectMIME(path, data)
	if err != nil {
		return Image{}, err
	}
	width, height, err := decodeDimensions(mime, data)
	if err != nil {
		return Image{}, &UnsupportedFormatError{Path: path, Format: mime}
	}
	return Image{
		Path:     path,
		MIMEType: mime,
		Data:     data,
		Width:    width,
		Height:   height,
	}, nil
}

// decodeDimensions returns the (width, height) of the
// image. The PNG/JPEG/GIF path uses the stdlib
// image.DecodeConfig. The WebP path parses the RIFF
// container header to extract the dimensions.
func decodeDimensions(mime string, data []byte) (int, int, error) {
	if mime == "image/webp" {
		return webpDimensions(data)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// webpDimensions parses the WebP container. The file is
// "RIFF<size>WEBP<chunk><size><data>". The chunk is one
// of VP8, VP8L, or VP8X, and each carries the width and
// height in a small fixed region of the chunk header.
func webpDimensions(data []byte) (int, int, error) {
	if len(data) < 20 {
		return 0, 0, fmt.Errorf("webp: too short")
	}
	if string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, fmt.Errorf("webp: bad container")
	}
	chunk := string(data[12:16])
	switch chunk {
	case "VP8 ":
		// VP8 chunk: 8-byte header ("VP8 " + size) + payload.
		// The payload starts with a 3-byte frame tag
		// (0x9D 0x01 0x2A) at file offset 20 (= 12 RIFF
		// header + 8 chunk header). The 16-bit width and
		// height follow the tag at file offsets 23 and 25.
		if len(data) < 30 || data[20] != 0x9D || data[21] != 0x01 || data[22] != 0x2A {
			return 0, 0, fmt.Errorf("webp: bad VP8 frame tag")
		}
		w := int(data[23]) | int(data[24])<<8
		h := int(data[25]) | int(data[26])<<8
		return w & 0x3FFF, h & 0x3FFF, nil
	case "VP8L":
		// VP8L signature 0x2F is at file offset 20 (the
		// first byte of the payload). The 14-bit width-1
		// and 14-bit height-1 follow in 4 bytes (LE).
		if len(data) < 25 || data[20] != 0x2F {
			return 0, 0, fmt.Errorf("webp: bad VP8L signature")
		}
		b1 := uint32(data[21])
		b2 := uint32(data[22])
		b3 := uint32(data[23])
		b4 := uint32(data[24])
		bits := b1 | b2<<8 | b3<<16 | b4<<24
		w := int((bits & 0x3FFF) + 1)
		h := int(((bits >> 14) & 0x3FFF) + 1)
		return w, h, nil
	case "VP8X":
		// VP8X payload: 4 bytes Rsv/I/E/X/A/R/R, then
		// 3 bytes width-1 and 3 bytes height-1 (LE).
		// The payload starts at file offset 20; the width
		// starts at offset 24.
		if len(data) < 30 {
			return 0, 0, fmt.Errorf("webp: bad VP8X")
		}
		w := int(data[24]) | int(data[25])<<8 | int(data[26])<<16
		h := int(data[27]) | int(data[28])<<8 | int(data[29])<<16
		return w + 1, h + 1, nil
	}
	return 0, 0, fmt.Errorf("webp: unknown chunk %q", chunk)
}

// detectMIME returns the image MIME type from the file
// extension, then validates the magic bytes.
func detectMIME(path string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	var mime string
	switch ext {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	default:
		return "", &UnsupportedFormatError{Path: path, Format: ext}
	}
	if !matchesMagic(mime, data) {
		return "", &UnsupportedFormatError{Path: path, Format: ext}
	}
	return mime, nil
}

// matchesMagic checks the file's leading bytes against the
// expected magic for the MIME type.
func matchesMagic(mime string, data []byte) bool {
	if len(data) < 8 {
		return false
	}
	switch mime {
	case "image/png":
		return data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G'
	case "image/jpeg":
		return data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
	case "image/gif":
		return string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"
	case "image/webp":
		return string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	}
	return false
}
