package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/middleware"
	"school-platform/internal/services"
	"school-platform/internal/utils"
)

// NotificationsHandler handles notification endpoints.
type NotificationsHandler struct {
	db    *gorm.DB
	notif *services.NotificationService
}

// NewNotificationsHandler creates a new NotificationsHandler.
func NewNotificationsHandler(db *gorm.DB, notif *services.NotificationService) *NotificationsHandler {
	return &NotificationsHandler{db: db, notif: notif}
}

// ListNotifications handles GET /api/notifications
func (h *NotificationsHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	notifs, total, err := h.notif.GetForUser(claims.UserID, page, pageSize)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"data":       notifs,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetUnreadCount handles GET /api/notifications/unread-count
func (h *NotificationsHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	count, err := h.notif.GetUnreadCount(claims.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch count")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{"count": count})
}

// MarkRead handles PUT /api/notifications/:id/read
func (h *NotificationsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	notifID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	if err := h.notif.MarkRead(notifID, claims.UserID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to mark as read")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Marked as read", nil)
}

// MarkAllRead handles PUT /api/notifications/read-all
func (h *NotificationsHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	if err := h.notif.MarkAllRead(claims.UserID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to mark all as read")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "All notifications marked as read", nil)
}
