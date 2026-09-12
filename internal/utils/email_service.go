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

	return s.send(ctx, email, "Tu código de verificación - Indagio", buildOTPEmailBody(code))
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

	return s.send(ctx, email, "Restablece tu contraseña - Indagio", buildPasswordResetEmailBody(code))
}

// send centraliza el envío real y el respeto a ctx.Done(), evitando repetir
// el patrón goroutine+select en cada método público del servicio.
func (s *emailService) send(ctx context.Context, to, subject, html string) error {
	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
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
		return fmt.Errorf("send email cancelled: %w", ctx.Err())
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
// Email templates
// ─────────────────────────────────────────────────────────────────

const (
	brandPrimary     = "#4338ca" // indigo-700
	brandPrimarySoft = "#eef2ff" // indigo-50
	brandText        = "#18181b"
	brandMuted       = "#71717a"
	brandBorder      = "#e4e4e7"
)

func buildOTPEmailBody(code string) string {
	body := fmt.Sprintf(`
		<p style="margin:0 0 16px;font-size:15px;color:%s;">Hola,</p>
		<p style="margin:0 0 24px;font-size:15px;color:%s;line-height:1.6;">
			Usa el siguiente código para verificar tu cuenta en <strong>Indagio</strong>:
		</p>
		%s
		<p style="margin:24px 0 0;font-size:13px;color:%s;line-height:1.6;">
			El código expira en 15 minutos. Si tú no solicitaste esto, puedes ignorar este correo con tranquilidad.
		</p>
	`, brandText, brandText, codeBox(code), brandMuted)

	return renderEmailLayout("Tu código de verificación", body)
}

func buildPasswordResetEmailBody(code string) string {
	body := fmt.Sprintf(`
		<p style="margin:0 0 16px;font-size:15px;color:%s;">Hola,</p>
		<p style="margin:0 0 24px;font-size:15px;color:%s;line-height:1.6;">
			Recibimos una solicitud para restablecer tu contraseña de <strong>Indagio</strong>. Usa este código para continuar:
		</p>
		%s
		<p style="margin:24px 0 0;font-size:13px;color:%s;line-height:1.6;">
			El código expira en 15 minutos. Si no solicitaste este cambio, tu contraseña seguirá siendo la misma — puedes ignorar este correo.
		</p>
	`, brandText, brandText, codeBox(code), brandMuted)

	return renderEmailLayout("Restablece tu contraseña", body)
}

// codeBox renderiza el bloque destacado con el código de un solo uso.
func codeBox(code string) string {
	return fmt.Sprintf(`
		<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin:8px 0;">
			<tr>
				<td align="center" style="background:%s;border:1px solid %s;border-radius:12px;padding:20px;">
					<span style="font-family:'Courier New',monospace;font-size:32px;font-weight:700;letter-spacing:10px;color:%s;">%s</span>
				</td>
			</tr>
		</table>
	`, brandPrimarySoft, brandBorder, brandPrimary, code)
}

// renderEmailLayout envuelve el contenido de cada correo con el mismo header
// (marca Indagio) y footer, para no repetir el esqueleto HTML en cada
// plantilla.
func renderEmailLayout(title, bodyHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="480" cellpadding="0" cellspacing="0" style="max-width:480px;width:100%%;background:#ffffff;border-radius:16px;border:1px solid %s;">
          <tr>
            <td style="padding:32px 32px 24px;">
              <table role="presentation" cellpadding="0" cellspacing="0">
                <tr>
                  <td style="width:40px;height:40px;background:%s;border-radius:9999px;text-align:center;vertical-align:middle;">
                    <span style="color:#ffffff;font-size:14px;font-weight:700;line-height:40px;">in</span>
                  </td>
                  <td style="padding-left:12px;">
                    <div style="font-size:15px;font-weight:600;color:%s;">Indagio</div>
                    <div style="font-size:12px;color:%s;">Workspace de investigación</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:0 32px 8px;">
              <h1 style="margin:0 0 16px;font-size:20px;font-weight:600;color:%s;">%s</h1>
              %s
            </td>
          </tr>
          <tr>
            <td style="padding:24px 32px 32px;border-top:1px solid %s;margin-top:24px;">
              <p style="margin:16px 0 0;font-size:11px;color:%s;">&copy; 2024-2026 Indagio</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, title, brandBorder, brandPrimary, brandText, brandMuted, brandText, title, bodyHTML, brandBorder, brandMuted)
}
