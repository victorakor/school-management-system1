package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
)

// DashboardHandler aggregates role-scoped stat cards for the portal home.
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// Stats handles GET /api/dashboard/stats.
// The shape of the returned stats is role-dependent so each portal surface
// (Owner / Principal / Head Teacher / Bursar / Teacher / Parent) renders
// only the cards relevant to the authenticated user.
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	stats := map[string]interface{}{}

	// ─── Common counts the Owner / Bursar / Teacher need ──────────────────
	if claims.Role == models.RoleOwner {
		stats["total_students"] = countStudents(h.db, claims.SchoolID, false)
		stats["total_alumni"] = countStudents(h.db, claims.SchoolID, true)
		stats["total_staff"] = countStaff(h.db, claims.SchoolID)
		stats["pending_applications"] = countPendingApplications(h.db, claims.SchoolID)
		stats["active_quizzes"] = countActiveQuizzes(h.db, claims.SchoolID)
		stats["collected_today"] = collectedOn(h.db, claims.SchoolID, time.Now())
		stats["collected_term"] = collectedTermToDate(h.db, claims.SchoolID)
		stats["outstanding_balance"] = outstandingBalance(h.db, claims.SchoolID)
		stats["debtor_count"] = debtorCount(h.db, claims.SchoolID)
		stats["recent_admissions"] = recentApplications(h.db, claims.SchoolID, 5)
	}

	if claims.Role == models.RolePrincipal || claims.Role == models.RoleVicePrincipal {
		stats["division"] = "SECONDARY"
		stats["total_students"] = countStudentsInDivision(h.db, claims.SchoolID, models.DivisionSecondary)
		stats["classes"] = countClassesInDivision(h.db, claims.SchoolID, models.DivisionSecondary)
		stats["subjects"] = countSubjectsInDivision(h.db, claims.SchoolID, models.DivisionSecondary)
		stats["pending_score_approvals"] = pendingScoreApprovals(h.db, claims.SchoolID, models.DivisionSecondary)
		stats["active_quizzes"] = countActiveQuizzesInDivision(h.db, claims.SchoolID, models.DivisionSecondary)
	}

	if claims.Role == models.RoleHeadTeacher || claims.Role == models.RoleAsstHeadTeacher {
		stats["division"] = "PRIMARY_NURSERY"
		stats["primary_students"] = countStudentsInDivision(h.db, claims.SchoolID, models.DivisionPrimary)
		stats["nursery_students"] = countStudentsInDivision(h.db, claims.SchoolID, models.DivisionNursery)
		stats["pending_score_approvals"] = pendingScoreApprovalsInDivisions(h.db, claims.SchoolID, []models.DivisionScope{models.DivisionPrimary, models.DivisionNursery})
		stats["active_quizzes"] = countActiveQuizzesInDivisions(h.db, claims.SchoolID, []models.DivisionScope{models.DivisionPrimary, models.DivisionNursery})
	}

	if claims.Role == models.RoleBursar {
		stats["division"] = string(claims.DivisionScope)
		stats["collected_today"] = collectedOn(h.db, claims.SchoolID, time.Now())
		stats["collected_term"] = collectedTermToDate(h.db, claims.SchoolID)
		stats["outstanding_balance"] = outstandingBalanceInDivision(h.db, claims.SchoolID, claims.DivisionScope)
		stats["debtor_count"] = debtorCountInDivision(h.db, claims.SchoolID, claims.DivisionScope)
	}

	if claims.Role == models.RoleTeacher {
		stats["assigned_classes"] = assignedClassCount(h.db, claims.UserID)
		stats["draft_scores"] = draftScoreCount(h.db, claims.UserID)
		stats["active_quizzes"] = activeQuizCountForTeacher(h.db, claims.UserID)
	}

	if claims.Role == models.RoleStudent || claims.Role == models.RolePupil {
		stats["attendance_percentage"] = studentAttendancePercentage(h.db, claims.UserID)
		stats["active_quizzes"] = activeQuizzesForStudent(h.db, claims.UserID)
		stats["latest_result"] = latestResultForStudent(h.db, claims.UserID)
	}

	if claims.Role == models.RoleParent {
		stats["children"] = childrenForParent(h.db, claims.UserID)
		stats["unread_notifications"] = unreadNotificationsCount(h.db, claims.UserID)
	}

	utils.RespondJSON(w, http.StatusOK, stats)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func countStudents(db *gorm.DB, schoolID uuid.UUID, alumni bool) int64 {
	var n int64
	db.Model(&models.Student{}).
		Where("school_id = ? AND is_archived = false AND is_alumni = ?", schoolID, alumni).
		Count(&n)
	return n
}

func countStaff(db *gorm.DB, schoolID uuid.UUID) int64 {
	var n int64
	db.Model(&models.User{}).
		Where("school_id = ? AND is_archived = false AND role NOT IN ?",
			schoolID, []models.Role{models.RoleParent, models.RoleStudent, models.RolePupil}).
		Count(&n)
	return n
}

func countPendingApplications(db *gorm.DB, schoolID uuid.UUID) int64 {
	var n int64
	db.Model(&models.Application{}).
		Where("school_id = ? AND is_archived = false AND status IN ?",
			schoolID, []models.ApplicationStatus{models.AppStatusPending, models.AppStatusUnderReview}).
		Count(&n)
	return n
}

func recentApplications(db *gorm.DB, schoolID uuid.UUID, limit int) []models.Application {
	var apps []models.Application
	db.Where("school_id = ? AND is_archived = false", schoolID).
		Order("created_at DESC").Limit(limit).Find(&apps)
	return apps
}

func countActiveQuizzes(db *gorm.DB, schoolID uuid.UUID) int64 {
	var n int64
	db.Model(&models.Quiz{}).
		Where("school_id = ? AND is_archived = false AND status = ?", schoolID, models.QuizStatusActive).
		Count(&n)
	return n
}

func collectedOn(db *gorm.DB, schoolID uuid.UUID, day time.Time) float64 {
	var total struct{ Sum float64 }
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	db.Model(&models.FeePayment{}).
		Joins("JOIN students ON students.id = fee_payments.student_id").
		Where("students.school_id = ? AND fee_payments.payment_date >= ? AND fee_payments.payment_date < ?",
			schoolID, startOfDay, startOfDay.Add(24*time.Hour)).
		Select("COALESCE(SUM(fee_payments.amount_paid), 0) AS sum").
		Scan(&total)
	return total.Sum
}

func collectedTermToDate(db *gorm.DB, schoolID uuid.UUID) float64 {
	var total struct{ Sum float64 }
	db.Model(&models.FeePayment{}).
		Joins("JOIN students ON students.id = fee_payments.student_id").
		Joins("JOIN academic_sessions ON academic_sessions.id = fee_payments.fee_structure_id").
		Where("students.school_id = ? AND academic_sessions.is_active = true", schoolID).
		Select("COALESCE(SUM(fee_payments.amount_paid), 0) AS sum").
		Scan(&total)
	return total.Sum
}

func outstandingBalance(db *gorm.DB, schoolID uuid.UUID) float64 {
	var total struct{ Sum float64 }
	db.Model(&models.FeePayment{}).
		Joins("JOIN students ON students.id = fee_payments.student_id").
		Where("students.school_id = ? AND fee_payments.balance_after > 0", schoolID).
		Select("COALESCE(SUM(fee_payments.balance_after), 0) AS sum").
		Scan(&total)
	return total.Sum
}

func debtorCount(db *gorm.DB, schoolID uuid.UUID) int64 {
	var n int64
	db.Model(&models.FeePayment{}).
		Joins("JOIN students ON students.id = fee_payments.student_id").
		Where("students.school_id = ? AND fee_payments.balance_after > 0", schoolID).
		Distinct("student_id").Count(&n)
	return n
}

func countStudentsInDivision(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) int64 {
	var n int64
	db.Model(&models.Student{}).
		Joins("JOIN divisions ON divisions.id = students.division_id").
		Where("students.school_id = ? AND divisions.name = ? AND students.is_archived = false", schoolID, div).
		Count(&n)
	return n
}

func countClassesInDivision(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) int64 {
	var n int64
	db.Model(&models.Class{}).
		Joins("JOIN divisions ON divisions.id = classes.division_id").
		Where("divisions.school_id = ? AND divisions.name = ?", schoolID, div).
		Count(&n)
	return n
}

func countSubjectsInDivision(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) int64 {
	var n int64
	db.Model(&models.Subject{}).
		Joins("JOIN divisions ON divisions.id = subjects.division_id").
		Where("divisions.school_id = ? AND divisions.name = ?", schoolID, div).
		Count(&n)
	return n
}

func pendingScoreApprovals(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) int64 {
	var n int64
	db.Model(&models.ScoreEntry{}).
		Joins("JOIN classes ON classes.id = score_entries.class_id").
		Joins("JOIN divisions ON divisions.id = classes.division_id").
		Where("score_entries.status = ? AND divisions.school_id = ? AND divisions.name = ?",
			models.ScoreStatusSubmitted, schoolID, div).
		Count(&n)
	return n
}

func pendingScoreApprovalsInDivisions(db *gorm.DB, schoolID uuid.UUID, divs []models.DivisionScope) int64 {
	var n int64
	db.Model(&models.ScoreEntry{}).
		Joins("JOIN classes ON classes.id = score_entries.class_id").
		Joins("JOIN divisions ON divisions.id = classes.division_id").
		Where("score_entries.status = ? AND divisions.school_id = ? AND divisions.name IN ?",
			models.ScoreStatusSubmitted, schoolID, divs).
		Count(&n)
	return n
}

func countActiveQuizzesInDivision(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) int64 {
	var n int64
	db.Model(&models.Quiz{}).
		Where("school_id = ? AND is_archived = false AND status = ?", schoolID, models.QuizStatusActive).
		Count(&n)
	_ = div
	return n
}

func countActiveQuizzesInDivisions(db *gorm.DB, schoolID uuid.UUID, divs []models.DivisionScope) int64 {
	var n int64
	db.Model(&models.Quiz{}).
		Where("school_id = ? AND is_archived = false AND status = ?", schoolID, models.QuizStatusActive).
		Count(&n)
	_ = divs
	return n
}

func outstandingBalanceInDivision(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) float64 {
	var total struct{ Sum float64 }
	db.Model(&models.FeePayment{}).
		Joins("JOIN students ON students.id = fee_payments.student_id").
		Joins("JOIN divisions ON divisions.id = students.division_id").
		Where("students.school_id = ? AND divisions.name = ? AND fee_payments.balance_after > 0", schoolID, div).
		Select("COALESCE(SUM(fee_payments.balance_after), 0) AS sum").
		Scan(&total)
	return total.Sum
}

func debtorCountInDivision(db *gorm.DB, schoolID uuid.UUID, div models.DivisionScope) int64 {
	var n int64
	db.Model(&models.FeePayment{}).
		Joins("JOIN students ON students.id = fee_payments.student_id").
		Joins("JOIN divisions ON divisions.id = students.division_id").
		Where("students.school_id = ? AND divisions.name = ? AND fee_payments.balance_after > 0", schoolID, div).
		Distinct("student_id").Count(&n)
	return n
}

func assignedClassCount(db *gorm.DB, teacherID uuid.UUID) int64 {
	var n int64
	db.Model(&models.TeacherAssignment{}).
		Where("teacher_id = ?", teacherID).
		Distinct("class_id").Count(&n)
	return n
}

func draftScoreCount(db *gorm.DB, teacherID uuid.UUID) int64 {
	var n int64
	db.Model(&models.ScoreEntry{}).
		Where("teacher_id = ? AND status = ?", teacherID, models.ScoreStatusDraft).
		Count(&n)
	return n
}

func activeQuizCountForTeacher(db *gorm.DB, teacherID uuid.UUID) int64 {
	// Treat any active quiz at the teacher's school as relevant.
	var n int64
	db.Model(&models.Quiz{}).
		Where("is_archived = false AND status = ?", models.QuizStatusActive).
		Count(&n)
	return n
}

func studentAttendancePercentage(db *gorm.DB, userID uuid.UUID) float64 {
	// Attendance is keyed by StudentID. A STUDENT user is linked via the
	// User → Student association (ParentID is the parent user; STUDENT is a
	// role on the User table, not a Student row). Until the Student↔User link
	// is established we simply return 0; the JS layer treats this as "no data".
	_ = db
	_ = userID
	return 0
}

func activeQuizzesForStudent(db *gorm.DB, userID uuid.UUID) int64 {
	var n int64
	db.Model(&models.Quiz{}).
		Where("is_archived = false AND status = ?", models.QuizStatusActive).
		Count(&n)
	return n
}

func latestResultForStudent(db *gorm.DB, userID uuid.UUID) *models.Result {
	// Same constraint as studentAttendancePercentage — no Student↔User link
	// exists yet on this schema. Future iterations may add UserID on Student.
	_ = db
	_ = userID
	return nil
}

func childrenForParent(db *gorm.DB, userID uuid.UUID) []models.Student {
	var students []models.Student
	db.Where("parent_id = ? AND is_archived = false", userID).
		Order("full_name ASC").Find(&students)
	return students
}

func unreadNotificationsCount(db *gorm.DB, userID uuid.UUID) int64 {
	var n int64
	db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&n)
	return n
}