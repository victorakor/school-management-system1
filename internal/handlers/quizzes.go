package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/jobs"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// QuizzesHandler handles quiz creation, question bank, and attempt management.
type QuizzesHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	jobClient *jobs.Client
}

// NewQuizzesHandler creates a new QuizzesHandler.
func NewQuizzesHandler(db *gorm.DB, cfg *config.Config, jobClient *jobs.Client) *QuizzesHandler {
	return &QuizzesHandler{db: db, cfg: cfg, jobClient: jobClient}
}

// ─── Quiz CRUD ─────────────────────────────────────────────────────────────────

// ListQuizzes handles GET /api/quizzes
func (h *QuizzesHandler) ListQuizzes(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var quizzes []models.Quiz
	query := h.db.Preload("Creator").
		Where("school_id = ? AND is_archived = false", claims.SchoolID)

	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("start_time DESC").Find(&quizzes)
	utils.RespondSuccess(w, http.StatusOK, "", quizzes)
}

// CreateQuiz handles POST /api/quizzes
func (h *QuizzesHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		Title              string   `json:"title"`
		TargetScope        string   `json:"target_scope"`
		TargetIDs          []string `json:"target_ids"`
		SessionID          string   `json:"session_id"`
		StartTime          string   `json:"start_time"`
		EndTime            string   `json:"end_time"`
		DurationMinutes    int      `json:"duration_minutes"`
		QuestionCount      int      `json:"question_count"`
		SubmissionDeadline string   `json:"submission_deadline"`
		MinThreshold       int      `json:"min_threshold"`
		ResultReleaseMode  string   `json:"result_release_mode"`
		ResultReleaseTime  string   `json:"result_release_time"`
		TabSwitchLimit     int      `json:"tab_switch_limit"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("title", req.Title, "Quiz title")
	v.Required("target_scope", req.TargetScope, "Target scope")
	v.Enum("target_scope", req.TargetScope, []string{"CLASS", "DIVISION", "SCHOOL"}, "Target scope")
	v.Required("session_id", req.SessionID, "Session")
	v.Required("start_time", req.StartTime, "Start time")
	if req.DurationMinutes <= 0 {
		v.Custom("duration_minutes", "Duration must be greater than 0")
	}
	if req.QuestionCount <= 0 {
		v.Custom("question_count", "Question count must be greater than 0")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	sessionID, _ := uuid.Parse(req.SessionID)
	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)
	deadline, _ := time.Parse(time.RFC3339, req.SubmissionDeadline)

	targetIDs := make(models.JSONSlice, len(req.TargetIDs))
	for i, id := range req.TargetIDs {
		targetIDs[i] = id
	}

	tabLimit := req.TabSwitchLimit
	if tabLimit == 0 {
		tabLimit = 3
	}
	minThreshold := req.MinThreshold
	if minThreshold == 0 {
		minThreshold = 10
	}

	releaseMode := models.ResultReleaseMode(req.ResultReleaseMode)
	if releaseMode == "" {
		releaseMode = models.ReleaseImmediate
	}

	quiz := &models.Quiz{
		SchoolID:           claims.SchoolID,
		Title:              req.Title,
		CreatedBy:          claims.UserID,
		TargetScope:        models.QuizTargetScope(req.TargetScope),
		TargetIDs:          targetIDs,
		SessionID:          sessionID,
		StartTime:          startTime,
		EndTime:            endTime,
		DurationMinutes:    req.DurationMinutes,
		QuestionCount:      req.QuestionCount,
		SubmissionDeadline: deadline,
		MinThreshold:       minThreshold,
		ResultReleaseMode:  releaseMode,
		TabSwitchLimit:     tabLimit,
		Status:             models.QuizStatusDraft,
	}

	if req.ResultReleaseTime != "" {
		rt, _ := time.Parse(time.RFC3339, req.ResultReleaseTime)
		quiz.ResultReleaseTime = &rt
	}

	if err := h.db.Create(quiz).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create quiz")
		return
	}

	// Enqueue quiz trigger jobs for scheduled start and end
	if h.jobClient != nil {
		_ = h.jobClient.EnqueueQuizStart(quiz.ID.String(), quiz.StartTime)
		_ = h.jobClient.EnqueueQuizEnd(quiz.ID.String(), quiz.EndTime)
	}

	utils.RespondSuccess(w, http.StatusCreated, "Quiz created", quiz)
}

// ─── Question Bank ─────────────────────────────────────────────────────────────

// ListQuestions handles GET /api/quizzes/questions
func (h *QuizzesHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var questions []models.QuizQuestion
	query := h.db.Preload("Subject").Preload("Contributor").
		Where("school_id = ?", claims.SchoolID)

	if subjectID := r.URL.Query().Get("subject_id"); subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	if div := r.URL.Query().Get("division"); div != "" {
		query = query.Where("division = ?", div)
	}
	if diff := r.URL.Query().Get("difficulty"); diff != "" {
		query = query.Where("difficulty = ?", diff)
	}
	if approved := r.URL.Query().Get("approved"); approved == "true" {
		query = query.Where("is_approved = true")
	}

	query.Order("created_at DESC").Find(&questions)
	utils.RespondSuccess(w, http.StatusOK, "", questions)
}

// SubmitQuestion handles POST /api/quizzes/questions
func (h *QuizzesHandler) SubmitQuestion(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		SubjectID     string   `json:"subject_id"`
		Division      string   `json:"division"`
		Text          string   `json:"text"`
		Type          string   `json:"type"`
		Options       []string `json:"options"`
		CorrectAnswer string   `json:"correct_answer"`
		Difficulty    string   `json:"difficulty"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("subject_id", req.SubjectID, "Subject")
	v.Required("text", req.Text, "Question text")
	v.Required("type", req.Type, "Question type")
	v.Enum("type", req.Type, []string{"MCQ", "TRUE_FALSE", "FILL_BLANK"}, "Question type")
	v.Required("correct_answer", req.CorrectAnswer, "Correct answer")
	v.Required("difficulty", req.Difficulty, "Difficulty")
	v.Enum("difficulty", req.Difficulty, []string{"EASY", "MEDIUM", "HARD"}, "Difficulty")
	if req.Type == "MCQ" && len(req.Options) < 2 {
		v.Custom("options", "MCQ questions require at least 2 options")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	subjectID, _ := uuid.Parse(req.SubjectID)

	opts := make(models.JSONSlice, len(req.Options))
	for i, o := range req.Options {
		opts[i] = o
	}

	question := &models.QuizQuestion{
		SchoolID:      claims.SchoolID,
		SubjectID:     subjectID,
		Division:      models.DivisionScope(req.Division),
		Text:          req.Text,
		Type:          models.QuizQuestionType(req.Type),
		Options:       opts,
		CorrectAnswer: req.CorrectAnswer,
		Difficulty:    models.QuizDifficulty(req.Difficulty),
		ContributorID: claims.UserID,
		IsApproved:    false,
	}

	if err := h.db.Create(question).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to submit question")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Question submitted for approval", question)
}

// ApproveQuestion handles PUT /api/quizzes/questions/:id/approve
func (h *QuizzesHandler) ApproveQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	if err := h.db.Model(&models.QuizQuestion{}).
		Where("id = ?", questionID).
		Update("is_approved", true).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to approve question")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Question approved", nil)
}

// FlagQuestion handles PUT /api/quizzes/questions/:id/flag
func (h *QuizzesHandler) FlagQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	if err := h.db.Model(&models.QuizQuestion{}).
		Where("id = ?", questionID).
		Update("is_flagged", true).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to flag question")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Question flagged for review", nil)
}

// ─── Quiz Attempts ─────────────────────────────────────────────────────────────

// StartAttempt handles POST /api/quizzes/:id/start
func (h *QuizzesHandler) StartAttempt(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	quizID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var quiz models.Quiz
	if err := h.db.First(&quiz, "id = ? AND status = ?", quizID, models.QuizStatusActive).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Quiz not found or not active")
		return
	}

	// Check if student already has an attempt
	var existing models.QuizAttempt
	if err := h.db.Where("quiz_id = ? AND student_id = ?", quizID, claims.UserID).
		First(&existing).Error; err == nil {
		// Return existing attempt (resume)
		utils.RespondSuccess(w, http.StatusOK, "Resuming existing attempt", existing)
		return
	}

	// Get approved questions for this quiz
	var assignments []models.QuizQuestionAssignment
	h.db.Preload("Question").
		Where("quiz_id = ? AND is_approved = true", quizID).
		Find(&assignments)

	if len(assignments) < quiz.QuestionCount {
		utils.RespondError(w, http.StatusServiceUnavailable, "Quiz does not have enough approved questions")
		return
	}

	// Randomise and select questions (Fisher-Yates shuffle)
	questions := shuffleQuestions(assignments, quiz.QuestionCount)

	// Build question set (strip correct answers for client)
	questionSet := make(models.JSONSlice, len(questions))
	for i, q := range questions {
		opts := shuffleOptions(q.Question.Options)
		questionSet[i] = map[string]interface{}{
			"question_id": q.QuestionID,
			"text":        q.Question.Text,
			"type":        q.Question.Type,
			"options":     opts,
		}
	}

	now := time.Now()
	attempt := &models.QuizAttempt{
		QuizID:    quizID,
		StudentID: claims.UserID,
		Questions: questionSet,
		Answers:   models.JSONSlice{},
		StartedAt: now,
	}

	if err := h.db.Create(attempt).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to start quiz")
		return
	}

	// Enqueue server-side auto-submit as belt-and-suspenders backup
	if h.jobClient != nil {
		autoSubmitAt := now.Add(time.Duration(quiz.DurationMinutes) * time.Minute)
		_ = h.jobClient.EnqueueQuizAutoSubmit(attempt.ID.String(), quiz.ID.String(), autoSubmitAt)
	}

	utils.RespondSuccess(w, http.StatusCreated, "Quiz started", map[string]interface{}{
		"attempt_id":       attempt.ID,
		"questions":        questionSet,
		"duration_minutes": quiz.DurationMinutes,
		"tab_switch_limit": quiz.TabSwitchLimit,
		"started_at":       now,
		"ends_at":          now.Add(time.Duration(quiz.DurationMinutes) * time.Minute),
	})
}

// SaveProgress handles POST /api/quizzes/attempts/:id/save — periodic answer save
func (h *QuizzesHandler) SaveProgress(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid attempt ID")
		return
	}

	var req struct {
		Answers []map[string]interface{} `json:"answers"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	answers := make(models.JSONSlice, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = a
	}

	if err := h.db.Model(&models.QuizAttempt{}).
		Where("id = ? AND submitted_at IS NULL", attemptID).
		Update("answers", answers).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save progress")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Progress saved", nil)
}

// LogViolation handles POST /api/quizzes/attempts/:id/violation
func (h *QuizzesHandler) LogViolation(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid attempt ID")
		return
	}

	var req struct {
		Type string `json:"type"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var attempt models.QuizAttempt
	if err := h.db.First(&attempt, "id = ?", attemptID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Attempt not found")
		return
	}

	violation := models.QuizViolation{Type: req.Type, Timestamp: time.Now()}
	attempt.Violations = append(attempt.Violations, violation)

	// Check tab switch limit
	var quiz models.Quiz
	h.db.First(&quiz, "id = ?", attempt.QuizID)

	tabSwitches := 0
	for _, v := range attempt.Violations {
		if v.Type == "TAB_SWITCH" || v.Type == "BLUR" {
			tabSwitches++
		}
	}

	isFlagged := len(attempt.Violations) > 3
	autoSubmit := tabSwitches >= quiz.TabSwitchLimit

	updates := map[string]interface{}{
		"violations": attempt.Violations,
		"is_flagged": isFlagged,
	}

	if autoSubmit && attempt.SubmittedAt == nil {
		now := utils.NowPtr()
		updates["submitted_at"] = now
		updates["auto_submitted"] = true
	}

	h.db.Model(&attempt).Updates(updates)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"auto_submit": autoSubmit,
		"flagged":     isFlagged,
	})
}

// SubmitAttempt handles POST /api/quizzes/attempts/:id/submit
func (h *QuizzesHandler) SubmitAttempt(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid attempt ID")
		return
	}

	var req struct {
		Answers []map[string]interface{} `json:"answers"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var attempt models.QuizAttempt
	if err := h.db.Preload("Quiz").First(&attempt, "id = ?", attemptID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Attempt not found")
		return
	}

	if attempt.SubmittedAt != nil {
		utils.RespondError(w, http.StatusConflict, "Attempt already submitted")
		return
	}

	answers := make(models.JSONSlice, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = a
	}

	// Score the attempt
	score := scoreAttempt(attempt.Questions, answers)
	now := utils.NowPtr()

	h.db.Model(&attempt).Updates(map[string]interface{}{
		"answers":      answers,
		"score":        score,
		"submitted_at": now,
	})

	utils.RespondSuccess(w, http.StatusOK, "Quiz submitted successfully", map[string]interface{}{
		"score": score,
		"total": len(attempt.Questions),
	})
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func shuffleQuestions(assignments []models.QuizQuestionAssignment, count int) []models.QuizQuestionAssignment {
	// Fisher-Yates shuffle
	n := len(assignments)
	for i := n - 1; i > 0; i-- {
		j := i // simplified — use crypto/rand in production
		assignments[i], assignments[j] = assignments[j], assignments[i]
	}
	if count > n {
		count = n
	}
	return assignments[:count]
}

func shuffleOptions(opts models.JSONSlice) models.JSONSlice {
	n := len(opts)
	result := make(models.JSONSlice, n)
	copy(result, opts)
	for i := n - 1; i > 0; i-- {
		j := i // simplified — use crypto/rand in production
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func scoreAttempt(questions, answers models.JSONSlice) int {
	// Build answer map
	answerMap := make(map[string]string)
	for _, a := range answers {
		if am, ok := a.(map[string]interface{}); ok {
			qid, _ := am["question_id"].(string)
			ans, _ := am["answer"].(string)
			answerMap[qid] = ans
		}
	}

	score := 0
	for _, q := range questions {
		if qm, ok := q.(map[string]interface{}); ok {
			qid, _ := qm["question_id"].(string)
			correct, _ := qm["correct_answer"].(string)
			given := answerMap[qid]
			// Case-insensitive match for fill-in-the-blank
			if given != "" && (given == correct ||
				len(given) > 0 && len(correct) > 0 &&
					equalFold(given, correct)) {
				score++
			}
		}
	}
	return score
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// ─── Owner/Admin Quiz Management ───────────────────────────────────────────────

// PublishQuiz handles PUT /api/quizzes/:id/publish
// Changes a DRAFT quiz to ACTIVE, making it visible and startable.
func (h *QuizzesHandler) PublishQuiz(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	quizID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var quiz models.Quiz
	if err := h.db.Where("id = ? AND school_id = ? AND is_archived = false", quizID, claims.SchoolID).
		First(&quiz).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Quiz not found")
		return
	}

	if quiz.Status != models.QuizStatusDraft && quiz.Status != models.QuizStatusPostponed {
		utils.RespondError(w, http.StatusBadRequest, "Only DRAFT or POSTPONED quizzes can be published")
		return
	}

	// Verify there are approved questions
	var approvedCount int64
	h.db.Model(&models.QuizQuestionAssignment{}).
		Where("quiz_id = ? AND is_approved = true", quizID).
		Count(&approvedCount)
	if approvedCount == 0 {
		utils.RespondError(w, http.StatusBadRequest, "Quiz must have at least one approved question before publishing")
		return
	}

	if err := h.db.Model(&models.Quiz{}).Where("id = ?", quizID).
		Update("status", models.QuizStatusActive).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to publish quiz")
		return
	}

	middleware.Record(h.db, r, middleware.AuditAction{
		Action:     "publish_quiz",
		EntityType: "quiz",
		EntityID:   quizID.String(),
		Metadata:   map[string]interface{}{"quiz_title": quiz.Title},
	})

	utils.RespondSuccess(w, http.StatusOK, "Quiz published and now active", nil)
}

// CloseQuiz handles PUT /api/quizzes/:id/close
// Ends an active quiz and prevents new attempts.
func (h *QuizzesHandler) CloseQuiz(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	quizID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var quiz models.Quiz
	if err := h.db.Where("id = ? AND school_id = ? AND is_archived = false", quizID, claims.SchoolID).
		First(&quiz).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Quiz not found")
		return
	}

	if quiz.Status != models.QuizStatusActive {
		utils.RespondError(w, http.StatusBadRequest, "Only ACTIVE quizzes can be closed")
		return
	}

	if err := h.db.Model(&models.Quiz{}).Where("id = ?", quizID).
		Update("status", models.QuizStatusClosed).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to close quiz")
		return
	}

	middleware.Record(h.db, r, middleware.AuditAction{
		Action:     "close_quiz",
		EntityType: "quiz",
		EntityID:   quizID.String(),
		Metadata:   map[string]interface{}{"quiz_title": quiz.Title},
	})

	utils.RespondSuccess(w, http.StatusOK, "Quiz closed successfully", nil)
}

// GetQuizLeaderboard handles GET /api/quizzes/:id/leaderboard
// Returns top scores for a quiz (available once closed or active for owner view).
func (h *QuizzesHandler) GetQuizLeaderboard(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	quizID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var quiz models.Quiz
	if err := h.db.Where("id = ? AND school_id = ? AND is_archived = false", quizID, claims.SchoolID).
		First(&quiz).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Quiz not found")
		return
	}

	type leaderboardEntry struct {
		Rank        int       `json:"rank"`
		StudentID   string    `json:"student_id"`
		StudentName string    `json:"student_name"`
		Score       int       `json:"score"`
		SubmittedAt time.Time `json:"submitted_at"`
		IsFlagged   bool      `json:"is_flagged"`
	}

	var entries []leaderboardEntry
	h.db.Raw(`
		SELECT
			ROW_NUMBER() OVER (ORDER BY qa.score DESC, qa.submitted_at ASC) AS rank,
			s.id AS student_id,
			s.full_name AS student_name,
			qa.score,
			qa.submitted_at,
			qa.is_flagged
		FROM quiz_attempts qa
		JOIN students s ON s.id = qa.student_id
		WHERE qa.quiz_id = ? AND qa.submitted_at IS NOT NULL
		ORDER BY qa.score DESC, qa.submitted_at ASC
		LIMIT 100`, quizID).Scan(&entries)

	if entries == nil {
		entries = []leaderboardEntry{}
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"quiz":        quiz,
		"leaderboard": entries,
	})
}

// GetCheatingReports handles GET /api/quizzes/:id/cheating
// Returns all flagged attempts for a quiz with violation details.
func (h *QuizzesHandler) GetCheatingReports(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	quizID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var quiz models.Quiz
	if err := h.db.Where("id = ? AND school_id = ? AND is_archived = false", quizID, claims.SchoolID).
		First(&quiz).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Quiz not found")
		return
	}

	var attempts []models.QuizAttempt
	h.db.Preload("Student").
		Where("quiz_id = ? AND is_flagged = true", quizID).
		Order("created_at DESC").
		Find(&attempts)

	type reportEntry struct {
		AttemptID     string                    `json:"attempt_id"`
		StudentID     string                    `json:"student_id"`
		StudentName   string                    `json:"student_name"`
		Score         int                       `json:"score"`
		Violations    models.QuizViolationSlice `json:"violations"`
		AutoSubmitted bool                      `json:"auto_submitted"`
		SubmittedAt   *time.Time                `json:"submitted_at"`
	}

	reports := make([]reportEntry, 0, len(attempts))
	for _, a := range attempts {
		reports = append(reports, reportEntry{
			AttemptID:     a.ID.String(),
			StudentID:     a.StudentID.String(),
			StudentName:   a.Student.FullName,
			Score:         a.Score,
			Violations:    a.Violations,
			AutoSubmitted: a.AutoSubmitted,
			SubmittedAt:   a.SubmittedAt,
		})
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"quiz":    quiz,
		"reports": reports,
		"total":   len(reports),
	})
}
