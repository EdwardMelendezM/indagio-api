package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Constantes para job types — evitar strings hardcodeados en los usecases
const (
	JobTypeSendOTP                     = "send_otp"
	JobTypeSendPushReaction            = "send_push_reaction"
	JobTypeSendPushComment             = "send_push_comment"
	JobTypeSendPushNewThread           = "send_push_new_thread"
	JobTypeSendPushChatMessage         = "send_push_chat_message"
	JobTypeSendPushScholarshipDeadline = "send_push_scholarship_deadline"
	JobTypeSendPasswordReset           = "send_password_reset"
)

type Job struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	RunAt       time.Time       `json:"run_at"`
	ErrorLog    string          `json:"error_log,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type JobRepository interface {
	Enqueue(ctx context.Context, jobType string, payload any) error
	Dequeue(ctx context.Context, jobTypes []string) (*Job, error)
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	// scheduleRetry=true: calcula backoff y reprograma run_at si attempts < max_attempts
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, scheduleRetry bool) error
	// Limpieza periódica de jobs completados/fallidos con más de N días
	PurgeOld(ctx context.Context, olderThanDays int) (int64, error)
}

// OTPJobPayload es el payload para jobs de envío de OTP por email.
type OTPJobPayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// PushReactionPayload es el payload para jobs de notificación push de reacción.
type PushReactionPayload struct {
	ReceiverID uuid.UUID `json:"receiver_id"`
	ActorName  string    `json:"actor_name"`
	TargetType string    `json:"target_type"`
	TargetID   uuid.UUID `json:"target_id"`
}

// PushCommentPayload es el payload para jobs de notificación push de comentario.
type PushCommentPayload struct {
	ReceiverID  uuid.UUID `json:"receiver_id"`
	ActorName   string    `json:"actor_name"`
	ThreadID    uuid.UUID `json:"thread_id"`
	ThreadTitle string    `json:"thread_title"`
}

// PasswordResetJobPayload es el payload para jobs de envío de email de reset de contraseña.
type PasswordResetJobPayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// PushChatMessagePayload es el payload para jobs de notificación push de
// mensajes de chat. Se genera cuando un participante envía un mensaje en una
// conversación y queremos notificar al destinatario que está offline.
//
// El campo DebouncedCount indica cuántos mensajes se agruparon en este push
// (1 para un envío único, ≥2 cuando el buffer de debouncing agrupó varios
// envíos consecutivos del mismo emisor al mismo destinatario/conversación
// dentro de la ventana de 3 segundos). El snippet, hasMedia y mediaCount
// corresponden al mensaje más reciente del grupo.
type PushChatMessagePayload struct {
	ReceiverID     uuid.UUID `json:"receiver_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	SenderName     string    `json:"sender_name"`
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
	Snippet        string    `json:"snippet"`
	HasMedia       bool      `json:"has_media"`
	MediaCount     int       `json:"media_count"`
	DebouncedCount int       `json:"debounced_count"`
}

// PushScholarshipDeadlinePayload — fired by the daily reminder worker for
// every bookmarked scholarship whose apply_deadline falls on the configured
// day-mark (7 / 3 / 1 days before). DaysBefore is included so the recipient
// sees something like "Cierra en 3 días" rather than a generic alert.
type PushScholarshipDeadlinePayload struct {
	ReceiverID       uuid.UUID `json:"receiver_id"`
	ScholarshipID    uuid.UUID `json:"scholarship_id"`
	ScholarshipTitle string    `json:"scholarship_title"`
	DaysBefore       int       `json:"days_before"`
}
