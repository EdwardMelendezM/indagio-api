package domain

import "github.com/google/uuid"

// VideoJob es lo que viaja por el channel de Go
type VideoJob struct {
	ThreadID   uuid.UUID
	RawKey     string
	Attempts   int     // 1 en el primer enqueue; +1 en cada reintento
	TrimStartS float64 // inclusive start of trim range (0 = no trim)
	TrimEndS   float64 // exclusive end of trim range (0 = no trim)
}

// Estados del video
const (
	VideoStatusProcessing = "PROCESANDO"
	VideoStatusReady      = "LISTO"
	VideoStatusFailed     = "FALLIDO"
)
