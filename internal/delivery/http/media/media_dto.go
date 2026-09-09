package media

type PresignUploadRequest struct {
	TargetType  string `json:"target_type" binding:"required,oneof=avatar thread comment chat"`
	ContentType string `json:"content_type" binding:"required,oneof=image/jpeg image/png image/webp video/mp4"`
	Extension   string `json:"extension" binding:"required"` // ej: "webp", "mp4"
}

type PresignUploadResponse struct {
	UploadURL string `json:"upload_url"` // URL temporal para el PUT del frontend
	FileURL   string `json:"file_url"`   // URL pública final que se guardará en la BD
}

// PresignedVideoRequest — body for POST /api/media/presigned-video.
// The client declares intent (content_type, max_size_bytes) so the server can
// validate and echo the cap back. The client is responsible for enforcing the
// size cap before upload; the server does not download to verify.
type PresignedVideoRequest struct {
	ContentType  string `json:"content_type" binding:"required,oneof=video/mp4"`
	MaxSizeBytes int64  `json:"max_size_bytes" binding:"required,min=1"`
}

type PresignedVideoResponse struct {
	UploadURL      string `json:"upload_url"`     // URL PUT firmada para R2
	RawKey         string `json:"raw_key"`        // ej: "videos/raw/uuid.mp4"
	MaxSizeBytes   int64  `json:"max_size_bytes"` // echo desde el cliente (o cap del server si el cliente pidió más)
	MaxDurationSec int    `json:"max_duration_s"` // cap de duración del server (videos más largos serán rechazados por el worker)
}

// PresignedStickerRequest — body for POST /api/media/presigned-sticker.
// The server clamps the requested size to its own cap and returns the
// caller's current quota so the FE can show "X of Y usados" in the UI.
type PresignedStickerRequest struct {
	ContentType  string `json:"content_type" binding:"required,oneof=image/jpeg image/png image/webp image/gif image/avif"`
	MaxSizeBytes int64  `json:"max_size_bytes" binding:"required,min=1"`
	Extension    string `json:"extension" binding:"required,min=2,max=8"` // ej: "png", "jpg", "webp", "gif", "avif"
}

// PresignedStickerResponse — server-issued presigned PUT URL plus the
// caller's current quota. The FE uses the quota to render an "X stickers
// restantes" hint in the upload dialog.
type PresignedStickerResponse struct {
	UploadURL    string `json:"upload_url"`     // URL PUT firmada para R2
	RawKey       string `json:"raw_key"`        // ej: "stickers/pending/<uuid>/<file>.png"
	MaxSizeBytes int64  `json:"max_size_bytes"` // server-clamped cap
	MaxStickers  int    `json:"max_stickers"`   // quota: max stickers per user
	MaxPacks     int    `json:"max_packs"`      // quota: max packs per user
	StickersUsed int    `json:"stickers_used"`  // caller's current sticker count
	PacksUsed    int    `json:"packs_used"`     // caller's current pack count
}
