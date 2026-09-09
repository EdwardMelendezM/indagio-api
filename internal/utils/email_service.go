package utils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/resend/resend-go/v3"

	"indagio-api/internal/config"
	"indagio-api/internal/domain"
)

type emailService struct {
	client      *resend.Client
	fromAddress string
	fromName    string
}

// NewEmailService creates a Resend-backed email service.
// Receives only what it needs from config — no global state.
func NewEmailService(cfg config.EmailConfig) (domain.EmailService, error) {
	if cfg.ResendAPIKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}
	if cfg.FromAddress == "" {
		return nil, fmt.Errorf("EMAIL_FROM_ADDRESS is required")
	}

	return &emailService{
		client:      resend.NewClient(cfg.ResendAPIKey),
		fromAddress: cfg.FromAddress,
		fromName:    cfg.FromName,
	}, nil
}

// SendOTP sends a 6-digit verification code to the given email address.
// Implements domain.EmailService.
func (s *emailService) SendOTP(ctx context.Context, email, code string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty: %w", domain.ErrValidation)
	}
	if code == "" {
		return fmt.Errorf("code cannot be empty: %w", domain.ErrValidation)
	}

	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{email},
		Subject: "Tu código de verificación - Hilos",
		Html:    buildOTPEmailBody(code),
	}

	// Run the blocking Resend call in a goroutine so we can respect ctx cancellation.
	type result struct {
		id  string
		err error
	}

	ch := make(chan result, 1)
	go func() {
		sent, err := s.client.Emails.Send(params)
		if err != nil {
			ch <- result{err: fmt.Errorf("resend: %w", err)}
			return
		}
		ch <- result{id: sent.Id}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("send otp cancelled: %w", ctx.Err())
	case r := <-ch:
		return r.err
	}
}

// SendPasswordReset sends a password reset code to the given email address.
// Implements domain.EmailService.
func (s *emailService) SendPasswordReset(ctx context.Context, email, code string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty: %w", domain.ErrValidation)
	}
	if code == "" {
		return fmt.Errorf("code cannot be empty: %w", domain.ErrValidation)
	}

	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{email},
		Subject: "Restablece su contraseña - Hilos",
		Html:    buildPasswordResetEmailBody(code),
	}

	type result struct {
		id  string
		err error
	}

	ch := make(chan result, 1)
	go func() {
		sent, err := s.client.Emails.Send(params)
		if err != nil {
			ch <- result{err: fmt.Errorf("resend: %w", err)}
			return
		}
		ch <- result{id: sent.Id}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("send password reset cancelled: %w", ctx.Err())
	case r := <-ch:
		return r.err
	}
}

// ─────────────────────────────────────────────────────────────────
// OTP generation
// ─────────────────────────────────────────────────────────────────

// GenerateOTP returns a random zero-padded 6-digit string (e.g. "047291").
func GenerateOTP() string {
	src := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(src)
	return fmt.Sprintf("%06d", rng.Intn(1_000_000))
}

// ─────────────────────────────────────────────────────────────────
// Email template
// ─────────────────────────────────────────────────────────────────

func buildOTPEmailBody(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #000; background: #fff; line-height: 1.6; margin: 0; padding: 0; }
    .container { max-width: 560px; margin: 40px auto; padding: 0 20px; }
    .code-box { font-size: 36px; font-weight: bold; letter-spacing: 10px; text-align: center; margin: 32px 0; padding: 20px; border: 1px solid #000; font-family: 'Courier New', monospace; }
    .footer { font-size: 11px; color: #666; margin-top: 40px; border-top: 1px solid #ccc; padding-top: 12px; }
  </style>
</head>
<body>
  <div class="container">
    <h2>Hilos</h2>
    <p>Hola, tu código de verificación es:</p>
    <div class="code-box">%s</div>
    <p>Expira en 15 minutos. Si no solicitaste este código, ignora este mensaje.</p>
    <div class="footer">
      <p>&copy; 2024-2026 Hilos &mdash;</p>
    </div>
  </div>
</body>
</html>`, code)
}

func buildPasswordResetEmailBody(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #000; background: #fff; line-height: 1.6; margin: 0; padding: 0; }
    .container { max-width: 560px; margin: 40px auto; padding: 0 20px; }
    .code-box { font-size: 36px; font-weight: bold; letter-spacing: 10px; text-align: center; margin: 32px 0; padding: 20px; border: 1px solid #000; font-family: 'Courier New', monospace; }
    .footer { font-size: 11px; color: #666; margin-top: 40px; border-top: 1px solid #ccc; padding-top: 12px; }
  </style>
</head>
<body>
  <div class="container">
    <h2>Hilos</h2>
    <p>Hola, hemos recibido una solicitud para restablecer tu contraseña.</p>
    <p>Tu código de recuperación es:</p>
    <div class="code-box">%s</div>
    <p>Expira en 15 minutos. Si no solicitaste este código, ignora este mensaje y tu contraseña seguirá siendo la misma.</p>
    <div class="footer">
      <p>&copy; 2024-2026 Hilos &mdash;</p>
    </div>
  </div>
</body>
</html>`, code)
}
