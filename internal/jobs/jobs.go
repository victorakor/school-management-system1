// Package jobs defines asynq task types and their payload structures.
package jobs

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// ─── Task Type Constants ───────────────────────────────────────────────────────

const (
	TypePDFGenerateReportCard        = "pdf:report_card"
	TypePDFGenerateAppointmentLetter = "pdf:appointment_letter"
	TypePDFGenerateFeeReceipt        = "pdf:fee_receipt"
	TypePDFGenerateFinanceReport     = "pdf:finance_report"
	TypeNotificationDispatch         = "notifications:dispatch"
	TypeQuizTriggerStart             = "quiz:trigger_start"
	TypeQuizTriggerEnd               = "quiz:trigger_end"
	TypeQuizAutoSubmit               = "quiz:auto_submit"
	TypeEmailSendOTP                 = "email:send_otp"
)

// ─── Queue Names ───────────────────────────────────────────────────────────────

const (
	QueuePDFGeneration        = "pdf:generation"
	QueueNotificationsDispatch = "notifications:dispatch"
	QueueQuizTrigger          = "quiz:trigger"
	QueueEmailSend            = "email:send"
)

// ─── Task Payloads ─────────────────────────────────────────────────────────────

// PDFReportCardPayload is the payload for generating a report card PDF.
type PDFReportCardPayload struct {
	ResultID string `json:"result_id"`
	SchoolID string `json:"school_id"`
}

// PDFAppointmentLetterPayload is the payload for generating an appointment letter PDF.
type PDFAppointmentLetterPayload struct {
	ApplicationID string `json:"application_id"`
	SchoolID      string `json:"school_id"`
}

// PDFFeeReceiptPayload is the payload for generating a fee receipt PDF.
type PDFFeeReceiptPayload struct {
	PaymentID string `json:"payment_id"`
	SchoolID  string `json:"school_id"`
}

// PDFFinanceReportPayload is the payload for generating a finance report.
type PDFFinanceReportPayload struct {
	SchoolID  string `json:"school_id"`
	BursarID  string `json:"bursar_id"`
	ReportType string `json:"report_type"` // "daily", "term", "session", "debtor"
	Format    string `json:"format"`      // "pdf" or "excel"
	Params    map[string]string `json:"params"`
}

// NotificationDispatchPayload is the payload for dispatching a notification.
type NotificationDispatchPayload struct {
	UserID   string `json:"user_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Type     string `json:"type"`
	EntityID string `json:"entity_id"`
}

// QuizTriggerPayload is the payload for quiz start/end triggers.
type QuizTriggerPayload struct {
	QuizID string `json:"quiz_id"`
}

// QuizAutoSubmitPayload is the payload for auto-submitting a quiz attempt.
type QuizAutoSubmitPayload struct {
	AttemptID string `json:"attempt_id"`
	QuizID    string `json:"quiz_id"`
}

// EmailOTPPayload is the payload for sending an OTP email.
type EmailOTPPayload struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	OTPCode   string `json:"otp_code"`
}

// ─── Task Constructors ─────────────────────────────────────────────────────────

// NewPDFReportCardTask creates a new report card PDF generation task.
func NewPDFReportCardTask(resultID, schoolID string) (*asynq.Task, error) {
	payload, err := json.Marshal(PDFReportCardPayload{ResultID: resultID, SchoolID: schoolID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePDFGenerateReportCard, payload,
		asynq.Queue(QueuePDFGeneration),
		asynq.MaxRetry(3),
	), nil
}

// NewPDFAppointmentLetterTask creates a new appointment letter PDF generation task.
func NewPDFAppointmentLetterTask(applicationID, schoolID string) (*asynq.Task, error) {
	payload, err := json.Marshal(PDFAppointmentLetterPayload{ApplicationID: applicationID, SchoolID: schoolID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePDFGenerateAppointmentLetter, payload,
		asynq.Queue(QueuePDFGeneration),
		asynq.MaxRetry(3),
	), nil
}

// NewPDFFeeReceiptTask creates a new fee receipt PDF generation task.
func NewPDFFeeReceiptTask(paymentID, schoolID string) (*asynq.Task, error) {
	payload, err := json.Marshal(PDFFeeReceiptPayload{PaymentID: paymentID, SchoolID: schoolID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePDFGenerateFeeReceipt, payload,
		asynq.Queue(QueuePDFGeneration),
		asynq.MaxRetry(3),
	), nil
}

// NewEmailOTPTask creates a new OTP email task.
func NewEmailOTPTask(userID, email, fullName, otpCode string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailOTPPayload{
		UserID: userID, Email: email, FullName: fullName, OTPCode: otpCode,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailSendOTP, payload,
		asynq.Queue(QueueEmailSend),
		asynq.MaxRetry(3),
	), nil
}

// NewQuizTriggerStartTask creates a quiz start trigger task.
func NewQuizTriggerStartTask(quizID string) (*asynq.Task, error) {
	payload, err := json.Marshal(QuizTriggerPayload{QuizID: quizID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeQuizTriggerStart, payload,
		asynq.Queue(QueueQuizTrigger),
		asynq.MaxRetry(2),
	), nil
}

// NewQuizAutoSubmitTask creates a quiz auto-submit task.
func NewQuizAutoSubmitTask(attemptID, quizID string) (*asynq.Task, error) {
	payload, err := json.Marshal(QuizAutoSubmitPayload{AttemptID: attemptID, QuizID: quizID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeQuizAutoSubmit, payload,
		asynq.Queue(QueueQuizTrigger),
		asynq.MaxRetry(1),
	), nil
}
