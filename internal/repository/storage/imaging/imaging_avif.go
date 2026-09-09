package imaging

// This file registers the AVIF decoder via a blank import. The
// gen2brain/avif package registers AVIF decoding via image.RegisterFormat
// in its init() so image.Decode will recognize AVIF automatically.

import (
	_ "github.com/gen2brain/avif"
)
