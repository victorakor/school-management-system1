package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// AttendanceHandler handles attendance marking and reporting.
type AttendanceHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAttendanceHandler creates a new AttendanceHandler.
func NewAttendanceHandler(db *gorm.DB, cfg *config.Config) *AttendanceHandler {
	return &AttendanceHandler{db: db, cfg: cfg}
}

// MarkAttendance handles POST /api/attendance — bulk mark attendance for a class
func (h *AttendanceHandler) MarkAttendance(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var req struct {
		ClassID   string `json:"class_id"`
		SubjectID string `json:"subject_id"`
		SessionID string `json:"session_id"`
		TermID    string `json:"term_id"`
		Date      string `json:"date"`
		Records   []struct {
			StudentID string `json:"student_id"`
			Status    string `json:"status"`
		} `json:"records"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	v := validators.New()
	v.Required("class_id", req.ClassID, "Class")
	v.Required("session_id", req.SessionID, "Session")
	v.Required("term_id", req.TermID, "Term")
	if len(req.Records) == 0 {
		v.Custom("records", "At least one attendance record is required")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}
	classID, _ := uuid.Parse(req.ClassID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)
	var subjectIDPtr *uuid.UUID
	if req.SubjectID != "" {
		sid, _ := uuid.Parse(req.SubjectID)
		subjectIDPtr = &sid
	}
	date := time.Now().Truncate(24 * time.Hour)
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			date = t
		}
	}
	var created, updated int
	for _, rec := range req.Records {
		studentID, err := uuid.Parse(rec.StudentID)
		if err != nil {
			continue
		}
		status := models.AttendanceStatus(rec.Status)
		if status != models.AttendancePresent && status != models.AttendanceAbsent && status != models.AttendanceLate {
			status = models.AttendanceAbsent
		}
		var existing models.Attendance
		err = h.db.Where("student_id = ? AND class_id = ? AND date = ? AND session_id = ? AND term_id = ?",
			studentID, classID, date, sessionID, termID).First(&existing).Error
		if err == nil {
			h.db.Model(&existing).Update("status", status)
			updated++
		} else {
			att := &models.Attendance{
				StudentID: studentID,
				ClassID:   classID,
				SubjectID: subjectIDPtr,
				SessionID: sessionID,
				TermID:    termID,
				TeacherID: claims.UserID,
				Date:      date,
				Status:    status,
			}
			h.db.Create(att)
			created++
		}
	}
	utils.RespondSuccess(w, http.StatusOK, "Attendance marked", map[string]interface{}{
		"created": created,
		"updated": updated,
	})
}

// GetAttendance handles GET /api/attendance — get attendance for a class/date
func (h *AttendanceHandler) GetAttendance(w http.ResponseWriter, r *http.Request) {
	classID, _ := uuid.Parse(r.URL.Query().Get("class_id"))
	sessionID, _ := uuid.Parse(r.URL.Query().Get("session_id"))
	termID, _ := uuid.Parse(r.URL.Query().Get("term_id"))
	dateStr := r.URL.Query().Get("date")

	query := h.db.Where("class_id = ? AND session_id = ? AND term_id = ?", classID, sessionID, termID)
	if dateStr != "" {
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			query = query.Where("date = ?", date)
		}
	}

	var records []models.Attendance
	query.Order("date DESC").Find(&records)
	utils.RespondSuccess(w, http.StatusOK, "", records)
}

// GetStudentAttendanceSummary handles GET /api/attendance/summary/:studentId
func (h *AttendanceHandler) GetStudentAttendanceSummary(w http.ResponseWriter, r *http.Request) {
	studentIDStr := r.URL.Query().Get("student_id")
	sessionID, _ := uuid.Parse(r.URL.Query().Get("session_id"))
	termID, _ := uuid.Parse(r.URL.Query().Get("term_id"))

	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	var total, present, absent, late int64
	h.db.Model(&models.Attendance{}).
		Where("student_id = ? AND session_id = ? AND term_id = ?", studentID, sessionID, termID).
		Count(&total)
	h.db.Model(&models.Attendance{}).
		Where("student_id = ? AND session_id = ? AND term_id = ? AND status = ?", studentID, sessionID, termID, models.AttendancePresent).
		Count(&present)
	h.db.Model(&models.Attendance{}).
		Where("student_id = ? AND session_id = ? AND term_id = ? AND status = ?", studentID, sessionID, termID, models.AttendanceAbsent).
		Count(&absent)
	h.db.Model(&models.Attendance{}).
		Where("student_id = ? AND session_id = ? AND term_id = ? AND status = ?", studentID, sessionID, termID, models.AttendanceLate).
		Count(&late)

	percentage := 0.0
	if total > 0 {
		percentage = float64(present+late) / float64(total) * 100
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"total":      total,
		"present":    present,
		"absent":     absent,
		"late":       late,
		"percentage": percentage,
	})
}
