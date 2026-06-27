package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/models"
	"school-platform/internal/utils"
)

// AdmissionService handles all admissions business logic.
type AdmissionService struct {
	db    *gorm.DB
	notif *NotificationService
}

// NewAdmissionService creates a new AdmissionService.
func NewAdmissionService(db *gorm.DB, notif *NotificationService) *AdmissionService {
	return &AdmissionService{db: db, notif: notif}
}

// CheckDuplicate checks if a child with the same name and DOB already exists.
// Returns an error if a duplicate is found.
func (s *AdmissionService) CheckDuplicate(schoolID uuid.UUID, childName string, childDOB time.Time) error {
	// Check existing applications
	var appCount int64
	s.db.Model(&models.Application{}).
		Where("school_id = ? AND LOWER(child_name) = LOWER(?) AND child_dob = ? AND is_archived = false",
			schoolID, childName, childDOB).
		Count(&appCount)
	if appCount > 0 {
		return errors.New("an application for this child already exists — please contact the school")
	}

	// Check enrolled students
	var studentCount int64
	s.db.Model(&models.Student{}).
		Where("school_id = ? AND LOWER(full_name) = LOWER(?) AND dob = ? AND is_archived = false",
			schoolID, childName, childDOB).
		Count(&studentCount)
	if studentCount > 0 {
		return errors.New("this child is already enrolled — please contact the school")
	}

	return nil
}

// IsAdmissionWindowOpen checks if there is an active admission window for the given division.
func (s *AdmissionService) IsAdmissionWindowOpen(schoolID uuid.UUID, division models.DivisionScope) (bool, *models.AdmissionWindow, error) {
	var window models.AdmissionWindow
	now := time.Now()

	err := s.db.Where(
		"school_id = ? AND is_active = true AND open_date <= ? AND close_date >= ?",
		schoolID, now, now,
	).First(&window).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	// Check if this division is included in the window
	for _, d := range window.Divisions {
		if d == string(division) {
			return true, &window, nil
		}
	}

	return false, nil, nil
}

// CreateApplication creates a new admission application.
func (s *AdmissionService) CreateApplication(app *models.Application) error {
	// Generate unique reference number
	app.RefNumber = utils.GenerateDocRef("APP")
	app.Status = models.AppStatusPending

	if err := s.db.Create(app).Error; err != nil {
		return fmt.Errorf("admissions: failed to create application: %w", err)
	}

	// Notify parent
	_ = s.notif.Create(
		app.ParentID,
		"Application Submitted",
		fmt.Sprintf("Your application for %s has been received. Reference: %s", app.ChildName, app.RefNumber),
		NotifTypeAdmission,
		app.ID.String(),
	)

	return nil
}

// AssignAppointmentSlot assigns the next available appointment slot to an application.
func (s *AdmissionService) AssignAppointmentSlot(appID uuid.UUID, window *models.AdmissionWindow) error {
	// Find the next available slot (Mon–Fri, 9 AM – 2 PM)
	date := utils.NextWeekday(time.Now().AddDate(0, 0, 2)) // at least 2 days from now

	for {
		for _, slot := range utils.AppointmentTimeSlots {
			// Count existing appointments for this date+time
			var count int64
			s.db.Model(&models.Application{}).
				Where("appointment_date = ? AND appointment_time = ? AND status != ?",
					date, slot, models.AppStatusDeclined).
				Count(&count)

			if int(count) < window.AppointmentCapacityPerSlot {
				// Slot available — assign it
				return s.db.Model(&models.Application{}).
					Where("id = ?", appID).
					Updates(map[string]interface{}{
						"appointment_date": date,
						"appointment_time": slot,
					}).Error
			}
		}
		// All slots on this day are full — try next weekday
		date = utils.NextWeekday(date.AddDate(0, 0, 1))
	}
}

// UpdateStatus updates the application status and notifies the parent.
func (s *AdmissionService) UpdateStatus(appID uuid.UUID, status models.ApplicationStatus, reason string, updatedBy uuid.UUID) error {
	var app models.Application
	if err := s.db.First(&app, "id = ?", appID).Error; err != nil {
		return fmt.Errorf("admissions: application not found: %w", err)
	}

	updates := map[string]interface{}{"status": status}
	if reason != "" {
		updates["decline_reason"] = reason
	}

	if err := s.db.Model(&app).Updates(updates).Error; err != nil {
		return fmt.Errorf("admissions: failed to update status: %w", err)
	}

	// Notify parent based on status
	var title, body string
	switch status {
	case models.AppStatusUnderReview:
		title = "Application Under Review"
		body = fmt.Sprintf("Your application for %s is now under review.", app.ChildName)
	case models.AppStatusAccepted:
		title = "Application Accepted! 🎉"
		body = fmt.Sprintf("Congratulations! %s has been accepted. Please check your portal for the admission letter.", app.ChildName)
	case models.AppStatusDeclined:
		title = "Application Update"
		body = fmt.Sprintf("We regret to inform you that the application for %s was not successful. Reason: %s", app.ChildName, reason)
	}

	if title != "" {
		_ = s.notif.Create(app.ParentID, title, body, NotifTypeAdmission, appID.String())
	}

	return nil
}

// GenerateAdmissionID generates and assigns a permanent Admission ID to a student.
func (s *AdmissionService) GenerateAdmissionID(schoolID uuid.UUID, division models.DivisionScope, schoolPrefix string) (string, error) {
	year := time.Now().Year()

	// Use a DB transaction to safely increment the sequence
	var seq models.AdmissionSequence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("school_id = ? AND division = ? AND year = ?", schoolID, division, year).
			FirstOrCreate(&seq, models.AdmissionSequence{
				SchoolID: schoolID,
				Division: division,
				Year:     year,
				LastSeq:  0,
			})
		if result.Error != nil {
			return result.Error
		}

		seq.LastSeq++
		return tx.Save(&seq).Error
	})
	if err != nil {
		return "", fmt.Errorf("admissions: failed to generate admission ID: %w", err)
	}

	return utils.GenerateAdmissionID(schoolPrefix, utils.DivisionCode(string(division)), seq.LastSeq), nil
}

// RescheduleAppointment assigns a new appointment slot and increments the reschedule count.
func (s *AdmissionService) RescheduleAppointment(appID uuid.UUID, window *models.AdmissionWindow) error {
	var app models.Application
	if err := s.db.First(&app, "id = ?", appID).Error; err != nil {
		return fmt.Errorf("admissions: application not found: %w", err)
	}

	if err := s.AssignAppointmentSlot(appID, window); err != nil {
		return err
	}

	// Increment reschedule count
	s.db.Model(&app).Update("reschedule_count", app.RescheduleCount+1)

	// Notify parent
	_ = s.notif.Create(
		app.ParentID,
		"Appointment Rescheduled",
		fmt.Sprintf("Your appointment for %s has been rescheduled. A new appointment letter will be available shortly.", app.ChildName),
		NotifTypeAdmission,
		appID.String(),
	)

	return nil
}
