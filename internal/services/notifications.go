// Package services contains all business logic called by HTTP handlers.
package services

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/models"
)

// NotificationService handles in-app notification creation and retrieval.
type NotificationService struct {
	db *gorm.DB
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// Create creates a new in-app notification for a user.
func (s *NotificationService) Create(userID uuid.UUID, title, body, notifType, entityID string) error {
	notif := &models.Notification{
		UserID:   userID,
		Title:    title,
		Body:     body,
		Type:     notifType,
		EntityID: entityID,
		IsRead:   false,
	}
	if err := s.db.Create(notif).Error; err != nil {
		return fmt.Errorf("notifications: failed to create: %w", err)
	}
	return nil
}

// CreateBulk creates notifications for multiple users at once.
func (s *NotificationService) CreateBulk(userIDs []uuid.UUID, title, body, notifType, entityID string) error {
	if len(userIDs) == 0 {
		return nil
	}
	notifs := make([]models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		n := models.Notification{
			UserID:   uid,
			Title:    title,
			Body:     body,
			Type:     notifType,
			EntityID: entityID,
			IsRead:   false,
		}
		n.ID = uuid.New()
		notifs = append(notifs, n)
	}
	return s.db.CreateInBatches(notifs, 100).Error
}

// GetForUser returns paginated notifications for a user.
func (s *NotificationService) GetForUser(userID uuid.UUID, page, pageSize int) ([]models.Notification, int64, error) {
	var notifs []models.Notification
	var total int64

	offset := (page - 1) * pageSize

	if err := s.db.Model(&models.Notification{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notifs).Error; err != nil {
		return nil, 0, err
	}

	return notifs, total, nil
}

// GetUnreadCount returns the count of unread notifications for a user.
func (s *NotificationService) GetUnreadCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}

// MarkRead marks a specific notification as read.
func (s *NotificationService) MarkRead(notifID, userID uuid.UUID) error {
	return s.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("is_read", true).Error
}

// MarkAllRead marks all notifications for a user as read.
func (s *NotificationService) MarkAllRead(userID uuid.UUID) error {
	return s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

// ─── Notification Type Constants ───────────────────────────────────────────────

const (
	NotifTypeAdmission   = "ADMISSION"
	NotifTypeResult      = "RESULT"
	NotifTypeQuiz        = "QUIZ"
	NotifTypePayment     = "PAYMENT"
	NotifTypeTimetable   = "TIMETABLE"
	NotifTypeAnnouncement = "ANNOUNCEMENT"
	NotifTypeSystem      = "SYSTEM"
	NotifTypeScore       = "SCORE"
	NotifTypePromotion   = "PROMOTION"
)
