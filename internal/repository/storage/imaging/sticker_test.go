package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mkSquarePNG builds a square PNG the given size with a solid-fill RGBA
// color. Used by the happy-path test.
func mkSquarePNG(t *testing.T, size int, fill color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestProcessSticker_HappyPath(t *testing.T) {
	raw := mkSquarePNG(t, 512, color.NRGBA{R: 200, G: 100, B: 50, A: 255})

	v, err := ProcessSticker(bytes.NewReader(raw))
	require.NoError(t, err)
	assert.NotEmpty(t, v.Full, "full variant must be non-empty")
	assert.NotEmpty(t, v.Thumb, "thumb variant must be non-empty")

	// PNG signature on the full variant.
	assert.True(t, bytes.HasPrefix(v.Full, []byte{0x89, 'P', 'N', 'G'}))
	// JPEG SOI marker on the thumb variant.
	assert.True(t, bytes.HasPrefix(v.Thumb, []byte{0xFF, 0xD8}))
}

func TestProcessSticker_RejectsNonSquare(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 256, 128)) // rectangular
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	_, err := ProcessSticker(bytes.NewReader(buf.Bytes()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStickerNotSquare)
}

func TestProcessSticker_RejectsOversizedDimension(t *testing.T) {
	raw := mkSquarePNG(t, MaxStickerInputDimension+1, color.NRGBA{R: 1, G: 2, B: 3, A: 255})

	_, err := ProcessSticker(bytes.NewReader(raw))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStickerTooLarge)
}

func TestProcessSticker_RejectsTooLargeInput(t *testing.T) {
	// Build a payload that exceeds MaxStickerInputBytes. Allocates
	// MaxStickerInputBytes+1 bytes of zeros — the io.LimitReader caps
	// the read; the decode path then rejects on length.
	raw := make([]byte, MaxStickerInputBytes+1)

	_, err := ProcessSticker(bytes.NewReader(raw))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTooLarge)
}
