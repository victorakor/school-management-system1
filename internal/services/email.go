package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// EmailService sends transactional emails via the Resend REST API.
type EmailService struct {
	apiKey  string
	fromAddr string
}

// NewEmailService creates a new EmailService.
func NewEmailService(apiKey, fromAddr string) *EmailService {
	return &EmailService{apiKey: apiKey, fromAddr: fromAddr}
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// SendOTP sends an OTP verification email to a parent.
func (s *EmailService) SendOTP(toEmail, fullName, otpCode string) error {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Inter, sans-serif; background: #F1F5F9; padding: 40px 0;">
  <div style="max-width: 480px; margin: 0 auto; background: white; border-radius: 16px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.06);">
    <div style="background: #0F2557; padding: 32px; text-align: center;">
      <h1 style="color: white; margin: 0; font-size: 1.5rem;">Email Verification</h1>
    </div>
    <div style="padding: 32px;">
      <p style="color: #0F172A; font-size: 1rem;">Hello <strong>%s</strong>,</p>
      <p style="color: #64748B;">Please use the verification code below to complete your registration:</p>
      <div style="background: #F1F5F9; border-radius: 12px; padding: 24px; text-align: center; margin: 24px 0;">
        <span style="font-family: 'JetBrains Mono', monospace; font-size: 2.5rem; font-weight: 700; color: #0F2557; letter-spacing: 0.5rem;">%s</span>
      </div>
      <p style="color: #64748B; font-size: 0.875rem;">This code expires in <strong>15 minutes</strong>. Do not share it with anyone.</p>
      <p style="color: #64748B; font-size: 0.875rem;">If you did not request this, please ignore this email.</p>
    </div>
    <div style="background: #F8FAFC; padding: 16px 32px; text-align: center; border-top: 1px solid #E2E8F0;">
      <p style="color: #94A3B8; font-size: 0.75rem; margin: 0;">This is an automated message — please do not reply.</p>
    </div>
  </div>
</body>
</html>`, fullName, otpCode)

	return s.send(toEmail, "Your Verification Code", html)
}

// SendStudentCredentials sends login credentials to a parent after student account creation.
func (s *EmailService) SendStudentCredentials(toEmail, parentName, studentName, loginEmail, tempPassword, portalURL string) error {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Inter, sans-serif; background: #F1F5F9; padding: 40px 0;">
  <div style="max-width: 480px; margin: 0 auto; background: white; border-radius: 16px; overflow: hidden; box-shadow: 0 4px 24px rgba(0,0,0,0.06);">
    <div style="background: #0F2557; padding: 32px; text-align: center;">
      <h1 style="color: white; margin: 0; font-size: 1.5rem;">Student Account Created</h1>
    </div>
    <div style="padding: 32px;">
      <p style="color: #0F172A;">Dear <strong>%s</strong>,</p>
      <p style="color: #64748B;">A student portal account has been created for <strong>%s</strong>.</p>
      <div style="background: #F1F5F9; border-radius: 12px; padding: 20px; margin: 24px 0;">
        <p style="margin: 0 0 8px; color: #64748B; font-size: 0.875rem;">Login Email</p>
        <p style="margin: 0 0 16px; color: #0F172A; font-weight: 600;">%s</p>
        <p style="margin: 0 0 8px; color: #64748B; font-size: 0.875rem;">Temporary Password</p>
        <p style="margin: 0; color: #0F172A; font-weight: 600; font-family: monospace;">%s</p>
      </div>
      <p style="color: #64748B; font-size: 0.875rem;">Please change the password after first login.</p>
      <a href="%s" style="display: inline-block; background: #2563EB; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: 600; margin-top: 16px;">Access Portal</a>
    </div>
  </div>
</body>
</html>`, parentName, studentName, loginEmail, tempPassword, portalURL)

	return s.send(toEmail, fmt.Sprintf("Student Account Created — %s", studentName), html)
}

func (s *EmailService) send(toEmail, subject, html string) error {
	payload := resendEmailRequest{
		From:    s.fromAddr,
		To:      []string{toEmail},
		Subject: subject,
		HTML:    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("email: failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("email: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Error().Int("status", resp.StatusCode).Str("to", toEmail).Msg("email: Resend API error")
		return fmt.Errorf("email: Resend API returned status %d", resp.StatusCode)
	}

	log.Info().Str("to", toEmail).Str("subject", subject).Msg("email: sent successfully")
	return nil
}
