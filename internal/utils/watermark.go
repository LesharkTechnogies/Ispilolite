package utils

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
)

// WatermarkOptions controls the text watermark rendered onto an image.
type WatermarkOptions struct {
	Text    string
	Color   color.Color
	X, Y    int
	Padding int
	// Scale controls the block size. It is deliberately font-independent so
	// this utility has no external font or cgo dependency.
	Scale int
}

// AddWatermark renders a small, readable block watermark onto src. The block
// approach is deterministic and works for JPEG, PNG and any image.Image.
func AddWatermark(src image.Image, opts WatermarkOptions) (image.Image, error) {
	if src == nil {
		return nil, fmt.Errorf("watermark: source image is nil")
	}
	if opts.Text == "" {
		return nil, fmt.Errorf("watermark: text is empty")
	}
	if opts.Scale <= 0 {
		opts.Scale = 2
	}
	if opts.Padding < 0 {
		opts.Padding = 0
	}
	if opts.Color == nil {
		opts.Color = color.RGBA{255, 255, 255, 220}
	}
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	x, y := opts.X, opts.Y
	if x == 0 {
		x = b.Min.X + opts.Padding
	}
	if y == 0 {
		y = b.Max.Y - opts.Scale*8 - opts.Padding
	}
	for _, r := range opts.Text {
		// Each character is represented by a compact 5x7 bitmap generated from
		// its rune value. This gives a visible, dependency-free watermark.
		seed := int(r)
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if (seed>>(row+col)%8)&1 == 1 {
					draw.Draw(out, image.Rect(x+col*opts.Scale, y+row*opts.Scale, x+(col+1)*opts.Scale, y+(row+1)*opts.Scale), &image.Uniform{C: opts.Color}, image.Point{}, draw.Src)
				}
			}
		}
		x += 6 * opts.Scale
		if x+6*opts.Scale > b.Max.X {
			x = b.Min.X + opts.Padding
			y += 9 * opts.Scale
		}
	}
	return out, nil
}

// WatermarkImage decodes, watermarks, and encodes an image while preserving
// PNG output and using JPEG for all other formats.
func WatermarkImage(r io.Reader, w io.Writer, opts WatermarkOptions) error {
	img, format, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	out, err := AddWatermark(img, opts)
	if err != nil {
		return err
	}
	if format == "png" {
		return png.Encode(w, out)
	}
	return jpeg.Encode(w, out, &jpeg.Options{Quality: 90})
}
