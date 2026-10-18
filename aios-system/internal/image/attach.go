package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	xdraw "golang.org/x/image/draw"
)

// ImageAttachment wraps an Image for use with a model
// (REQ-IMG-ATT-001 to 004). The attachment supports the
// Anthropic image block, the OpenAI image URL, and resize
// to a maximum dimension.
type ImageAttachment struct {
	Image Image
}

// NewImageAttachment wraps an Image.
func NewImageAttachment(img Image) *ImageAttachment {
	return &ImageAttachment{Image: img}
}

// ToAnthropicBlock returns the Anthropic image block
// (REQ-IMG-ATT-002). The block is a map the caller can
// pass as a content block to the Anthropic messages API.
//
// Shape:
//
//	{
//	  "type": "image",
//	  "source": {
//	    "type": "base64",
//	    "media_type": "<mime>",
//	    "data": "<base64>"
//	  }
//	}
func (a *ImageAttachment) ToAnthropicBlock() map[string]any {
	return map[string]any{
		"type": "image",
		"source": map[string]any{
			"type":       "base64",
			"media_type": a.Image.MIMEType,
			"data":       encodeBase64(a.Image.Data),
		},
	}
}

// ToOpenAIBlock returns the OpenAI image URL block
// (REQ-IMG-ATT-003). The block is a map the caller can
// pass as a content part to the OpenAI chat-completions API.
//
// Shape:
//
//	{
//	  "type": "image_url",
//	  "image_url": { "url": "data:<mime>;base64,<data>" }
//	}
func (a *ImageAttachment) ToOpenAIBlock() map[string]any {
	url := fmt.Sprintf("data:%s;base64,%s", a.Image.MIMEType, encodeBase64(a.Image.Data))
	return map[string]any{
		"type": "image_url",
		"image_url": map[string]any{
			"url": url,
		},
	}
}

// Resize returns a new ImageAttachment with the image
// resized so that the longest side is at most maxDim
// (REQ-IMG-ATT-004). The aspect ratio is preserved. The
// resampler is the CatmullRom kernel from x/image/draw,
// which is a high-quality cubic resampler. When the
// image is already smaller than maxDim on both sides,
// the image is returned unchanged.
func (a *ImageAttachment) Resize(maxDim int) (*ImageAttachment, error) {
	if maxDim <= 0 {
		return nil, fmt.Errorf("image: resize: maxDim must be > 0")
	}
	if a.Image.Width <= maxDim && a.Image.Height <= maxDim {
		return a, nil
	}
	src, _, err := image.Decode(bytes.NewReader(a.Image.Data))
	if err != nil {
		return nil, &UnsupportedFormatError{Path: a.Image.Path, Format: a.Image.MIMEType}
	}
	newW, newH := fitInside(a.Image.Width, a.Image.Height, maxDim)
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	encoded, err := encodeImage(dst, a.Image.MIMEType)
	if err != nil {
		return nil, err
	}
	resized := a.Image
	resized.Data = encoded
	resized.Width = newW
	resized.Height = newH
	return &ImageAttachment{Image: resized}, nil
}

// fitInside returns the new (w, h) so that the longer
// side is at most maxDim and the aspect ratio is preserved.
func fitInside(w, h, maxDim int) (int, int) {
	if w >= h {
		nw := maxDim
		nh := (h * maxDim) / w
		if nh < 1 {
			nh = 1
		}
		return nw, nh
	}
	nh := maxDim
	nw := (w * maxDim) / h
	if nw < 1 {
		nw = 1
	}
	return nw, nh
}

// encodeImage encodes the image in the given MIME type.
// The MIME type is one of the supported read formats.
func encodeImage(img image.Image, mime string) ([]byte, error) {
	var buf bytes.Buffer
	switch mime {
	case "image/png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case "image/jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, err
		}
	default:
		// GIF and WebP are not re-encoded here. Resize
		// to PNG for portability.
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// encodeBase64 returns the base64 encoding of data. The
// encoding is the standard unpadded form.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
