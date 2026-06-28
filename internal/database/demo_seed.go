package database

// DemoSeed populates the database with realistic demo data for all roles,
// classes, subjects, students, and approved scores so every portal feature
// can be tested without manual data entry.
//
// It is idempotent — checking a sentinel user before inserting anything.
// Call after Seed() in main.go startup.

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"school-platform/internal/models"
)

// fixed UUIDs so re-runs are idempotent
var (
	demoSchoolID     = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	demoSessionID    = uuid.MustParse("00000000-0000-0000-0000-000000000030")
	demoDivNursery   = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	demoDivPrimary   = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	demoDivSecondary = uuid.MustParse("00000000-0000-0000-0000-000000000023")

	// Staff — these IDs only matter for rows demo_seed itself creates.
	// If 02_seed_all_roles.sql already inserted these emails with different
	// IDs, the email-based upsert below handles it gracefully.
	demoPrincipalID = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	demoVpID        = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	demoHeadID      = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	demoAsstHeadID  = uuid.MustParse("00000000-0000-0000-0000-000000000014")
	demoBursarSecID = uuid.MustParse("00000000-0000-0000-0000-000000000015")
	demoBursarPriID = uuid.MustParse("00000000-0000-0000-0000-000000000016")
	demoTeacher1ID  = uuid.MustParse("00000000-0000-0000-0000-000000000017")
	demoTeacher2ID  = uuid.MustParse("00000000-0000-0000-0000-000000000018")
	demoTeacher3ID  = uuid.MustParse("00000000-0000-0000-0000-000000000019")

	// Parent / Student
	demoParent1ID = uuid.MustParse("00000000-0000-0000-0000-000000000050")
	demoParent2ID = uuid.MustParse("00000000-0000-0000-0000-000000000051")

	// Term
	demoTerm1ID = uuid.MustParse("00000000-0000-0000-0000-000000000031")

	// Classes
	demoClassJSS1A = uuid.MustParse("00000000-0000-0000-0000-000000000060")
	demoClassPri4A = uuid.MustParse("00000000-0000-0000-0000-000000000061")
	demoClassNur2  = uuid.MustParse("00000000-0000-0000-0000-000000000062")

	// Subjects (secondary)
	demoSubjMath   = uuid.MustParse("00000000-0000-0000-0000-000000000070")
	demoSubjEng    = uuid.MustParse("00000000-0000-0000-0000-000000000071")
	demoSubjSci    = uuid.MustParse("00000000-0000-0000-0000-000000000072")
	demoSubjSoc    = uuid.MustParse("00000000-0000-0000-0000-000000000073")
	demoSubjFrench = uuid.MustParse("00000000-0000-0000-0000-000000000074")

	// Subjects (primary)
	demoSubjPMath = uuid.MustParse("00000000-0000-0000-0000-000000000075")
	demoSubjPEng  = uuid.MustParse("00000000-0000-0000-0000-000000000076")
)

// DemoSeed runs only once (idempotent via email-based sentinel check).
func DemoSeed(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	// Sentinel: check by EMAIL, not ID.
	// 02_seed_all_roles.sql uses different UUIDs for the same emails,
	// so an ID-based check would miss them and cause email unique violations.
	var existing models.User
	if err := db.Where("email = ?", "principal@leaps.test").First(&existing).Error; err == nil {
		logSeed("demo seed already applied — skipping")
		return nil
	}

	logSeed("running demo seed...")

	pw := demoHash("Demo!2026")

	// ── Staff users ───────────────────────────────────────────────────────
	// resolveOrCreate: if a user with this email already exists (from SQL seed),
	// return their actual ID. Otherwise insert a new row with our fixed ID.
	resolve := func(fixedID uuid.UUID, email, fullName, phone string, role models.Role, scope models.DivisionScope) uuid.UUID {
		var u models.User
		if db.Where("email = ?", email).First(&u).Error == nil {
			return u.ID // already exists — use whichever ID the SQL seed gave it
		}
		u = models.User{
			Base: base(fixedID), SchoolID: demoSchoolID,
			FullName: fullName, Email: email, Phone: phone,
			PasswordHash: pw, Role: role, DivisionScope: scope,
			IsActive: true, IsVerified: true,
		}
		if err := db.Create(&u).Error; err != nil {
			logSeed("warn: could not create %s: %v", email, err)
			return fixedID
		}
		return fixedID
	}

	principalID := resolve(demoPrincipalID, "principal@leaps.test", "Adebayo Okonkwo", "+234-801-000-0001", models.RolePrincipal, models.DivisionSecondary)
	vpID := resolve(demoVpID, "vp@leaps.test", "Ngozi Adeyemi", "+234-801-000-0002", models.RoleVicePrincipal, models.DivisionSecondary)
	_ = vpID
	headID := resolve(demoHeadID, "headteacher@leaps.test", "Chukwuma Eze", "+234-801-000-0003", models.RoleHeadTeacher, models.DivisionPrimary)
	_ = headID
	resolve(demoAsstHeadID, "assthead@leaps.test", "Amina Garba", "+234-801-000-0004", models.RoleAsstHeadTeacher, models.DivisionPrimary)
	resolve(demoBursarSecID, "bursar.sec@leaps.test", "Emeka Nwosu", "+234-801-000-0005", models.RoleBursar, models.DivisionSecondary)
	resolve(demoBursarPriID, "bursar.pri@leaps.test", "Fatima Usman", "+234-801-000-0006", models.RoleBursar, models.DivisionPrimary)
	teacher1ID := resolve(demoTeacher1ID, "teacher1@leaps.test", "Olumide Fashola", "+234-801-000-0007", models.RoleTeacher, models.DivisionSecondary)
	teacher2ID := resolve(demoTeacher2ID, "teacher2@leaps.test", "Blessing Okeke", "+234-801-000-0008", models.RoleTeacher, models.DivisionSecondary)
	teacher3ID := resolve(demoTeacher3ID, "teacher3@leaps.test", "Yusuf Ibrahim", "+234-801-000-0009", models.RoleTeacher, models.DivisionPrimary)
	parent1ID := resolve(demoParent1ID, "parent1@leaps.test", "Mrs. Chioma Okafor", "+234-802-000-0001", models.RoleParent, models.DivisionSecondary)
	parent2ID := resolve(demoParent2ID, "parent2@leaps.test", "Mr. Tunde Bakare", "+234-802-000-0002", models.RoleParent, models.DivisionPrimary)

	logSeed("resolved/created demo staff accounts")

	// ── Term ──────────────────────────────────────────────────────────────
	now := time.Now().UTC()

	// Re-use existing active term if present (SQL seed may have created one)
	var activeTerm models.Term
	termID := demoTerm1ID
	if db.Where("session_id = ? AND is_active = true", demoSessionID).First(&activeTerm).Error == nil {
		termID = activeTerm.ID
		logSeed("re-using existing active term: %s (id=%s)", activeTerm.Name, termID)
	} else {
		term := models.Term{
			Base:               base(demoTerm1ID),
			SessionID:          demoSessionID,
			Name:               "First Term",
			StartDate:          time.Date(now.Year(), 9, 1, 0, 0, 0, 0, time.UTC),
			EndDate:            time.Date(now.Year(), 12, 15, 0, 0, 0, 0, time.UTC),
			NextResumptionDate: time.Date(now.Year()+1, 1, 8, 0, 0, 0, 0, time.UTC),
			IsActive:           true,
		}
		db.FirstOrCreate(&term, "id = ?", demoTerm1ID)
		termID = demoTerm1ID
	}

	// ── Classes ───────────────────────────────────────────────────────────
	ftID := teacher1ID
	ftPriID := teacher3ID
	classes := []models.Class{
		{Base: base(demoClassJSS1A), DivisionID: demoDivSecondary, Name: "JSS 1", Stream: "A", FormTeacherID: &ftID},
		{Base: base(demoClassPri4A), DivisionID: demoDivPrimary, Name: "Primary 4", Stream: "A", FormTeacherID: &ftPriID},
		{Base: base(demoClassNur2), DivisionID: demoDivNursery, Name: "Nursery 2"},
	}
	for _, c := range classes {
		db.FirstOrCreate(&c, "id = ?", c.ID)
	}

	// ── Subjects ──────────────────────────────────────────────────────────
	subjects := []models.Subject{
		{Base: base(demoSubjMath), DivisionID: demoDivSecondary, Name: "Mathematics", Code: "MTH"},
		{Base: base(demoSubjEng), DivisionID: demoDivSecondary, Name: "English Language", Code: "ENG"},
		{Base: base(demoSubjSci), DivisionID: demoDivSecondary, Name: "Basic Science", Code: "BSC"},
		{Base: base(demoSubjSoc), DivisionID: demoDivSecondary, Name: "Social Studies", Code: "SST"},
		{Base: base(demoSubjFrench), DivisionID: demoDivSecondary, Name: "French", Code: "FRN"},
		{Base: base(demoSubjPMath), DivisionID: demoDivPrimary, Name: "Mathematics", Code: "MTH"},
		{Base: base(demoSubjPEng), DivisionID: demoDivPrimary, Name: "English Language", Code: "ENG"},
	}
	for _, s := range subjects {
		db.FirstOrCreate(&s, "id = ?", s.ID)
	}

	// ── Teacher assignments ────────────────────────────────────────────────
	assignments := []models.TeacherAssignment{
		{Base: randBase(), TeacherID: teacher1ID, ClassID: demoClassJSS1A, SubjectID: demoSubjMath, SessionID: demoSessionID, TermID: termID},
		{Base: randBase(), TeacherID: teacher1ID, ClassID: demoClassJSS1A, SubjectID: demoSubjSci, SessionID: demoSessionID, TermID: termID},
		{Base: randBase(), TeacherID: teacher2ID, ClassID: demoClassJSS1A, SubjectID: demoSubjEng, SessionID: demoSessionID, TermID: termID},
		{Base: randBase(), TeacherID: teacher2ID, ClassID: demoClassJSS1A, SubjectID: demoSubjSoc, SessionID: demoSessionID, TermID: termID},
		{Base: randBase(), TeacherID: teacher2ID, ClassID: demoClassJSS1A, SubjectID: demoSubjFrench, SessionID: demoSessionID, TermID: termID},
		{Base: randBase(), TeacherID: teacher3ID, ClassID: demoClassPri4A, SubjectID: demoSubjPMath, SessionID: demoSessionID, TermID: termID},
		{Base: randBase(), TeacherID: teacher3ID, ClassID: demoClassPri4A, SubjectID: demoSubjPEng, SessionID: demoSessionID, TermID: termID},
	}
	for _, a := range assignments {
		var ex models.TeacherAssignment
		if db.Where("teacher_id = ? AND class_id = ? AND subject_id = ? AND term_id = ?",
			a.TeacherID, a.ClassID, a.SubjectID, a.TermID).First(&ex).Error != nil {
			db.Create(&a)
		}
	}

	// ── Score structures ───────────────────────────────────────────────────
	caComponents := models.ScoreComponentSlice{
		{Name: "Test 1", MaxMarks: 10},
		{Name: "Test 2", MaxMarks: 10},
		{Name: "Assignment", MaxMarks: 10},
	}
	secSubjects := []uuid.UUID{demoSubjMath, demoSubjEng, demoSubjSci, demoSubjSoc, demoSubjFrench}
	priSubjects := []uuid.UUID{demoSubjPMath, demoSubjPEng}

	for _, subj := range secSubjects {
		var ex models.ScoreStructure
		if db.Where("subject_id = ? AND class_id = ? AND term_id = ?", subj, demoClassJSS1A, termID).First(&ex).Error != nil {
			ss := models.ScoreStructure{Base: randBase(), SubjectID: subj, ClassID: demoClassJSS1A, SessionID: demoSessionID, TermID: termID, Components: caComponents, ExamMarks: 70, Total: 100}
			db.Create(&ss)
		}
	}
	for _, subj := range priSubjects {
		var ex models.ScoreStructure
		if db.Where("subject_id = ? AND class_id = ? AND term_id = ?", subj, demoClassPri4A, termID).First(&ex).Error != nil {
			ss := models.ScoreStructure{Base: randBase(), SubjectID: subj, ClassID: demoClassPri4A, SessionID: demoSessionID, TermID: termID, Components: caComponents, ExamMarks: 70, Total: 100}
			db.Create(&ss)
		}
	}

	// ── Students ──────────────────────────────────────────────────────────
	secStudents := demoStudentsSeed(db, demoDivSecondary, demoClassJSS1A, "LPA/SEC/2025", &parent1ID, termID, 10)
	priStudents := demoStudentsSeed(db, demoDivPrimary, demoClassPri4A, "LPA/PRI/2025", &parent2ID, termID, 8)

	// ── Score entries (APPROVED) ───────────────────────────────────────────
	approvedAt := time.Now().Add(-48 * time.Hour)
	approvedBy := principalID

	createScores := func(students []models.Student, classID uuid.UUID, subjectIDs []uuid.UUID, teacherID uuid.UUID) {
		r := rand.New(rand.NewSource(42))
		for _, student := range students {
			for _, subjectID := range subjectIDs {
				var ex models.ScoreEntry
				if db.Where("student_id = ? AND subject_id = ? AND term_id = ?", student.ID, subjectID, termID).First(&ex).Error == nil {
					continue
				}
				t1 := float64(6 + r.Intn(5))
				t2 := float64(5 + r.Intn(6))
				asgn := float64(6 + r.Intn(5))
				exam := float64(45 + r.Intn(26))
				total := t1 + t2 + asgn + exam
				entry := models.ScoreEntry{
					Base: randBase(), StudentID: student.ID, SubjectID: subjectID,
					ClassID: classID, SessionID: demoSessionID, TermID: termID,
					TeacherID: teacherID,
					Components: models.ScoreEntryComponentSlice{
						{Name: "Test 1", Score: t1},
						{Name: "Test 2", Score: t2},
						{Name: "Assignment", Score: asgn},
					},
					ExamScore: exam, Total: total,
					TeacherRemark: remarkForScore(total),
					Status:        models.ScoreStatusApproved,
					SubmittedAt:   &approvedAt, ApprovedAt: &approvedAt, ApprovedBy: &approvedBy,
				}
				db.Create(&entry)
			}
		}
	}
	createScores(secStudents, demoClassJSS1A, secSubjects, teacher1ID)
	createScores(priStudents, demoClassPri4A, priSubjects, teacher3ID)

	// ── Demo student portal account ────────────────────────────────────────
	if len(secStudents) > 0 {
		stuUserID := uuid.MustParse("00000000-0000-0000-0000-000000000052")
		var stuUser models.User
		if db.First(&stuUser, "id = ?", stuUserID).Error != nil {
			stuUser = models.User{
				Base: base(stuUserID), SchoolID: demoSchoolID,
				FullName: secStudents[0].FullName, Email: "student@leaps.test",
				Phone: "+234-803-000-0001", PasswordHash: pw,
				Role: models.RoleStudent, DivisionScope: models.DivisionSecondary,
				IsActive: true, IsVerified: true,
			}
			// Only create if email not already taken (SQL seed may have it)
			var existing models.User
			if db.Where("email = ?", "student@leaps.test").First(&existing).Error != nil {
				db.Create(&stuUser)
			}
		}
	}

	// ── Fee structure ─────────────────────────────────────────────────────
	var existingFee models.FeeStructure
	if db.Where("division_id = ? AND term_id = ?", demoDivSecondary, termID).First(&existingFee).Error != nil {
		feeCategories := models.JSONSlice{
			map[string]interface{}{"name": "Tuition Fee", "amount_by_class": map[string]interface{}{demoClassJSS1A.String(): 45000}},
			map[string]interface{}{"name": "PTA Levy", "amount_by_class": map[string]interface{}{demoClassJSS1A.String(): 5000}},
			map[string]interface{}{"name": "Development Levy", "amount_by_class": map[string]interface{}{demoClassJSS1A.String(): 10000}},
		}
		fee := models.FeeStructure{
			Base: randBase(), DivisionID: demoDivSecondary,
			SessionID: demoSessionID, TermID: termID, BursarID: demoBursarSecID,
			Categories: feeCategories,
		}
		db.Create(&fee)
	}

	logSeed("demo seed complete — staff/parent accounts use password: Demo!2026")
	logSeed("  principal@leaps.test / vp@leaps.test / headteacher@leaps.test")
	logSeed("  bursar.sec@leaps.test / teacher1@leaps.test / teacher2@leaps.test")
	logSeed("  parent1@leaps.test / parent2@leaps.test / student@leaps.test")
	return nil
}

// demoStudentsSeed creates n students idempotently and returns the list.
func demoStudentsSeed(db *gorm.DB, divID, classID uuid.UUID, prefix string, parentID *uuid.UUID, termID uuid.UUID, n int) []models.Student {
	firstNames := []string{"Tochukwu", "Fatima", "Emeka", "Amina", "Chidi", "Ngozi", "Seun", "Hauwa", "Kola", "Zainab", "David", "Blessing", "Ibrahim", "Grace", "Uche", "Aisha", "Damilola", "Nkechi", "Ahmed", "Chioma"}
	lastNames := []string{"Okafor", "Musa", "Adeleke", "Eze", "Bakare", "Nwachukwu", "Garba", "Adeyemi", "Obi", "Hassan"}
	r := rand.New(rand.NewSource(99))
	var students []models.Student
	for i := 0; i < n; i++ {
		admID := fmt.Sprintf("%s/%03d", prefix, i+1)
		var s models.Student
		if db.Where("admission_id = ?", admID).First(&s).Error == nil {
			students = append(students, s)
			continue
		}
		name := firstNames[r.Intn(len(firstNames))] + " " + lastNames[r.Intn(len(lastNames))]
		gender := "Male"
		if r.Intn(2) == 0 {
			gender = "Female"
		}
		dob := time.Date(2010+r.Intn(5), time.Month(1+r.Intn(12)), 1+r.Intn(28), 0, 0, 0, 0, time.UTC)
		s = models.Student{
			Base: randBase(), AdmissionID: admID, SchoolID: demoSchoolID,
			DivisionID: divID, FullName: name, DOB: dob, Gender: gender,
			ParentID: parentID, IsActive: true,
		}
		db.Create(&s)
		history := models.StudentClassHistory{
			Base: randBase(), StudentID: s.ID, ClassID: classID,
			SessionID: demoSessionID, TermID: termID,
		}
		db.Create(&history)
		students = append(students, s)
	}
	return students
}

func base(id uuid.UUID) models.Base {
	return models.Base{ID: id}
}

func randBase() models.Base {
	return models.Base{ID: uuid.New()}
}

func demoHash(password string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(h)
}

func remarkForScore(total float64) string {
	switch {
	case total >= 70:
		return "Excellent performance"
	case total >= 60:
		return "Very good work"
	case total >= 50:
		return "Good effort, keep it up"
	case total >= 40:
		return "Fair — needs improvement"
	default:
		return "Requires urgent attention"
	}
}
