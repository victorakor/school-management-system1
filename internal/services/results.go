package services

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/models"
	"school-platform/internal/utils"
)

// ResultService handles automated result calculation.
// ALL calculations are performed here — never manually by users.
type ResultService struct {
	db    *gorm.DB
	notif *NotificationService
}

// NewResultService creates a new ResultService.
func NewResultService(db *gorm.DB, notif *NotificationService) *ResultService {
	return &ResultService{db: db, notif: notif}
}

// CalculateResults calculates results for all students in a class for a given term.
// This is called automatically once ALL subject scores for the class are approved.
func (s *ResultService) CalculateResults(classID, sessionID, termID uuid.UUID) error {
	// 1. Get all approved score entries for this class/session/term
	var scoreEntries []models.ScoreEntry
	if err := s.db.Preload("Student").Preload("Subject").
		Where("class_id = ? AND session_id = ? AND term_id = ? AND status = ?",
			classID, sessionID, termID, models.ScoreStatusApproved).
		Find(&scoreEntries).Error; err != nil {
		return fmt.Errorf("results: failed to fetch score entries: %w", err)
	}

	if len(scoreEntries) == 0 {
		return fmt.Errorf("results: no approved scores found for this class/term")
	}

	// 2. Get grading scale for the school
	var classRecord models.Class
	if err := s.db.Preload("Division").First(&classRecord, "id = ?", classID).Error; err != nil {
		return fmt.Errorf("results: failed to fetch class: %w", err)
	}

	var gradeScales []models.GradeScale
	s.db.Where("school_id = ?", classRecord.Division.SchoolID).
		Order("min_score DESC").
		Find(&gradeScales)

	// 3. Group scores by student and subject
	scoreMap := make(map[subjectScoreKey]*models.ScoreEntry)
	studentIDs := make(map[uuid.UUID]bool)
	subjectIDs := make(map[uuid.UUID]bool)

	for i := range scoreEntries {
		key := subjectScoreKey{scoreEntries[i].StudentID, scoreEntries[i].SubjectID}
		scoreMap[key] = &scoreEntries[i]
		studentIDs[scoreEntries[i].StudentID] = true
		subjectIDs[scoreEntries[i].SubjectID] = true
	}

	// 4. For each subject, calculate class stats (highest, lowest, average, positions)
	type subjectStats struct {
		Scores   []float64
		Students []uuid.UUID
	}
	subjectStatsMap := make(map[uuid.UUID]*subjectStats)

	for key, entry := range scoreMap {
		if _, ok := subjectStatsMap[key.SubjectID]; !ok {
			subjectStatsMap[key.SubjectID] = &subjectStats{}
		}
		subjectStatsMap[key.SubjectID].Scores = append(subjectStatsMap[key.SubjectID].Scores, entry.Total)
		subjectStatsMap[key.SubjectID].Students = append(subjectStatsMap[key.SubjectID].Students, key.StudentID)
	}

	// 5. Calculate per-student overall totals for class position ranking
	type studentTotal struct {
		StudentID uuid.UUID
		Total     float64
	}
	var studentTotals []studentTotal

	for studentID := range studentIDs {
		var total float64
		for subjectID := range subjectIDs {
			key := subjectScoreKey{studentID, subjectID}
			if entry, ok := scoreMap[key]; ok {
				total += entry.Total
			}
		}
		studentTotals = append(studentTotals, studentTotal{studentID, total})
	}

	// Sort by total descending for class position
	sort.Slice(studentTotals, func(i, j int) bool {
		return studentTotals[i].Total > studentTotals[j].Total
	})

	classPositionMap := make(map[uuid.UUID]int)
	for i, st := range studentTotals {
		classPositionMap[st.StudentID] = i + 1
	}

	totalStudents := len(studentIDs)

	// 6. Get attendance for each student this term
	type attendanceRecord struct {
		Present int
		Total   int
	}
	attendanceMap := make(map[uuid.UUID]attendanceRecord)
	for studentID := range studentIDs {
		var present, total int64
		s.db.Model(&models.Attendance{}).
			Where("student_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
				studentID, classID, sessionID, termID).
			Count(&total)
		s.db.Model(&models.Attendance{}).
			Where("student_id = ? AND class_id = ? AND session_id = ? AND term_id = ? AND status = ?",
				studentID, classID, sessionID, termID, models.AttendancePresent).
			Count(&present)
		attendanceMap[studentID] = attendanceRecord{int(present), int(total)}
	}

	// 7. Create/update Result and ResultSubject records for each student
	for _, st := range studentTotals {
		studentID := st.StudentID
		numSubjects := len(subjectIDs)
		avg := 0.0
		if numSubjects > 0 {
			avg = utils.RoundFloat(st.Total/float64(numSubjects), 2)
		}

		// Count passed/failed subjects
		passed, failed := 0, 0
		for subjectID := range subjectIDs {
			key := subjectScoreKey{studentID, subjectID}
			if entry, ok := scoreMap[key]; ok {
				grade := s.calculateGrade(entry.Total, gradeScales)
				if s.isPassingGrade(grade, gradeScales) {
					passed++
				} else {
					failed++
				}
			}
		}

		att := attendanceMap[studentID]

		// Generate doc ref for result
		docRef := utils.GenerateDocRef("RES")

		// Upsert Result record
		var result models.Result
		s.db.Where("student_id = ? AND class_id = ? AND session_id = ? AND term_id = ?",
			studentID, classID, sessionID, termID).
			FirstOrInit(&result)

		result.StudentID = studentID
		result.ClassID = classID
		result.SessionID = sessionID
		result.TermID = termID
		result.TotalMarks = st.Total
		result.Average = avg
		result.ClassPosition = classPositionMap[studentID]
		result.TotalStudents = totalStudents
		result.SubjectsPassed = passed
		result.SubjectsFailed = failed
		result.AttendancePresent = att.Present
		result.AttendanceTotal = att.Total

		if result.DocRef == "" {
			result.DocRef = docRef
		}

		if err := s.db.Save(&result).Error; err != nil {
			return fmt.Errorf("results: failed to save result for student %s: %w", studentID, err)
		}

		// 8. Create ResultSubject records for each subject
		for subjectID := range subjectIDs {
			key := subjectScoreKey{studentID, subjectID}
			entry, ok := scoreMap[key]
			if !ok {
				continue
			}

			stats := subjectStatsMap[subjectID]
			highest, lowest, classAvg := calculateSubjectStats(stats.Scores)

			// Calculate subject position
			subjectPos := calculateSubjectPosition(studentID, stats.Students, scoreMap, subjectID)

			grade := s.calculateGrade(entry.Total, gradeScales)

			var rs models.ResultSubject
			s.db.Where("result_id = ? AND subject_id = ?", result.ID, subjectID).
				FirstOrInit(&rs)

			rs.ResultID = result.ID
			rs.SubjectID = subjectID
			rs.CATotal = entry.Total - entry.ExamScore
			rs.ExamScore = entry.ExamScore
			rs.Total = entry.Total
			rs.Grade = grade
			rs.SubjectPosition = subjectPos
			rs.HighestScore = highest
			rs.LowestScore = lowest
			rs.ClassAvg = utils.RoundFloat(classAvg, 2)
			rs.TeacherRemark = entry.TeacherRemark

			if err := s.db.Save(&rs).Error; err != nil {
				return fmt.Errorf("results: failed to save result subject: %w", err)
			}
		}
	}

	return nil
}

// calculateGrade returns the grade letter for a given score.
func (s *ResultService) calculateGrade(score float64, scales []models.GradeScale) string {
	for _, scale := range scales {
		if score >= scale.MinScore && score <= scale.MaxScore {
			return scale.Grade
		}
	}
	return "F"
}

// isPassingGrade returns true if the grade is a passing grade.
func (s *ResultService) isPassingGrade(grade string, scales []models.GradeScale) bool {
	for _, scale := range scales {
		if scale.Grade == grade {
			return scale.IsPassing
		}
	}
	return false
}

// calculateSubjectStats returns highest, lowest, and average scores for a subject.
func calculateSubjectStats(scores []float64) (highest, lowest, avg float64) {
	if len(scores) == 0 {
		return 0, 0, 0
	}
	highest = scores[0]
	lowest = scores[0]
	sum := 0.0
	for _, s := range scores {
		if s > highest {
			highest = s
		}
		if s < lowest {
			lowest = s
		}
		sum += s
	}
	avg = sum / float64(len(scores))
	return
}

type subjectScoreKey struct {
	StudentID uuid.UUID
	SubjectID uuid.UUID
}

// calculateSubjectPosition returns the student's position in the class for a subject.
func calculateSubjectPosition(studentID uuid.UUID, students []uuid.UUID, scoreMap map[subjectScoreKey]*models.ScoreEntry, subjectID uuid.UUID) int {
	type entry struct {
		StudentID uuid.UUID
		Score     float64
	}
	var entries []entry
	for _, sid := range students {
		key := subjectScoreKey{sid, subjectID}
		if e, ok := scoreMap[key]; ok {
			entries = append(entries, entry{sid, e.Total})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})
	for i, e := range entries {
		if e.StudentID == studentID {
			return i + 1
		}
	}
	return 0
}

// PublishResults marks results as published for a class/term.
func (s *ResultService) PublishResults(classID, sessionID, termID, publishedBy uuid.UUID) error {
	result := s.db.Model(&models.Result{}).
		Where("class_id = ? AND session_id = ? AND term_id = ?", classID, sessionID, termID).
		Updates(map[string]interface{}{
			"is_published": true,
			"published_by": publishedBy,
		})

	if result.Error != nil {
		return fmt.Errorf("results: failed to publish: %w", result.Error)
	}

	return nil
}
