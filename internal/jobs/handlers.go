package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"school-platform/internal/models"
	"school-platform/internal/pdf"
	"school-platform/internal/services"
)

// WorkerDeps holds dependencies injected into job handlers.
type WorkerDeps struct {
	DB        *gorm.DB
	Email     *services.EmailService
	PDF       *pdf.Generator
	BaseURL   string
}

// RegisterHandlers wires every job type the server schedules to a handler.
func RegisterHandlers(mux *asynq.ServeMux, deps *WorkerDeps) {
	mux.HandleFunc(TypeEmailSendOTP, HandleEmailOTP(deps))
	mux.HandleFunc(TypeNotificationDispatch, HandleNotificationDispatch(deps))
	mux.HandleFunc(TypePDFGenerateReportCard, HandleGenerateReportCard(deps))
	mux.HandleFunc(TypePDFGenerateAppointmentLetter, HandleGenerateAppointmentLetter(deps))
	mux.HandleFunc(TypePDFGenerateFeeReceipt, HandleGenerateFeeReceipt(deps))
	mux.HandleFunc(TypePDFGenerateFinanceReport, HandleGenerateFinanceReport(deps))
	mux.HandleFunc(TypeQuizTriggerStart, HandleQuizTriggerStart(deps))
	mux.HandleFunc(TypeQuizTriggerEnd, HandleQuizTriggerEnd(deps))
	mux.HandleFunc(TypeQuizAutoSubmit, HandleQuizAutoSubmit(deps))
}

// ─── Email OTP ─────────────────────────────────────────────────────────────────

func HandleEmailOTP(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p EmailOTPPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid OTP payload: %w", err)
		}
		log.Info().Str("email", p.Email).Str("user_id", p.UserID).Msg("jobs: sending OTP email")

		if deps == nil || deps.Email == nil {
			log.Warn().Msg("jobs: email service not configured — OTP logged only")
			log.Debug().Str("otp", p.OTPCode).Str("email", p.Email).Msg("jobs: dev OTP code")
			return nil
		}
		return deps.Email.SendOTP(p.Email, p.FullName, p.OTPCode)
	}
}

// ─── Notification dispatch ─────────────────────────────────────────────────────

func HandleNotificationDispatch(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p NotificationDispatchPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid notification payload: %w", err)
		}
		if deps == nil || deps.DB == nil {
			return fmt.Errorf("jobs: DB not wired into worker")
		}
		userID, err := uuid.Parse(p.UserID)
		if err != nil {
			return fmt.Errorf("jobs: invalid user_id in payload: %w", err)
		}
		ns := services.NewNotificationService(deps.DB)
		return ns.Create(userID, p.Title, p.Body, p.Type, p.EntityID)
	}
}

// ─── PDF Generation ────────────────────────────────────────────────────────────

type pdfReportCardData struct {
	Result   *models.Result
	Subjects []models.ResultSubject
	Student  *models.Student
	Class    *models.Class
	School   *models.School
	Session  *models.AcademicSession
	Term     *models.Term
	DocRef   string
	QRURL    string
}

func HandleGenerateReportCard(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p PDFReportCardPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid report card payload: %w", err)
		}
		log.Info().Str("result_id", p.ResultID).Msg("jobs: generating report card PDF")

		if deps == nil || deps.PDF == nil || deps.DB == nil {
			return fmt.Errorf("jobs: PDF generator / DB not wired")
		}

		var result models.Result
		if err := deps.DB.Preload("Student").First(&result, "id = ?", p.ResultID).Error; err != nil {
			return fmt.Errorf("jobs: result not found: %w", err)
		}

		var subjects []models.ResultSubject
		deps.DB.Preload("Subject").Where("result_id = ?", p.ResultID).Find(&subjects)

		var student models.Student
		deps.DB.First(&student, "id = ?", result.StudentID)

		var class models.Class
		deps.DB.First(&class, "id = ?", result.ClassID)

		var session models.AcademicSession
		deps.DB.First(&session, "id = ?", result.SessionID)

		var term models.Term
		deps.DB.First(&term, "id = ?", result.TermID)

		var school models.School
		deps.DB.First(&school, "id = ?", p.SchoolID)

		docRef := result.DocRef
		if docRef == "" {
			docRef = "RES-" + result.ID.String()[:8]
			result.DocRef = docRef
			deps.DB.Model(&result).Update("doc_ref", docRef)
		}

		verifyURL := deps.BaseURL + "/verify?ref=" + docRef
		qrURL, _ := services.GenerateQRCodeBase64(verifyURL)

		data := pdfReportCardData{
			Result:   &result,
			Subjects: subjects,
			Student:  &student,
			Class:    &class,
			School:   &school,
			Session:  &session,
			Term:     &term,
			DocRef:   docRef,
			QRURL:    qrURL,
		}

		pdfBytes, err := deps.PDF.GenerateFromTemplate("report-card.html", data)
		if err != nil {
			return fmt.Errorf("jobs: report card PDF generation failed: %w", err)
		}

		// PDF bytes are uploaded to Cloudinary via the documents handler when
		// the Cloudinary SDK is configured. For now we store the reference so
		// downstream consumers know the file is ready.
		log.Info().
			Str("result_id", result.ID.String()).
			Str("doc_ref", docRef).
			Int("size_bytes", len(pdfBytes)).
			Msg("jobs: report card PDF generated (awaiting Cloudinary upload)")

		now := nowPtr()
		deps.DB.Model(&result).Updates(map[string]interface{}{
			"generated_at": now,
			"qr_url":       verifyURL,
		})
		return nil
	}
}

func HandleGenerateAppointmentLetter(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p PDFAppointmentLetterPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid appointment letter payload: %w", err)
		}
		log.Info().Str("application_id", p.ApplicationID).Msg("jobs: generating appointment letter PDF")

		if deps == nil || deps.PDF == nil || deps.DB == nil {
			return fmt.Errorf("jobs: PDF generator / DB not wired")
		}

		var app models.Application
		if err := deps.DB.Preload("Parent").First(&app, "id = ?", p.ApplicationID).Error; err != nil {
			return fmt.Errorf("jobs: application not found: %w", err)
		}
		var school models.School
		deps.DB.First(&school, "id = ?", p.SchoolID)

		docRef := "APP-" + app.RefNumber
		verifyURL := deps.BaseURL + "/verify?ref=" + docRef
		qrURL, _ := services.GenerateQRCodeBase64(verifyURL)

		type appointmentData struct {
			App      *models.Application
			School   *models.School
			Parent   *models.User
			DocRef   string
			QRURL    string
			VerifyURL string
		}

		data := appointmentData{
			App:       &app,
			School:    &school,
			Parent:    &app.Parent,
			DocRef:    docRef,
			QRURL:     qrURL,
			VerifyURL: verifyURL,
		}

		pdfBytes, err := deps.PDF.GenerateFromTemplate("appointment-letter.html", data)
		if err != nil {
			return fmt.Errorf("jobs: appointment letter PDF generation failed: %w", err)
		}

		letter := &models.AppointmentLetter{
			ApplicationID: app.ID,
			DocRef:        docRef,
			QRURL:         verifyURL,
			GeneratedAt:   nowPtr(),
		}
		deps.DB.Create(letter)

		// Record in DocumentVerification table so /verify resolves.
		deps.DB.Create(&models.DocumentVerification{
			DocRef:      docRef,
			DocType:     models.DocTypeAppointmentLetter,
			EntityID:    app.ID.String(),
			GeneratedAt: letter.GeneratedAt,
			IssuedBy:    "ADMISSIONS",
			StudentName: app.ChildName,
		})

		log.Info().
			Str("application_id", app.ID.String()).
			Str("doc_ref", docRef).
			Int("size_bytes", len(pdfBytes)).
			Msg("jobs: appointment letter PDF generated")
		return nil
	}
}

func HandleGenerateFeeReceipt(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p PDFFeeReceiptPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid fee receipt payload: %w", err)
		}
		log.Info().Str("payment_id", p.PaymentID).Msg("jobs: generating fee receipt PDF")

		if deps == nil || deps.PDF == nil || deps.DB == nil {
			return fmt.Errorf("jobs: PDF generator / DB not wired")
		}

		var payment models.FeePayment
		if err := deps.DB.Preload("Student").First(&payment, "id = ?", p.PaymentID).Error; err != nil {
			return fmt.Errorf("jobs: payment not found: %w", err)
		}
		var school models.School
		deps.DB.First(&school, "id = ?", p.SchoolID)

		verifyURL := deps.BaseURL + "/verify?ref=" + payment.DocRef
		qrURL, _ := services.GenerateQRCodeBase64(verifyURL)

		type receiptData struct {
			Payment  *models.FeePayment
			Student  *models.Student
			School   *models.School
			QRURL    string
			VerifyURL string
		}

		data := receiptData{
			Payment:   &payment,
			Student:   &payment.Student,
			School:    &school,
			QRURL:     qrURL,
			VerifyURL: verifyURL,
		}

		pdfBytes, err := deps.PDF.GenerateFromTemplate("fee-receipt.html", data)
		if err != nil {
			return fmt.Errorf("jobs: fee receipt PDF generation failed: %w", err)
		}

		log.Info().
			Str("payment_id", payment.ID.String()).
			Str("doc_ref", payment.DocRef).
			Int("size_bytes", len(pdfBytes)).
			Msg("jobs: fee receipt PDF generated")
		return nil
	}
}

func HandleGenerateFinanceReport(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p PDFFinanceReportPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid finance report payload: %w", err)
		}
		log.Info().
			Str("bursar_id", p.BursarID).
			Str("report_type", p.ReportType).
			Str("format", p.Format).
			Msg("jobs: generating finance report")
		if deps == nil || deps.PDF == nil || deps.DB == nil {
			return fmt.Errorf("jobs: PDF generator / DB not wired")
		}
		// Generation of finance reports uses excelize for spreadsheets and
		// chromedp for PDFs. The schema of the report is driven by p.ReportType.
		return fmt.Errorf("jobs: finance report generation is not yet implemented (type=%s format=%s)", p.ReportType, p.Format)
	}
}

// ─── Quiz triggers ─────────────────────────────────────────────────────────────

func HandleQuizTriggerStart(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p QuizTriggerPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid quiz trigger payload: %w", err)
		}
		if deps == nil || deps.DB == nil {
			return fmt.Errorf("jobs: DB not wired")
		}
		res := deps.DB.Model(&models.Quiz{}).
			Where("id = ? AND status = ?", p.QuizID, models.QuizStatusDraft).
			Update("status", models.QuizStatusActive)
		if res.Error != nil {
			return fmt.Errorf("jobs: failed to activate quiz: %w", res.Error)
		}
		if res.RowsAffected > 0 {
			log.Info().Str("quiz_id", p.QuizID).Msg("jobs: quiz activated")
		}
		return nil
	}
}

func HandleQuizTriggerEnd(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p QuizTriggerPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid quiz end payload: %w", err)
		}
		if deps == nil || deps.DB == nil {
			return fmt.Errorf("jobs: DB not wired")
		}
		res := deps.DB.Model(&models.Quiz{}).
			Where("id = ? AND status = ?", p.QuizID, models.QuizStatusActive).
			Update("status", models.QuizStatusClosed)
		if res.Error != nil {
			return fmt.Errorf("jobs: failed to close quiz: %w", res.Error)
		}
		if res.RowsAffected > 0 {
			log.Info().Str("quiz_id", p.QuizID).Msg("jobs: quiz closed")
		}
		return nil
	}
}

func HandleQuizAutoSubmit(deps *WorkerDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p QuizAutoSubmitPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("jobs: invalid auto-submit payload: %w", err)
		}
		if deps == nil || deps.DB == nil {
			return fmt.Errorf("jobs: DB not wired")
		}
		var attempt models.QuizAttempt
		if err := deps.DB.First(&attempt, "id = ?", p.AttemptID).Error; err != nil {
			return nil // attempt may have been deleted; nothing to do
		}
		if attempt.SubmittedAt != nil {
			return nil // already submitted
		}
		now := nowPtr()
		deps.DB.Model(&attempt).Updates(map[string]interface{}{
			"submitted_at":   now,
			"auto_submitted": true,
			"is_flagged":     true,
		})
		log.Info().Str("attempt_id", p.AttemptID).Msg("jobs: attempt auto-submitted")
		return nil
	}
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// nowPtr returns the current UTC time for use as GORM update values.
func nowPtr() time.Time { return time.Now().UTC() }

// keep import strings referenced in tooling paths
var _ = strings.HasPrefix
