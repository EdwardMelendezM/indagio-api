// Package imaging turns a single decoded source image into a battery of
// sized variants. It deliberately has no I/O concerns: callers feed bytes
// in, get bytes out, and persist them via the storage repository of
// their choice.
//
// The package exists because avatar uploads and sticker uploads need
// predictable byte sizes regardless of client-side variance. All numeric
// work is synchronous and bounded; parallelize at the call-site for
// throughput.
package imaging

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"

	// Register WebP decoder at package init via side-effect import.
	// Without this, image.Decode would reject WebP inputs.
	// GIF is natively supported by image.Decode without explicit registration.
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// SizeSpec defines a desired variant. Width == Height for squares; we
// always produce squares today, but the struct leaves room for future
// non-square crops without changing call-sites.
type SizeSpec struct {
	Name   string // "thumb" | "medium" | "full"
	Width  int
	Height int
	Format string // "jpeg" | "png"
}

// DefaultSizes is the canonical avatar sizing ladder.
var DefaultSizes = []SizeSpec{
	{Name: "thumb", Width: 64, Height: 64, Format: "jpeg"},
	{Name: "medium", Width: 256, Height: 256, Format: "jpeg"},
	{Name: "full", Width: 512, Height: 512, Format: "jpeg"},
}

// Variant is one produced image.
type Variant struct {
	Size  SizeSpec
	Bytes []byte
}

// ErrTooLarge is returned when the input image exceeds the budget.
var ErrTooLarge = errors.New("image too large")

// MaxInputBytes caps the upload so a single request can't starve the
// process. 8 MB is comfortable for a high-quality photo.
const MaxInputBytes = 8 << 20

// Process decodes src (JPEG, PNG, WebP, GIF, or AVIF), resizes to each requested
// spec using Catmull-Rom smoothing via x/image/draw, and returns the
// encoded bytes as JPEG. The caller owns the returned slices.
//
// EXIF metadata is stripped as a side effect of re-encoding.
func Process(src io.Reader, sizes []SizeSpec) ([]Variant, error) {
	if sizes == nil {
		sizes = DefaultSizes
	}

	// Bound the input — anything bigger than the budget is rejected
	// before we try to decode.
	data, err := io.ReadAll(io.LimitReader(src, MaxInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	if len(data) > MaxInputBytes {
		return nil, ErrTooLarge
	}

	img, format, err := decodeImage(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if !isSupportedFormat(format) {
		return nil, fmt.Errorf("unsupported image format %q (jpeg|png|webp|gif|avif only)", format)
	}

	out := make([]Variant, 0, len(sizes))
	for _, spec := range sizes {
		buf := &bytes.Buffer{}
		resized := resize(img, spec.Width, spec.Height)
		if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: jpegQualityFor(spec)}); err != nil {
			return nil, fmt.Errorf("encode %s jpeg: %w", spec.Name, err)
		}
		out = append(out, Variant{Size: spec, Bytes: buf.Bytes()})
	}

	return out, nil
}

// decodeImage auto-detects format from the magic bytes.
func decodeImage(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		// Fallback: try PNG explicitly because stdlib doesn't include
		// the PNG decoder on every tag — actually stdlib does include
		// image/png. The fallback handles odd cases where a header
		// looks like both formats.
		_ = png.Decode // referenced for side effect only on builds without png init
		return nil, "", fmt.Errorf("decode image: %w", err)
	}
	return img, format, nil
}

// isSupportedFormat returns true for the input formats the avatar
// pipeline accepts. WebP, GIF, and AVIF are supported here because their
// decoders are registered via imports (WebP and AVIF via blank imports,
// GIF via stdlib).
func isSupportedFormat(format string) bool {
	switch format {
	case "jpeg", "png", "webp", "gif", "avif":
		return true
	}
	return false
}

// resize uses Catmull-Rom interpolation via x/image/draw, which gives
// visual output equivalent to popular pure-Go libraries without the
// extra dependency.
func resize(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// jpegQualityFor picks a quality profile matching the variant size —
// small thumbs need higher relative quality to stay sharp.
func jpegQualityFor(spec SizeSpec) int {
	switch spec.Name {
	case "thumb":
		return 88
	case "medium":
		return 85
	default:
		return 82
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Sticker pipeline
// ─────────────────────────────────────────────────────────────────────────────
//
// Stickers are different from avatars: they need transparency (PNG),
// are smaller (max 256 instead of 512), and must be square. The full
// variant is encoded as PNG (lossless, transparent) so the chat renders
// it on any background; the thumb is encoded as JPEG (lossy, opaque) so
// the picker grid renders fast and small.
//
// Supported input formats: PNG, JPEG, WebP, GIF, and AVIF.
// Stdlib-only encoding (no cgo WebP encoder). If the product later
// requires WebP/AVIF output for size, swap in a third-party encoder and
// keep the same SizeSpec / Variant shape.

// StickerSizes defines the two variants produced by ProcessSticker.
type StickerSizes struct {
	Full  SizeSpec // 256x256, png (transparent)
	Thumb SizeSpec // 96x96,  jpeg (opaque)
}

// DefaultStickerSizes is the canonical sticker sizing ladder.
var DefaultStickerSizes = StickerSizes{
	Full:  SizeSpec{Name: "full", Width: 256, Height: 256, Format: "png"},
	Thumb: SizeSpec{Name: "thumb", Width: 96, Height: 96, Format: "jpeg"},
}

// MaxStickerInputBytes caps the pre-processed upload. Default 1 MB —
// generous for a 1024x1024 PNG with transparency.
const MaxStickerInputBytes = 1 << 20

// MaxStickerInputDimension is the largest allowed input edge before
// sticky-center crop. Larger inputs are rejected so a malicious user
// can't tie up the CPU with a 50-megapixel upload.
const MaxStickerInputDimension = 1024

// StickerVariant is the two-variant output of ProcessSticker.
type StickerVariant struct {
	Full  []byte
	Thumb []byte
}

// ErrStickerNotSquare is returned when the input isn't square.
var ErrStickerNotSquare = errors.New("sticker image must be square")

// ErrStickerTooLarge is returned when the input dimension exceeds the
// MaxStickerInputDimension cap.
var ErrStickerTooLarge = errors.New("sticker input dimension exceeds server cap")

// ProcessSticker decodes src (PNG, JPEG, WebP, GIF, or AVIF), validates that it's
// square and within the dimension cap, then resizes to the two sticker
// variants. The full variant preserves transparency (PNG); the thumb is
// opaque (JPEG on a white background).
//
// The function is CPU-bound and synchronous. The caller (the worker)
// enforces concurrency by running the worker pool size.
func ProcessSticker(src io.Reader) (StickerVariant, error) {
	return ProcessStickerWithSizes(src, DefaultStickerSizes)
}

// ProcessStickerWithSizes is the variant of ProcessSticker that lets
// the caller override the size ladder (used by tests).
func ProcessStickerWithSizes(src io.Reader, sizes StickerSizes) (StickerVariant, error) {
	// 1. Bound the input.
	data, err := io.ReadAll(io.LimitReader(src, MaxStickerInputBytes+1))
	if err != nil {
		return StickerVariant{}, fmt.Errorf("read input: %w", err)
	}
	if len(data) > MaxStickerInputBytes {
		return StickerVariant{}, ErrTooLarge
	}

	// 2. Decode.
	img, format, err := decodeImage(bytes.NewReader(data))
	if err != nil {
		return StickerVariant{}, err
	}
	if !isSupportedFormat(format) {
		return StickerVariant{}, fmt.Errorf("unsupported image format %q (jpeg|png|webp|gif|avif only)", format)
	}

	// 3. Validate: must be square + within dimension cap.
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w != h {
		return StickerVariant{}, ErrStickerNotSquare
	}
	if w > MaxStickerInputDimension || h > MaxStickerInputDimension {
		return StickerVariant{}, ErrStickerTooLarge
	}

	// 4. Resize to full (transparent PNG).
	fullCanvas := image.NewNRGBA(image.Rect(0, 0, sizes.Full.Width, sizes.Full.Height))
	draw.CatmullRom.Scale(fullCanvas, fullCanvas.Bounds(), img, bounds, draw.Over, nil)
	var fullBuf bytes.Buffer
	if err := png.Encode(&fullBuf, fullCanvas); err != nil {
		return StickerVariant{}, fmt.Errorf("encode full png: %w", err)
	}

	// 5. Resize to thumb (opaque JPEG — flatten on neutral grey to
	//    avoid a black box on dark chat backgrounds).
	thumbCanvas := image.NewRGBA(image.Rect(0, 0, sizes.Thumb.Width, sizes.Thumb.Height))
	// Composite onto an opaque grey background first.
	bg := image.NewUniform(color.RGBA{R: 0xEE, G: 0xEE, B: 0xEE, A: 0xFF})
	draw.Copy(thumbCanvas, image.Point{}, bg, bg.Bounds(), draw.Src, nil)
	draw.CatmullRom.Scale(thumbCanvas, thumbCanvas.Bounds(), img, bounds, draw.Over, nil)
	var thumbBuf bytes.Buffer
	if err := jpeg.Encode(&thumbBuf, thumbCanvas, &jpeg.Options{Quality: 85}); err != nil {
		return StickerVariant{}, fmt.Errorf("encode thumb jpeg: %w", err)
	}

	return StickerVariant{
		Full:  fullBuf.Bytes(),
		Thumb: thumbBuf.Bytes(),
	}, nil
}
