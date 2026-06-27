package jobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client wraps an asynq.Client to provide a typed API for the rest of the app.
type Client struct {
	c *asynq.Client
}

// NewClient creates a Client connected to the given Redis URL.
func NewClient(redisURL string) *Client {
	return &Client{c: asynq.NewClient(asynq.RedisClientOpt{Addr: redisURL})}
}

// enqueueJSON marshals the payload and enqueues it under the given task type and queue.
func (c *Client) enqueueJSON(taskType, queue string, payload interface{}, maxRetry int, processAt time.Time) error {
	if c == nil || c.c == nil {
		return fmt.Errorf("jobs: client not initialised")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	opts := []asynq.Option{asynq.Queue(queue), asynq.MaxRetry(maxRetry)}
	if !processAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(processAt))
	}
	_, err = c.c.Enqueue(asynq.NewTask(taskType, b, opts...))
	return err
}

// EnqueueEmailOTP enqueues an OTP email task.
func (c *Client) EnqueueEmailOTP(userID, email, fullName, otpCode string) error {
	return c.enqueueJSON(TypeEmailSendOTP, QueueEmailSend,
		EmailOTPPayload{UserID: userID, Email: email, FullName: fullName, OTPCode: otpCode},
		3, time.Time{})
}

// EnqueueNotification enqueues an in-app notification dispatch.
func (c *Client) EnqueueNotification(userID, title, body, notifType, entityID string) error {
	return c.enqueueJSON(TypeNotificationDispatch, QueueNotificationsDispatch,
		NotificationDispatchPayload{UserID: userID, Title: title, Body: body, Type: notifType, EntityID: entityID},
		3, time.Time{})
}

// EnqueueAppointmentLetterPDF enqueues an appointment letter PDF task.
func (c *Client) EnqueueAppointmentLetterPDF(applicationID, schoolID string) error {
	return c.enqueueJSON(TypePDFGenerateAppointmentLetter, QueuePDFGeneration,
		PDFAppointmentLetterPayload{ApplicationID: applicationID, SchoolID: schoolID},
		3, time.Time{})
}

// EnqueueReportCardPDF enqueues a report card PDF task.
func (c *Client) EnqueueReportCardPDF(resultID, schoolID string) error {
	return c.enqueueJSON(TypePDFGenerateReportCard, QueuePDFGeneration,
		PDFReportCardPayload{ResultID: resultID, SchoolID: schoolID},
		3, time.Time{})
}

// EnqueueFeeReceiptPDF enqueues a fee receipt PDF task.
func (c *Client) EnqueueFeeReceiptPDF(paymentID, schoolID string) error {
	return c.enqueueJSON(TypePDFGenerateFeeReceipt, QueuePDFGeneration,
		PDFFeeReceiptPayload{PaymentID: paymentID, SchoolID: schoolID},
		3, time.Time{})
}

// EnqueueQuizStart schedules the quiz status transition at start time.
func (c *Client) EnqueueQuizStart(quizID string, at time.Time) error {
	return c.enqueueJSON(TypeQuizTriggerStart, QueueQuizTrigger,
		QuizTriggerPayload{QuizID: quizID}, 2, at)
}

// EnqueueQuizEnd schedules the quiz closing at end time.
func (c *Client) EnqueueQuizEnd(quizID string, at time.Time) error {
	return c.enqueueJSON(TypeQuizTriggerEnd, QueueQuizTrigger,
		QuizTriggerPayload{QuizID: quizID}, 2, at)
}

// EnqueueQuizAutoSubmit schedules the auto-submit at start + duration.
func (c *Client) EnqueueQuizAutoSubmit(attemptID, quizID string, at time.Time) error {
	return c.enqueueJSON(TypeQuizAutoSubmit, QueueQuizTrigger,
		QuizAutoSubmitPayload{AttemptID: attemptID, QuizID: quizID}, 1, at)
}

// Shutdown releases the underlying asynq client connection.
func (c *Client) Shutdown() {
	if c == nil || c.c == nil {
		return
	}
	_ = c.c.Close()
}