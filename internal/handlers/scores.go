package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// ScoresHandler handles score structure setup and score entry endpoints.
type ScoresHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewScoresHandler creates a new ScoresHandler.
func NewScoresHandler(db *gorm.DB, cfg *config.Config) *ScoresHandler {
	return &ScoresHandler{db: db, cfg: cfg}
}

// ─── Score Structure ───────────────────────────────────────────────────────────

// GetScoreStructure handles GET /api/scores/structure
func (h *ScoresHandler) GetScoreStructure(w http.ResponseWriter, r *http.Request) {
	subjectID, _ := uuid.Parse(r.URL.Query().Get("subject_id"))
	classID, _ := uuid.Parse(r.URL.Query().Get("class_id"))
	sessionID, _ := uuid.Parse(r.URL.Query().Get("session_id"))
	termID, _ := uuid.Parse(r.URL.Query().Get("term_id"))

	var structure models.ScoreStructure
	err := h.db.Where("subject_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
		subjectID, classID, sessionID, termID).First(&structure).Error
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, "Score structure not found for this subject/class/term")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "", structure)
}

// UpsertScoreStructure handles POST /api/scores/structure
func (h *ScoresHandler) UpsertScoreStructure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubjectID  string `json:"subject_id"`
		ClassID    string `json:"class_id"`
		SessionID  string `json:"session_id"`
		TermID     string `json:"term_id"`
		Components []struct {
			Name     string `json:"name"`
			MaxMarks int    `json:"max_marks"`
		} `json:"components"`
		ExamMarks int `json:"exam_marks"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("subject_id", req.SubjectID, "Subject")
	v.Required("class_id", req.ClassID, "Class")
	v.Required("session_id", req.SessionID, "Session")
	v.Required("term_id", req.TermID, "Term")
	if len(req.Components) == 0 {
		v.Custom("components", "At least one CA component is required")
	}

	// Validate CA total + exam = 100
	caTotal := 0
	for _, c := range req.Components {
		caTotal += c.MaxMarks
	}
	v.ScoreTotalEquals100("exam_marks", caTotal, req.ExamMarks)

	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	subjectID, _ := uuid.Parse(req.SubjectID)
	classID, _ := uuid.Parse(req.ClassID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)

	components := make(models.ScoreComponentSlice, len(req.Components))
	for i, c := range req.Components {
		components[i] = models.ScoreComponent{Name: c.Name, MaxMarks: c.MaxMarks}
	}

	var structure models.ScoreStructure
	h.db.Where("subject_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
		subjectID, classID, sessionID, termID).FirstOrInit(&structure)

	structure.SubjectID = subjectID
	structure.ClassID = classID
	structure.SessionID = sessionID
	structure.TermID = termID
	structure.Components = components
	structure.ExamMarks = req.ExamMarks
	structure.Total = 100

	if err := h.db.Save(&structure).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save score structure")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Score structure saved", structure)
}

// ─── Score Entry ───────────────────────────────────────────────────────────────

// ListScores handles GET /api/scores — lists score entries, optionally filtered by status.
// Used by the dashboard "Pending Approvals" panel and the scores management screen.
func (h *ScoresHandler) ListScores(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	type scoreRow struct {
		ID          uuid.UUID          `json:"id"`
		SubjectName string             `json:"subject_name"`
		ClassName   string             `json:"class_name"`
		TeacherName string             `json:"teacher_name"`
		Status      models.ScoreStatus `json:"status"`
		SubmittedAt *string            `json:"submitted_at"`
	}

	query := h.db.
		Model(&models.ScoreEntry{}).
		Select(`score_entries.id,
		        subjects.name  AS subject_name,
		        classes.name   AS class_name,
		        users.full_name AS teacher_name,
		        score_entries.status,
		        score_entries.submitted_at`).
		Joins("JOIN subjects ON subjects.id = score_entries.subject_id").
		Joins("JOIN classes  ON classes.id  = score_entries.class_id").
		Joins("JOIN divisions ON divisions.id = classes.division_id").
		Joins("LEFT JOIN users ON users.id = score_entries.teacher_id").
		Where("divisions.school_id = ?", claims.SchoolID)

	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("score_entries.status = ?", status)
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	var rows []scoreRow
	query.Order("score_entries.submitted_at DESC").Limit(limit).Scan(&rows)
	if rows == nil {
		rows = []scoreRow{}
	}

	utils.RespondSuccess(w, http.StatusOK, "", rows)
}

// GetScoreSheet handles GET /api/scores/sheet — returns students + existing scores for a class/subject/term
func (h *ScoresHandler) GetScoreSheet(w http.ResponseWriter, r *http.Request) {
	classID, _ := uuid.Parse(r.URL.Query().Get("class_id"))
	subjectID, _ := uuid.Parse(r.URL.Query().Get("subject_id"))
	sessionID, _ := uuid.Parse(r.URL.Query().Get("session_id"))
	termID, _ := uuid.Parse(r.URL.Query().Get("term_id"))

	// Get score structure
	var structure models.ScoreStructure
	if err := h.db.Where("subject_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
		subjectID, classID, sessionID, termID).First(&structure).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Score structure not configured for this subject/class/term")
		return
	}

	// Get enrolled students for this class/session/term
	var histories []models.StudentClassHistory
	h.db.Preload("Student").
		Where("class_id = ? AND session_id = ? AND term_id = ?", classID, sessionID, termID).
		Find(&histories)

	// Get existing score entries
	var entries []models.ScoreEntry
	h.db.Where("class_id = ? AND subject_id = ? AND session_id = ? AND term_id = ?",
		classID, subjectID, sessionID, termID).Find(&entries)

	entryMap := make(map[uuid.UUID]*models.ScoreEntry)
	for i := range entries {
		entryMap[entries[i].StudentID] = &entries[i]
	}

	type studentRow struct {
		StudentID   uuid.UUID                       `json:"student_id"`
		StudentName string                          `json:"student_name"`
		Components  models.ScoreEntryComponentSlice `json:"components"`
		ExamScore   float64                         `json:"exam_score"`
		Total       float64                         `json:"total"`
		Status      models.ScoreStatus              `json:"status"`
		Remark      string                          `json:"remark"`
	}

	rows := make([]studentRow, 0, len(histories))
	for _, h := range histories {
		row := studentRow{
			StudentID:   h.StudentID,
			StudentName: h.Student.FullName,
		}
		if entry, ok := entryMap[h.StudentID]; ok {
			row.Components = entry.Components
			row.ExamScore = entry.ExamScore
			row.Total = entry.Total
			row.Status = entry.Status
			row.Remark = entry.TeacherRemark
		}
		rows = append(rows, row)
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"structure": structure,
		"students":  rows,
	})
}

// SaveScores handles POST /api/scores/save — save as draft or submit
func (h *ScoresHandler) SaveScores(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		ClassID   string `json:"class_id"`
		SubjectID string `json:"subject_id"`
		SessionID string `json:"session_id"`
		TermID    string `json:"term_id"`
		Submit    bool   `json:"submit"` // false = draft, true = submit for approval
		Scores    []struct {
			StudentID  string `json:"student_id"`
			Components []struct {
				Name  string  `json:"name"`
				Score float64 `json:"score"`
			} `json:"components"`
			ExamScore float64 `json:"exam_score"`
			Remark    string  `json:"remark"`
		} `json:"scores"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	classID, _ := uuid.Parse(req.ClassID)
	subjectID, _ := uuid.Parse(req.SubjectID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)

	// Get score structure for validation
	var structure models.ScoreStructure
	if err := h.db.Where("subject_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
		subjectID, classID, sessionID, termID).First(&structure).Error; err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Score structure not configured")
		return
	}

	// Validate all scores against structure maximums
	v := validators.New()
	for _, s := range req.Scores {
		for _, comp := range s.Components {
			for _, structComp := range structure.Components {
				if comp.Name == structComp.Name {
					v.ScoreNotExceed(
						"score_"+s.StudentID+"_"+comp.Name,
						comp.Score,
						float64(structComp.MaxMarks),
						comp.Name,
					)
				}
			}
		}
		v.ScoreNotExceed("exam_"+s.StudentID, req.Scores[0].ExamScore, float64(structure.ExamMarks), "Exam")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	status := models.ScoreStatusDraft
	if req.Submit {
		// Validate all students are scored before submitting
		var enrolledCount int64
		h.db.Model(&models.StudentClassHistory{}).
			Where("class_id = ? AND session_id = ? AND term_id = ?", classID, sessionID, termID).
			Count(&enrolledCount)
		if int64(len(req.Scores)) < enrolledCount {
			utils.RespondError(w, http.StatusUnprocessableEntity,
				"All students must be scored before submitting — partial submission is not allowed")
			return
		}
		status = models.ScoreStatusSubmitted
	}

	now := utils.NowPtr()
	for _, s := range req.Scores {
		studentID, _ := uuid.Parse(s.StudentID)

		components := make(models.ScoreEntryComponentSlice, len(s.Components))
		for i, c := range s.Components {
			components[i] = models.ScoreEntryComponent{Name: c.Name, Score: c.Score}
		}

		// Calculate CA total
		caTotal := 0.0
		for _, c := range s.Components {
			caTotal += c.Score
		}
		total := caTotal + s.ExamScore

		var entry models.ScoreEntry
		h.db.Where("student_id = ? AND subject_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
			studentID, subjectID, classID, sessionID, termID).FirstOrInit(&entry)

		// Don't allow editing approved/submitted entries without unlock
		if entry.Status == models.ScoreStatusApproved {
			continue
		}
		if entry.Status == models.ScoreStatusSubmitted && !entry.UnlockRequested {
			continue
		}

		entry.StudentID = studentID
		entry.SubjectID = subjectID
		entry.ClassID = classID
		entry.SessionID = sessionID
		entry.TermID = termID
		entry.TeacherID = claims.UserID
		entry.Components = components
		entry.ExamScore = s.ExamScore
		entry.Total = total
		entry.TeacherRemark = s.Remark
		entry.Status = status

		if req.Submit && entry.SubmittedAt == nil {
			entry.SubmittedAt = now
		}

		h.db.Save(&entry)
	}

	msg := "Scores saved as draft"
	if req.Submit {
		msg = "Scores submitted for approval"
	}
	utils.RespondSuccess(w, http.StatusOK, msg, nil)
}

// ApproveScores handles PUT /api/scores/:id/approve
func (h *ScoresHandler) ApproveScores(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid score entry ID")
		return
	}

	now := utils.NowPtr()
	if err := h.db.Model(&models.ScoreEntry{}).
		Where("id = ? AND status = ?", entryID, models.ScoreStatusSubmitted).
		Updates(map[string]interface{}{
			"status":      models.ScoreStatusApproved,
			"approved_at": now,
			"approved_by": claims.UserID,
		}).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to approve scores")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Scores approved", nil)
}

// RejectScores handles PUT /api/scores/:id/reject
func (h *ScoresHandler) RejectScores(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid score entry ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("reason", req.Reason, "Rejection reason")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	if err := h.db.Model(&models.ScoreEntry{}).
		Where("id = ?", entryID).
		Updates(map[string]interface{}{
			"status":           models.ScoreStatusDraft,
			"rejection_reason": req.Reason,
		}).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to reject scores")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Scores returned to teacher for correction", nil)
}

// RequestUnlock handles POST /api/scores/:id/unlock-request
func (h *ScoresHandler) RequestUnlock(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid score entry ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("reason", req.Reason, "Reason for unlock request")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	if err := h.db.Model(&models.ScoreEntry{}).
		Where("id = ?", entryID).
		Updates(map[string]interface{}{
			"unlock_requested": true,
			"unlock_reason":    req.Reason,
		}).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to submit unlock request")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Unlock request submitted", nil)
}
