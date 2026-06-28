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
	schoolID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID   = uuid.MustParse("00000000-0000-0000-0000-000000000030")
	divNursery  = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	divPrimary  = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	divSecondary = uuid.MustParse("00000000-0000-0000-0000-000000000023")

	// Staff
	principalID  = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	vpID         = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	headID       = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	asstHeadID   = uuid.MustParse("00000000-0000-0000-0000-000000000014")
	bursarSecID  = uuid.MustParse("00000000-0000-0000-0000-000000000015")
	bursarPriID  = uuid.MustParse("00000000-0000-0000-0000-000000000016")
	teacher1ID   = uuid.MustParse("00000000-0000-0000-0000-000000000017")
	teacher2ID   = uuid.MustParse("00000000-0000-0000-0000-000000000018")
	teacher3ID   = uuid.MustParse("00000000-0000-0000-0000-000000000019")

	// Parent / Student
	parent1ID  = uuid.MustParse("00000000-0000-0000-0000-000000000050")
	parent2ID  = uuid.MustParse("00000000-0000-0000-0000-000000000051")

	// Term
	term1ID = uuid.MustParse("00000000-0000-0000-0000-000000000031")

	// Classes
	classJSS1A  = uuid.MustParse("00000000-0000-0000-0000-000000000060")
	classPri4A  = uuid.MustParse("00000000-0000-0000-0000-000000000061")
	classNur2   = uuid.MustParse("00000000-0000-0000-0000-000000000062")

	// Subjects (secondary)
	subjMath   = uuid.MustParse("00000000-0000-0000-0000-000000000070")
	subjEng    = uuid.MustParse("00000000-0000-0000-0000-000000000071")
	subjSci    = uuid.MustParse("00000000-0000-0000-0000-000000000072")
	subjSoc    = uuid.MustParse("00000000-0000-0000-0000-000000000073")
	subjFrench = uuid.MustParse("00000000-0000-0000-0000-000000000074")

	// Subjects (primary)
	subjPMath = uuid.MustParse("00000000-0000-0000-0000-000000000075")
	subjPEng  = uuid.MustParse("00000000-0000-0000-0000-000000000076")

	// Sentinel: if principal already exists, skip entire demo seed
	demoSentinel = principalID
)

// DemoSeed runs only once (idempotent via demoSentinel check).
func DemoSeed(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	// Skip if already seeded
	var existing models.User
	if err := db.First(&existing, "id = ?", demoSentinel).Error; err == nil {
		return nil // already done
	}

	logSeed("running demo seed...")

	pw := demoHash("Demo!2026")

	// ── Staff users ───────────────────────────────────────────────────────
	staff := []models.User{
		{Base: base(principalID), SchoolID: schoolID, FullName: "Adebayo Okonkwo", Email: "principal@leaps.test", Phone: "+234-801-000-0001", PasswordHash: pw, Role: models.RolePrincipal, DivisionScope: models.DivisionSecondary, IsActive: true, IsVerified: true},
		{Base: base(vpID), SchoolID: schoolID, FullName: "Ngozi Adeyemi", Email: "vp@leaps.test", Phone: "+234-801-000-0002", PasswordHash: pw, Role: models.RoleVicePrincipal, DivisionScope: models.DivisionSecondary, IsActive: true, IsVerified: true},
		{Base: base(headID), SchoolID: schoolID, FullName: "Chukwuma Eze", Email: "headteacher@leaps.test", Phone: "+234-801-000-0003", PasswordHash: pw, Role: models.RoleHeadTeacher, DivisionScope: models.DivisionPrimary, IsActive: true, IsVerified: true},
		{Base: base(asstHeadID), SchoolID: schoolID, FullName: "Amina Garba", Email: "assthead@leaps.test", Phone: "+234-801-000-0004", PasswordHash: pw, Role: models.RoleAsstHeadTeacher, DivisionScope: models.DivisionPrimary, IsActive: true, IsVerified: true},
		{Base: base(bursarSecID), SchoolID: schoolID, FullName: "Emeka Nwosu", Email: "bursar.sec@leaps.test", Phone: "+234-801-000-0005", PasswordHash: pw, Role: models.RoleBursar, DivisionScope: models.DivisionSecondary, IsActive: true, IsVerified: true},
		{Base: base(bursarPriID), SchoolID: schoolID, FullName: "Fatima Usman", Email: "bursar.pri@leaps.test", Phone: "+234-801-000-0006", PasswordHash: pw, Role: models.RoleBursar, DivisionScope: models.DivisionPrimary, IsActive: true, IsVerified: true},
		{Base: base(teacher1ID), SchoolID: schoolID, FullName: "Olumide Fashola", Email: "teacher1@leaps.test", Phone: "+234-801-000-0007", PasswordHash: pw, Role: models.RoleTeacher, DivisionScope: models.DivisionSecondary, IsActive: true, IsVerified: true},
		{Base: base(teacher2ID), SchoolID: schoolID, FullName: "Blessing Okeke", Email: "teacher2@leaps.test", Phone: "+234-801-000-0008", PasswordHash: pw, Role: models.RoleTeacher, DivisionScope: models.DivisionSecondary, IsActive: true, IsVerified: true},
		{Base: base(teacher3ID), SchoolID: schoolID, FullName: "Yusuf Ibrahim", Email: "teacher3@leaps.test", Phone: "+234-801-000-0009", PasswordHash: pw, Role: models.RoleTeacher, DivisionScope: models.DivisionPrimary, IsActive: true, IsVerified: true},
		{Base: base(parent1ID), SchoolID: schoolID, FullName: "Mrs. Chioma Okafor", Email: "parent1@leaps.test", Phone: "+234-802-000-0001", PasswordHash: pw, Role: models.RoleParent, DivisionScope: models.DivisionSecondary, IsActive: true, IsVerified: true},
		{Base: base(parent2ID), SchoolID: schoolID, FullName: "Mr. Tunde Bakare", Email: "parent2@leaps.test", Phone: "+234-802-000-0002", PasswordHash: pw, Role: models.RoleParent, DivisionScope: models.DivisionPrimary, IsActive: true, IsVerified: true},
	}
	for _, u := range staff {
		if err := db.FirstOrCreate(&u, "id = ?", u.ID).Error; err != nil {
			logSeed("warn: staff %s: %v", u.Email, err)
		}
	}
	logSeed("created %d demo staff/parent accounts", len(staff))

	// ── Term ──────────────────────────────────────────────────────────────
	now := time.Now().UTC()
	term := models.Term{
		Base:               base(term1ID),
		SessionID:          sessionID,
		Name:               "First Term",
		StartDate:          time.Date(now.Year(), 9, 1, 0, 0, 0, 0, time.UTC),
		EndDate:            time.Date(now.Year(), 12, 15, 0, 0, 0, 0, time.UTC),
		NextResumptionDate: time.Date(now.Year()+1, 1, 8, 0, 0, 0, 0, time.UTC),
		IsActive:           true,
	}
	db.FirstOrCreate(&term, "id = ?", term1ID)

	// ── Classes ───────────────────────────────────────────────────────────
	ftID := teacher1ID
	ftPriID := teacher3ID
	classes := []models.Class{
		{Base: base(classJSS1A), DivisionID: divSecondary, Name: "JSS 1", Stream: "A", FormTeacherID: &ftID},
		{Base: base(classPri4A), DivisionID: divPrimary, Name: "Primary 4", Stream: "A", FormTeacherID: &ftPriID},
		{Base: base(classNur2), DivisionID: divNursery, Name: "Nursery 2"},
	}
	for _, c := range classes {
		db.FirstOrCreate(&c, "id = ?", c.ID)
	}

	// ── Subjects ──────────────────────────────────────────────────────────
	subjects := []models.Subject{
		{Base: base(subjMath), DivisionID: divSecondary, Name: "Mathematics", Code: "MTH"},
		{Base: base(subjEng), DivisionID: divSecondary, Name: "English Language", Code: "ENG"},
		{Base: base(subjSci), DivisionID: divSecondary, Name: "Basic Science", Code: "BSC"},
		{Base: base(subjSoc), DivisionID: divSecondary, Name: "Social Studies", Code: "SST"},
		{Base: base(subjFrench), DivisionID: divSecondary, Name: "French", Code: "FRN"},
		{Base: base(subjPMath), DivisionID: divPrimary, Name: "Mathematics", Code: "MTH"},
		{Base: base(subjPEng), DivisionID: divPrimary, Name: "English Language", Code: "ENG"},
	}
	for _, s := range subjects {
		db.FirstOrCreate(&s, "id = ?", s.ID)
	}

	// ── Teacher assignments ────────────────────────────────────────────────
	assignments := []models.TeacherAssignment{
		{Base: randBase(), TeacherID: teacher1ID, ClassID: classJSS1A, SubjectID: subjMath, SessionID: sessionID, TermID: term1ID},
		{Base: randBase(), TeacherID: teacher1ID, ClassID: classJSS1A, SubjectID: subjSci, SessionID: sessionID, TermID: term1ID},
		{Base: randBase(), TeacherID: teacher2ID, ClassID: classJSS1A, SubjectID: subjEng, SessionID: sessionID, TermID: term1ID},
		{Base: randBase(), TeacherID: teacher2ID, ClassID: classJSS1A, SubjectID: subjSoc, SessionID: sessionID, TermID: term1ID},
		{Base: randBase(), TeacherID: teacher2ID, ClassID: classJSS1A, SubjectID: subjFrench, SessionID: sessionID, TermID: term1ID},
		{Base: randBase(), TeacherID: teacher3ID, ClassID: classPri4A, SubjectID: subjPMath, SessionID: sessionID, TermID: term1ID},
		{Base: randBase(), TeacherID: teacher3ID, ClassID: classPri4A, SubjectID: subjPEng, SessionID: sessionID, TermID: term1ID},
	}
	for _, a := range assignments {
		// Check by teacher+class+subject to avoid dups
		var existing models.TeacherAssignment
		if db.Where("teacher_id = ? AND class_id = ? AND subject_id = ? AND term_id = ?",
			a.TeacherID, a.ClassID, a.SubjectID, a.TermID).First(&existing).Error != nil {
			db.Create(&a)
		}
	}

	// ── Score structures ───────────────────────────────────────────────────
	caComponents := models.ScoreComponentSlice{
		{Name: "Test 1", MaxMarks: 10},
		{Name: "Test 2", MaxMarks: 10},
		{Name: "Assignment", MaxMarks: 10},
	}
	secSubjects := []uuid.UUID{subjMath, subjEng, subjSci, subjSoc, subjFrench}
	priSubjects := []uuid.UUID{subjPMath, subjPEng}

	for _, subj := range secSubjects {
		var existing models.ScoreStructure
		if db.Where("subject_id = ? AND class_id = ? AND term_id = ?", subj, classJSS1A, term1ID).First(&existing).Error != nil {
			ss := models.ScoreStructure{Base: randBase(), SubjectID: subj, ClassID: classJSS1A, SessionID: sessionID, TermID: term1ID, Components: caComponents, ExamMarks: 70, Total: 100}
			db.Create(&ss)
		}
	}
	for _, subj := range priSubjects {
		var existing models.ScoreStructure
		if db.Where("subject_id = ? AND class_id = ? AND term_id = ?", subj, classPri4A, term1ID).First(&existing).Error != nil {
			ss := models.ScoreStructure{Base: randBase(), SubjectID: subj, ClassID: classPri4A, SessionID: sessionID, TermID: term1ID, Components: caComponents, ExamMarks: 70, Total: 100}
			db.Create(&ss)
		}
	}

	// ── Students ──────────────────────────────────────────────────────────
	secStudents := demoStudents(db, divSecondary, classJSS1A, "LPA/SEC/2025", &parent1ID, 10)
	priStudents := demoStudents(db, divPrimary, classPri4A, "LPA/PRI/2025", &parent2ID, 8)

	// ── Score entries (APPROVED) ───────────────────────────────────────────
	approvedAt := time.Now().Add(-48 * time.Hour)
	approvedBy := principalID

	createScores := func(students []models.Student, classID uuid.UUID, subjectIDs []uuid.UUID, teacherID uuid.UUID) {
		r := rand.New(rand.NewSource(42))
		for _, student := range students {
			for _, subjectID := range subjectIDs {
				var existing models.ScoreEntry
				if db.Where("student_id = ? AND subject_id = ? AND term_id = ?", student.ID, subjectID, term1ID).First(&existing).Error == nil {
					continue
				}
				t1 := float64(6 + r.Intn(5))   // 6-10
				t2 := float64(5 + r.Intn(6))   // 5-10
				asgn := float64(6 + r.Intn(5)) // 6-10
				exam := float64(45 + r.Intn(26)) // 45-70
				total := t1 + t2 + asgn + exam

				entry := models.ScoreEntry{
					Base:       randBase(),
					StudentID:  student.ID,
					SubjectID:  subjectID,
					ClassID:    classID,
					SessionID:  sessionID,
					TermID:     term1ID,
					TeacherID:  teacherID,
					Components: models.ScoreEntryComponentSlice{
						{Name: "Test 1", Score: t1},
						{Name: "Test 2", Score: t2},
						{Name: "Assignment", Score: asgn},
					},
					ExamScore:     exam,
					Total:         total,
					TeacherRemark: remarkForScore(total),
					Status:        models.ScoreStatusApproved,
					SubmittedAt:   &approvedAt,
					ApprovedAt:    &approvedAt,
					ApprovedBy:    &approvedBy,
				}
				db.Create(&entry)
			}
		}
	}

	createScores(secStudents, classJSS1A, secSubjects, teacher1ID)
	createScores(priStudents, classPri4A, priSubjects, teacher3ID)

	// ── Demo student portal accounts ───────────────────────────────────────
	// Link first secondary student as a STUDENT user for demo login
	if len(secStudents) > 0 {
		stuUserID := uuid.MustParse("00000000-0000-0000-0000-000000000052")
		var stuUser models.User
		if db.First(&stuUser, "id = ?", stuUserID).Error != nil {
			stuParentID := parent1ID
			stuUser = models.User{
				Base: base(stuUserID), SchoolID: schoolID,
				FullName: secStudents[0].FullName, Email: "student@leaps.test",
				Phone: "+234-803-000-0001", PasswordHash: pw,
				Role: models.RoleStudent, DivisionScope: models.DivisionSecondary,
				IsActive: true, IsVerified: true,
			}
			db.Create(&stuUser)
			// Link student record to this user
			db.Model(&secStudents[0]).Update("parent_id", stuParentID)
		}
	}

	// ── Fee structure ─────────────────────────────────────────────────────
	var existingFee models.FeeStructure
	if db.Where("division_id = ? AND term_id = ?", divSecondary, term1ID).First(&existingFee).Error != nil {
		feeCategories := models.JSONSlice{
			map[string]interface{}{"name": "Tuition Fee", "amount_by_class": map[string]interface{}{classJSS1A.String(): 45000}},
			map[string]interface{}{"name": "PTA Levy", "amount_by_class": map[string]interface{}{classJSS1A.String(): 5000}},
			map[string]interface{}{"name": "Development Levy", "amount_by_class": map[string]interface{}{classJSS1A.String(): 10000}},
		}
		fee := models.FeeStructure{
			Base:       randBase(),
			DivisionID: divSecondary,
			SessionID:  sessionID,
			TermID:     term1ID,
			BursarID:   bursarSecID,
			Categories: feeCategories,
		}
		db.Create(&fee)
	}

	logSeed("demo seed complete — all demo accounts use password: Demo!2026")
	logSeed("demo logins: principal@leaps.test / vp@leaps.test / headteacher@leaps.test")
	logSeed("             bursar.sec@leaps.test / teacher1@leaps.test / teacher2@leaps.test")
	logSeed("             parent1@leaps.test / parent2@leaps.test / student@leaps.test")
	return nil
}

// demoStudents creates n students idempotently, returns the list.
func demoStudents(db *gorm.DB, divID, classID uuid.UUID, prefix string, parentID *uuid.UUID, n int) []models.Student {
	firstNames := []string{"Tochukwu","Fatima","Emeka","Amina","Chidi","Ngozi","Seun","Hauwa","Kola","Zainab","David","Blessing","Ibrahim","Grace","Uche","Aisha","Damilola","Nkechi","Ahmed","Chioma"}
	lastNames  := []string{"Okafor","Musa","Adeleke","Eze","Bakare","Nwachukwu","Garba","Adeyemi","Obi","Hassan"}
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
			Base:        randBase(),
			AdmissionID: admID,
			SchoolID:    schoolID,
			DivisionID:  divID,
			FullName:    name,
			DOB:         dob,
			Gender:      gender,
			ParentID:    parentID,
			IsActive:    true,
		}
		db.Create(&s)
		// Enroll in class
		history := models.StudentClassHistory{
			Base:      randBase(),
			StudentID: s.ID,
			ClassID:   classID,
			SessionID: sessionID,
			TermID:    term1ID,
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
