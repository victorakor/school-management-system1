package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Custom JSON Types ─────────────────────────────────────────────────────────

// JSONMap is a generic JSON object stored as JSONB in PostgreSQL.
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONMap: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// JSONSlice is a generic JSON array stored as JSONB in PostgreSQL.
type JSONSlice []interface{}

func (j JSONSlice) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONSlice) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONSlice: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// ─── Enums ─────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleOwner           Role = "OWNER"
	RolePrincipal       Role = "PRINCIPAL"
	RoleVicePrincipal   Role = "VICE_PRINCIPAL"
	RoleHeadTeacher     Role = "HEAD_TEACHER"
	RoleAsstHeadTeacher Role = "ASST_HEAD_TEACHER"
	RoleBursar          Role = "BURSAR"
	RoleTeacher         Role = "TEACHER"
	RoleStudent         Role = "STUDENT"
	RolePupil           Role = "PUPIL"
	RoleParent          Role = "PARENT"
)

type DivisionScope string

const (
	DivisionNursery   DivisionScope = "NURSERY"
	DivisionPrimary   DivisionScope = "PRIMARY"
	DivisionSecondary DivisionScope = "SECONDARY"
	DivisionAll       DivisionScope = "ALL"
)

type ApplicationStatus string

const (
	AppStatusPending     ApplicationStatus = "PENDING"
	AppStatusUnderReview ApplicationStatus = "UNDER_REVIEW"
	AppStatusAccepted    ApplicationStatus = "ACCEPTED"
	AppStatusDeclined    ApplicationStatus = "DECLINED"
)

type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "PRESENT"
	AttendanceAbsent  AttendanceStatus = "ABSENT"
	AttendanceLate    AttendanceStatus = "LATE"
)

type ScoreStatus string

const (
	ScoreStatusDraft     ScoreStatus = "DRAFT"
	ScoreStatusSubmitted ScoreStatus = "SUBMITTED"
	ScoreStatusApproved  ScoreStatus = "APPROVED"
	ScoreStatusRejected  ScoreStatus = "REJECTED"
)

type QuizTargetScope string

const (
	QuizScopeClass    QuizTargetScope = "CLASS"
	QuizScopeDivision QuizTargetScope = "DIVISION"
	QuizScopeSchool   QuizTargetScope = "SCHOOL"
)

type QuizStatus string

const (
	QuizStatusDraft     QuizStatus = "DRAFT"
	QuizStatusActive    QuizStatus = "ACTIVE"
	QuizStatusClosed    QuizStatus = "CLOSED"
	QuizStatusPostponed QuizStatus = "POSTPONED"
)

type QuizQuestionType string

const (
	QuestionMCQ       QuizQuestionType = "MCQ"
	QuestionTrueFalse QuizQuestionType = "TRUE_FALSE"
	QuestionFillBlank QuizQuestionType = "FILL_BLANK"
)

type QuizDifficulty string

const (
	DifficultyEasy   QuizDifficulty = "EASY"
	DifficultyMedium QuizDifficulty = "MEDIUM"
	DifficultyHard   QuizDifficulty = "HARD"
)

type ResultReleaseMode string

const (
	ReleaseImmediate  ResultReleaseMode = "IMMEDIATE"
	ReleaseScheduled  ResultReleaseMode = "SCHEDULED"
)

type TimetableType string

const (
	TimetableUploadedXLSX TimetableType = "UPLOADED_XLSX"
	TimetableUploadedPDF  TimetableType = "UPLOADED_PDF"
	TimetableBuilder      TimetableType = "BUILDER"
)

type DocType string

const (
	DocTypeAppointmentLetter DocType = "APPOINTMENT_LETTER"
	DocTypeAdmissionLetter   DocType = "ADMISSION_LETTER"
	DocTypeReportCard        DocType = "REPORT_CARD"
	DocTypeFeeReceipt        DocType = "FEE_RECEIPT"
	DocTypeFinanceReport     DocType = "FINANCE_REPORT"
)

type MediaType string

const (
	MediaImage MediaType = "IMAGE"
	MediaVideo MediaType = "VIDEO"
)

// ─── Base Model ────────────────────────────────────────────────────────────────

// Base provides UUID primary key and timestamps for all models.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // GORM soft delete — never used for user data; use IsArchived instead
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// ─── School ────────────────────────────────────────────────────────────────────

type School struct {
	Base
	Name                       string `gorm:"not null" json:"name"`
	Motto                      string `json:"motto"`
	Address                    string `json:"address"`
	Phone                      string `json:"phone"`
	Email                      string `json:"email"`
	LogoURL                    string `json:"logo_url"`
	StampURL                   string `json:"stamp_url"`
	SignaturePrincipalURL       string `json:"signature_principal_url"`
	SignatureHeadteacherURL     string `json:"signature_headteacher_url"`
	PrimaryColor               string `gorm:"default:'#0F2557'" json:"primary_color"`
	Prefix                     string `gorm:"not null" json:"prefix"` // e.g. "GRA"
	WatermarkEnabled           bool   `gorm:"default:false" json:"watermark_enabled"`
	BursarName                 string `json:"bursar_name"`
	// Configurable content
	AdmissionDocumentsList     string `json:"admission_documents_list"` // newline-separated
	SchoolDirections           string `json:"school_directions"`
	FAQContent                 string `json:"faq_content"` // JSON array of {q, a}
	TestimonialsContent        string `json:"testimonials_content"` // JSON array
	VideoHeroURL               string `json:"video_hero_url"`
	MaxVideoUploadMB           int    `gorm:"default:100" json:"max_video_upload_mb"`
}

// ─── User ──────────────────────────────────────────────────────────────────────

type User struct {
	Base
	SchoolID      uuid.UUID     `gorm:"type:uuid;not null;index" json:"school_id"`
	School        School        `gorm:"foreignKey:SchoolID" json:"-"`
	FullName      string        `gorm:"not null" json:"full_name"`
	Email         string        `gorm:"uniqueIndex;not null" json:"email"`
	Phone         string        `json:"phone"`
	PasswordHash  string        `gorm:"not null" json:"-"`
	Role          Role          `gorm:"type:varchar(30);not null;index" json:"role"`
	DivisionScope DivisionScope `gorm:"type:varchar(20);not null;default:'ALL'" json:"division_scope"`
	IsActive      bool          `gorm:"default:true" json:"is_active"`
	IsArchived    bool          `gorm:"default:false" json:"is_archived"`
	IsVerified    bool          `gorm:"default:false" json:"is_verified"` // email verified
	LastLogin     *time.Time    `json:"last_login,omitempty"`
	AvatarURL     string        `json:"avatar_url"`
	// OTP fields (for parent email verification)
	OTPCode      string     `gorm:"column:otp_code" json:"-"`
	OTPExpiresAt *time.Time `gorm:"column:otp_expires_at" json:"-"`
}

// ─── Division ──────────────────────────────────────────────────────────────────

type Division struct {
	Base
	SchoolID uuid.UUID     `gorm:"type:uuid;not null;index" json:"school_id"`
	School   School        `gorm:"foreignKey:SchoolID" json:"-"`
	Name     DivisionScope `gorm:"type:varchar(20);not null" json:"name"`
}

// ─── Academic Session ──────────────────────────────────────────────────────────

type AcademicSession struct {
	Base
	SchoolID   uuid.UUID `gorm:"type:uuid;not null;index" json:"school_id"`
	School     School    `gorm:"foreignKey:SchoolID" json:"-"`
	Name       string    `gorm:"not null" json:"name"` // e.g. "2025/2026"
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	IsActive   bool      `gorm:"default:false;index" json:"is_active"`
	IsArchived bool      `gorm:"default:false" json:"is_archived"`
}

// ─── Term ──────────────────────────────────────────────────────────────────────

type Term struct {
	Base
	SessionID           uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	Session             AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	Name                string    `gorm:"not null" json:"name"` // e.g. "First Term"
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	NextResumptionDate  time.Time `json:"next_resumption_date"`
	IsActive            bool      `gorm:"default:false;index" json:"is_active"`
}

// ─── Class ─────────────────────────────────────────────────────────────────────

type Class struct {
	Base
	DivisionID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"division_id"`
	Division      Division   `gorm:"foreignKey:DivisionID" json:"-"`
	Name          string     `gorm:"not null" json:"name"` // e.g. "JSS 2A"
	Stream        string     `json:"stream"`               // e.g. "A", "B", "Science"
	FormTeacherID *uuid.UUID `gorm:"type:uuid;index" json:"form_teacher_id,omitempty"`
	FormTeacher   *User      `gorm:"foreignKey:FormTeacherID" json:"form_teacher,omitempty"`
}

// ─── Subject ───────────────────────────────────────────────────────────────────

type Subject struct {
	Base
	DivisionID uuid.UUID `gorm:"type:uuid;not null;index" json:"division_id"`
	Division   Division  `gorm:"foreignKey:DivisionID" json:"-"`
	Name       string    `gorm:"not null" json:"name"`
	Code       string    `gorm:"not null" json:"code"` // e.g. "MTH"
	IsArchived bool      `gorm:"default:false" json:"is_archived"`
}

// ─── Teacher Assignment ────────────────────────────────────────────────────────

type TeacherAssignment struct {
	Base
	TeacherID uuid.UUID `gorm:"type:uuid;not null;index" json:"teacher_id"`
	Teacher   User      `gorm:"foreignKey:TeacherID" json:"-"`
	ClassID   uuid.UUID `gorm:"type:uuid;not null;index" json:"class_id"`
	Class     Class     `gorm:"foreignKey:ClassID" json:"-"`
	SubjectID uuid.UUID `gorm:"type:uuid;not null;index" json:"subject_id"`
	Subject   Subject   `gorm:"foreignKey:SubjectID" json:"-"`
	SessionID uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	Session   AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	TermID    uuid.UUID `gorm:"type:uuid;not null;index" json:"term_id"`
	Term      Term      `gorm:"foreignKey:TermID" json:"-"`
}

// ─── Student ───────────────────────────────────────────────────────────────────

type Student struct {
	Base
	AdmissionID string     `gorm:"uniqueIndex;not null" json:"admission_id"`
	SchoolID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"school_id"`
	School      School     `gorm:"foreignKey:SchoolID" json:"-"`
	DivisionID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"division_id"`
	Division    Division   `gorm:"foreignKey:DivisionID" json:"-"`
	FullName    string     `gorm:"not null" json:"full_name"`
	DOB         time.Time  `json:"dob"`
	Gender      string     `gorm:"type:varchar(10)" json:"gender"`
	PhotoURL    string     `json:"photo_url"`
	ParentID    *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Parent      *User      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	IsAlumni    bool       `gorm:"default:false" json:"is_alumni"`
	IsArchived  bool       `gorm:"default:false" json:"is_archived"`
}

// ─── Student Class History ─────────────────────────────────────────────────────

type StudentClassHistory struct {
	Base
	StudentID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
	Student         Student    `gorm:"foreignKey:StudentID" json:"-"`
	ClassID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"class_id"`
	Class           Class      `gorm:"foreignKey:ClassID" json:"-"`
	SessionID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"session_id"`
	Session         AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	TermID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"term_id"`
	Term            Term       `gorm:"foreignKey:TermID" json:"-"`
	PromotedBy      *uuid.UUID `gorm:"type:uuid;index" json:"promoted_by,omitempty"`
	Promoter        *User      `gorm:"foreignKey:PromotedBy" json:"-"`
	PromotionDate   *time.Time `json:"promotion_date,omitempty"`
	RetentionReason string     `json:"retention_reason,omitempty"`
}

// ─── Application ───────────────────────────────────────────────────────────────

type Application struct {
	Base
	SchoolID             uuid.UUID         `gorm:"type:uuid;not null;index" json:"school_id"`
	School               School            `gorm:"foreignKey:SchoolID" json:"-"`
	ParentID             uuid.UUID         `gorm:"type:uuid;not null;index" json:"parent_id"`
	Parent               User              `gorm:"foreignKey:ParentID" json:"-"`
	ChildName            string            `gorm:"not null" json:"child_name"`
	ChildDOB             time.Time         `json:"child_dob"`
	ChildGender          string            `gorm:"type:varchar(10)" json:"child_gender"`
	Division             DivisionScope     `gorm:"type:varchar(20);not null" json:"division"`
	PassportURL          string            `json:"passport_url"`
	BirthCertURL         string            `json:"birth_cert_url"`
	PrevSchool           string            `json:"prev_school"`
	PrevClass            string            `json:"prev_class"`
	PrevReportURL        string            `json:"prev_report_url"`
	EmergencyContactName  string           `json:"emergency_contact_name"`
	EmergencyContactPhone string           `json:"emergency_contact_phone"`
	HomeAddress          string            `json:"home_address"`
	MedicalConditions    string            `json:"medical_conditions"`
	Status               ApplicationStatus `gorm:"type:varchar(20);default:'PENDING';index" json:"status"`
	RefNumber            string            `gorm:"uniqueIndex;not null" json:"ref_number"`
	AppointmentDate      *time.Time        `json:"appointment_date,omitempty"`
	AppointmentTime      string            `json:"appointment_time,omitempty"`
	RescheduleCount      int               `gorm:"default:0" json:"reschedule_count"`
	DeclineReason        string            `json:"decline_reason,omitempty"`
	IsArchived           bool              `gorm:"default:false" json:"is_archived"`
}

// ─── Appointment Letter ────────────────────────────────────────────────────────

type AppointmentLetter struct {
	Base
	ApplicationID uuid.UUID `gorm:"type:uuid;not null;index" json:"application_id"`
	Application   Application `gorm:"foreignKey:ApplicationID" json:"-"`
	DocRef        string    `gorm:"uniqueIndex;not null" json:"doc_ref"`
	QRURL         string    `json:"qr_url"`
	FileURL       string    `json:"file_url"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// ─── Admission Letter ──────────────────────────────────────────────────────────

type AdmissionLetter struct {
	Base
	ApplicationID uuid.UUID `gorm:"type:uuid;not null;index" json:"application_id"`
	Application   Application `gorm:"foreignKey:ApplicationID" json:"-"`
	UploadedBy    uuid.UUID `gorm:"type:uuid;not null;index" json:"uploaded_by"`
	Uploader      User      `gorm:"foreignKey:UploadedBy" json:"-"`
	FileURL       string    `json:"file_url"`
	DocRef        string    `gorm:"uniqueIndex;not null" json:"doc_ref"`
	QRURL         string    `json:"qr_url"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// ─── Attendance ────────────────────────────────────────────────────────────────

type Attendance struct {
	Base
	StudentID uuid.UUID        `gorm:"type:uuid;not null;index" json:"student_id"`
	Student   Student          `gorm:"foreignKey:StudentID" json:"-"`
	ClassID   uuid.UUID        `gorm:"type:uuid;not null;index" json:"class_id"`
	Class     Class            `gorm:"foreignKey:ClassID" json:"-"`
	SubjectID *uuid.UUID       `gorm:"type:uuid;index" json:"subject_id,omitempty"`
	Subject   *Subject         `gorm:"foreignKey:SubjectID" json:"-"`
	SessionID uuid.UUID        `gorm:"type:uuid;not null;index" json:"session_id"`
	Session   AcademicSession  `gorm:"foreignKey:SessionID" json:"-"`
	TermID    uuid.UUID        `gorm:"type:uuid;not null;index" json:"term_id"`
	Term      Term             `gorm:"foreignKey:TermID" json:"-"`
	TeacherID uuid.UUID        `gorm:"type:uuid;not null;index" json:"teacher_id"`
	Teacher   User             `gorm:"foreignKey:TeacherID" json:"-"`
	Date      time.Time        `gorm:"index" json:"date"`
	Status    AttendanceStatus `gorm:"type:varchar(10);not null" json:"status"`
}

// ─── Score Structure ───────────────────────────────────────────────────────────

// ScoreComponent represents a single CA component (e.g. Test 1 = 10 marks).
type ScoreComponent struct {
	Name     string `json:"name"`
	MaxMarks int    `json:"max_marks"`
}

type ScoreComponentSlice []ScoreComponent

func (s ScoreComponentSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *ScoreComponentSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("ScoreComponentSlice: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, s)
}

type ScoreStructure struct {
	Base
	SubjectID  uuid.UUID           `gorm:"type:uuid;not null;index" json:"subject_id"`
	Subject    Subject             `gorm:"foreignKey:SubjectID" json:"-"`
	ClassID    uuid.UUID           `gorm:"type:uuid;not null;index" json:"class_id"`
	Class      Class               `gorm:"foreignKey:ClassID" json:"-"`
	SessionID  uuid.UUID           `gorm:"type:uuid;not null;index" json:"session_id"`
	Session    AcademicSession     `gorm:"foreignKey:SessionID" json:"-"`
	TermID     uuid.UUID           `gorm:"type:uuid;not null;index" json:"term_id"`
	Term       Term                `gorm:"foreignKey:TermID" json:"-"`
	Components ScoreComponentSlice `gorm:"type:jsonb" json:"components"`
	ExamMarks  int                 `gorm:"not null" json:"exam_marks"`
	Total      int                 `gorm:"not null;default:100" json:"total"` // always 100
}

// ─── Score Entry ───────────────────────────────────────────────────────────────

// ScoreEntryComponent represents a student's score for a single CA component.
type ScoreEntryComponent struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

type ScoreEntryComponentSlice []ScoreEntryComponent

func (s ScoreEntryComponentSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *ScoreEntryComponentSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("ScoreEntryComponentSlice: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, s)
}

type ScoreEntry struct {
	Base
	StudentID        uuid.UUID                `gorm:"type:uuid;not null;index" json:"student_id"`
	Student          Student                  `gorm:"foreignKey:StudentID" json:"-"`
	SubjectID        uuid.UUID                `gorm:"type:uuid;not null;index" json:"subject_id"`
	Subject          Subject                  `gorm:"foreignKey:SubjectID" json:"-"`
	ClassID          uuid.UUID                `gorm:"type:uuid;not null;index" json:"class_id"`
	Class            Class                    `gorm:"foreignKey:ClassID" json:"-"`
	SessionID        uuid.UUID                `gorm:"type:uuid;not null;index" json:"session_id"`
	Session          AcademicSession          `gorm:"foreignKey:SessionID" json:"-"`
	TermID           uuid.UUID                `gorm:"type:uuid;not null;index" json:"term_id"`
	Term             Term                     `gorm:"foreignKey:TermID" json:"-"`
	TeacherID        uuid.UUID                `gorm:"type:uuid;not null;index" json:"teacher_id"`
	Teacher          User                     `gorm:"foreignKey:TeacherID" json:"-"`
	Components       ScoreEntryComponentSlice `gorm:"type:jsonb" json:"components"`
	ExamScore        float64                  `json:"exam_score"`
	Total            float64                  `json:"total"`
	TeacherRemark    string                   `json:"teacher_remark"`
	Status           ScoreStatus              `gorm:"type:varchar(20);default:'DRAFT';index" json:"status"`
	SubmittedAt      *time.Time               `json:"submitted_at,omitempty"`
	ApprovedAt       *time.Time               `json:"approved_at,omitempty"`
	ApprovedBy       *uuid.UUID               `gorm:"type:uuid;index" json:"approved_by,omitempty"`
	Approver         *User                    `gorm:"foreignKey:ApprovedBy" json:"-"`
	RejectionReason  string                   `json:"rejection_reason,omitempty"`
	UnlockRequested  bool                     `gorm:"default:false" json:"unlock_requested"`
	UnlockReason     string                   `json:"unlock_reason,omitempty"`
}

// ─── Audit Log ─────────────────────────────────────────────────────────────────

type AuditLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID" json:"-"`
	Action     string    `gorm:"not null" json:"action"`
	EntityType string    `gorm:"index" json:"entity_type"`
	EntityID   string    `gorm:"index" json:"entity_id"`
	IPAddress  string    `json:"ip_address"`
	Metadata   JSONMap   `gorm:"type:jsonb" json:"metadata"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// ─── Result ────────────────────────────────────────────────────────────────────

type Result struct {
	Base
	StudentID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"student_id"`
	Student            Student    `gorm:"foreignKey:StudentID" json:"-"`
	ClassID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"class_id"`
	Class              Class      `gorm:"foreignKey:ClassID" json:"-"`
	SessionID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"session_id"`
	Session            AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	TermID             uuid.UUID  `gorm:"type:uuid;not null;index" json:"term_id"`
	Term               Term       `gorm:"foreignKey:TermID" json:"-"`
	TotalMarks         float64    `json:"total_marks"`
	Average            float64    `json:"average"`
	ClassPosition      int        `json:"class_position"`
	TotalStudents      int        `json:"total_students"`
	SubjectsPassed     int        `json:"subjects_passed"`
	SubjectsFailed     int        `json:"subjects_failed"`
	AttendancePresent  int        `json:"attendance_present"`
	AttendanceTotal    int        `json:"attendance_total"`
	ClassTeacherRemark string     `json:"class_teacher_remark"`
	AdminRemark        string     `json:"admin_remark"`
	IsPublished        bool       `gorm:"default:false;index" json:"is_published"`
	PublishedAt        *time.Time `json:"published_at,omitempty"`
	PublishedBy        *uuid.UUID `gorm:"type:uuid;index" json:"published_by,omitempty"`
	Publisher          *User      `gorm:"foreignKey:PublishedBy" json:"-"`
	DocRef             string     `gorm:"uniqueIndex" json:"doc_ref"`
	QRURL              string     `json:"qr_url"`
	GeneratedAt        *time.Time `json:"generated_at,omitempty"`
}

// ─── Result Subject ────────────────────────────────────────────────────────────

type ResultSubject struct {
	Base
	ResultID        uuid.UUID `gorm:"type:uuid;not null;index" json:"result_id"`
	Result          Result    `gorm:"foreignKey:ResultID" json:"-"`
	SubjectID       uuid.UUID `gorm:"type:uuid;not null;index" json:"subject_id"`
	Subject         Subject   `gorm:"foreignKey:SubjectID" json:"-"`
	CATotal         float64   `json:"ca_total"`
	ExamScore       float64   `json:"exam_score"`
	Total           float64   `json:"total"`
	Grade           string    `json:"grade"`
	SubjectPosition int       `json:"subject_position"`
	HighestScore    float64   `json:"highest_score"`
	LowestScore     float64   `json:"lowest_score"`
	ClassAvg        float64   `json:"class_avg"`
	TeacherRemark   string    `json:"teacher_remark"`
}

// ─── Quiz ──────────────────────────────────────────────────────────────────────

type Quiz struct {
	Base
	SchoolID           uuid.UUID       `gorm:"type:uuid;not null;index" json:"school_id"`
	School             School          `gorm:"foreignKey:SchoolID" json:"-"`
	Title              string          `gorm:"not null" json:"title"`
	CreatedBy          uuid.UUID       `gorm:"type:uuid;not null;index" json:"created_by"`
	Creator            User            `gorm:"foreignKey:CreatedBy" json:"-"`
	TargetScope        QuizTargetScope `gorm:"type:varchar(20);not null" json:"target_scope"`
	TargetIDs          JSONSlice       `gorm:"type:jsonb" json:"target_ids"` // class/division IDs
	SessionID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"session_id"`
	Session            AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	StartTime          time.Time       `json:"start_time"`
	EndTime            time.Time       `json:"end_time"`
	DurationMinutes    int             `gorm:"not null" json:"duration_minutes"`
	QuestionCount      int             `gorm:"not null" json:"question_count"`
	SubmissionDeadline time.Time       `json:"submission_deadline"`
	MinThreshold       int             `gorm:"default:10" json:"min_threshold"`
	ResultReleaseMode  ResultReleaseMode `gorm:"type:varchar(20);default:'IMMEDIATE'" json:"result_release_mode"`
	ResultReleaseTime  *time.Time      `json:"result_release_time,omitempty"`
	TabSwitchLimit     int             `gorm:"default:3" json:"tab_switch_limit"`
	Status             QuizStatus      `gorm:"type:varchar(20);default:'DRAFT';index" json:"status"`
	IsArchived         bool            `gorm:"default:false" json:"is_archived"`
}

// ─── Quiz Question ─────────────────────────────────────────────────────────────

type QuizQuestion struct {
	Base
	SchoolID      uuid.UUID        `gorm:"type:uuid;not null;index" json:"school_id"`
	School        School           `gorm:"foreignKey:SchoolID" json:"-"`
	SubjectID     uuid.UUID        `gorm:"type:uuid;not null;index" json:"subject_id"`
	Subject       Subject          `gorm:"foreignKey:SubjectID" json:"-"`
	Division      DivisionScope    `gorm:"type:varchar(20);not null;index" json:"division"`
	Text          string           `gorm:"not null" json:"text"`
	Type          QuizQuestionType `gorm:"type:varchar(20);not null" json:"type"`
	Options       JSONSlice        `gorm:"type:jsonb" json:"options"` // [{label, value}]
	CorrectAnswer string           `gorm:"not null" json:"correct_answer"`
	Difficulty    QuizDifficulty   `gorm:"type:varchar(10);not null" json:"difficulty"`
	ContributorID uuid.UUID        `gorm:"type:uuid;not null;index" json:"contributor_id"`
	Contributor   User             `gorm:"foreignKey:ContributorID" json:"-"`
	IsApproved    bool             `gorm:"default:false;index" json:"is_approved"`
	IsFlagged     bool             `gorm:"default:false" json:"is_flagged"`
}

// ─── Quiz Attempt ──────────────────────────────────────────────────────────────

// QuizViolation records a single anti-cheat violation event.
type QuizViolation struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

type QuizViolationSlice []QuizViolation

func (q QuizViolationSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(q)
	return string(b), err
}

func (q *QuizViolationSlice) Scan(value interface{}) error {
	if value == nil {
		*q = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("QuizViolationSlice: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, q)
}

type QuizAttempt struct {
	Base
	QuizID       uuid.UUID          `gorm:"type:uuid;not null;index" json:"quiz_id"`
	Quiz         Quiz               `gorm:"foreignKey:QuizID" json:"-"`
	StudentID    uuid.UUID          `gorm:"type:uuid;not null;index" json:"student_id"`
	Student      Student            `gorm:"foreignKey:StudentID" json:"-"`
	Questions    JSONSlice          `gorm:"type:jsonb" json:"questions"` // randomised question set
	Answers      JSONSlice          `gorm:"type:jsonb" json:"answers"`   // student's answers
	Score        int                `json:"score"`
	StartedAt    time.Time          `json:"started_at"`
	SubmittedAt  *time.Time         `json:"submitted_at,omitempty"`
	Violations   QuizViolationSlice `gorm:"type:jsonb" json:"violations"`
	IsFlagged    bool               `gorm:"default:false;index" json:"is_flagged"`
	AutoSubmitted bool              `gorm:"default:false" json:"auto_submitted"`
}

// ─── Fee Structure ─────────────────────────────────────────────────────────────

type FeeStructure struct {
	Base
	DivisionID uuid.UUID `gorm:"type:uuid;not null;index" json:"division_id"`
	Division   Division  `gorm:"foreignKey:DivisionID" json:"-"`
	SessionID  uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	Session    AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	TermID     uuid.UUID `gorm:"type:uuid;not null;index" json:"term_id"`
	Term       Term      `gorm:"foreignKey:TermID" json:"-"`
	BursarID   uuid.UUID `gorm:"type:uuid;not null;index" json:"bursar_id"`
	Bursar     User      `gorm:"foreignKey:BursarID" json:"-"`
	Categories JSONSlice `gorm:"type:jsonb" json:"categories"` // [{name, amount_by_class: {class_id: amount}}]
}

// ─── Fee Payment ───────────────────────────────────────────────────────────────

type FeePayment struct {
	Base
	StudentID      uuid.UUID `gorm:"type:uuid;not null;index" json:"student_id"`
	Student        Student   `gorm:"foreignKey:StudentID" json:"-"`
	FeeStructureID uuid.UUID `gorm:"type:uuid;not null;index" json:"fee_structure_id"`
	FeeStructure   FeeStructure `gorm:"foreignKey:FeeStructureID" json:"-"`
	CategoryName   string    `gorm:"not null" json:"category_name"`
	AmountOwed     float64   `gorm:"not null" json:"amount_owed"`
	AmountPaid     float64   `gorm:"not null" json:"amount_paid"`
	PaymentDate    time.Time `gorm:"index" json:"payment_date"`
	RecordedBy     uuid.UUID `gorm:"type:uuid;not null;index" json:"recorded_by"`
	Recorder       User      `gorm:"foreignKey:RecordedBy" json:"-"`
	ReceiptRef     string    `gorm:"uniqueIndex;not null" json:"receipt_ref"`
	BalanceAfter   float64   `json:"balance_after"`
	DocRef         string    `gorm:"uniqueIndex;not null" json:"doc_ref"`
	QRURL          string    `json:"qr_url"`
	Notes          string    `json:"notes"`
	IsArchived     bool      `gorm:"default:false" json:"is_archived"`
}

// ─── Fee Discount ──────────────────────────────────────────────────────────────

type FeeDiscount struct {
	Base
	StudentID      uuid.UUID `gorm:"type:uuid;not null;index" json:"student_id"`
	Student        Student   `gorm:"foreignKey:StudentID" json:"-"`
	FeeStructureID uuid.UUID `gorm:"type:uuid;not null;index" json:"fee_structure_id"`
	FeeStructure   FeeStructure `gorm:"foreignKey:FeeStructureID" json:"-"`
	Percentage     float64   `gorm:"not null" json:"percentage"`
	Reason         string    `json:"reason"`
	AppliedBy      uuid.UUID `gorm:"type:uuid;not null;index" json:"applied_by"`
	Applier        User      `gorm:"foreignKey:AppliedBy" json:"-"`
}

// ─── Timetable ─────────────────────────────────────────────────────────────────

type Timetable struct {
	Base
	SchoolID      uuid.UUID     `gorm:"type:uuid;not null;index" json:"school_id"`
	School        School        `gorm:"foreignKey:SchoolID" json:"-"`
	DivisionID    uuid.UUID     `gorm:"type:uuid;not null;index" json:"division_id"`
	Division      Division      `gorm:"foreignKey:DivisionID" json:"-"`
	ClassID       *uuid.UUID    `gorm:"type:uuid;index" json:"class_id,omitempty"` // nil = division-level
	Class         *Class        `gorm:"foreignKey:ClassID" json:"-"`
	SessionID     uuid.UUID     `gorm:"type:uuid;not null;index" json:"session_id"`
	Session       AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	TermID        uuid.UUID     `gorm:"type:uuid;not null;index" json:"term_id"`
	Term          Term          `gorm:"foreignKey:TermID" json:"-"`
	Data          JSONMap       `gorm:"type:jsonb" json:"data"` // structured timetable JSON
	Version       int           `gorm:"default:1" json:"version"`
	Type          TimetableType `gorm:"type:varchar(20);not null" json:"type"`
	EffectiveFrom time.Time     `json:"effective_from"`
	EffectiveTo   *time.Time    `json:"effective_to,omitempty"`
	IsCurrent     bool          `gorm:"default:true;index" json:"is_current"`
	CreatedBy     uuid.UUID     `gorm:"type:uuid;not null;index" json:"created_by"`
	Creator       User          `gorm:"foreignKey:CreatedBy" json:"-"`
}

// ─── Activity Post ─────────────────────────────────────────────────────────────

// MediaItem represents a single media file in a post.
type MediaItem struct {
	URL  string    `json:"url"`
	Type MediaType `json:"type"`
}

type MediaItemSlice []MediaItem

func (m MediaItemSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

func (m *MediaItemSlice) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("MediaItemSlice: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, m)
}

type ActivityPost struct {
	Base
	SchoolID    uuid.UUID     `gorm:"type:uuid;not null;index" json:"school_id"`
	School      School        `gorm:"foreignKey:SchoolID" json:"-"`
	PostedBy    uuid.UUID     `gorm:"type:uuid;not null;index" json:"posted_by"`
	Poster      User          `gorm:"foreignKey:PostedBy" json:"-"`
	Caption     string        `json:"caption"`
	MediaURLs   MediaItemSlice `gorm:"type:jsonb" json:"media_urls"`
	DivisionTag DivisionScope `gorm:"type:varchar(20);not null" json:"division_tag"`
	IsPublished bool          `gorm:"default:false;index" json:"is_published"`
	IsArchived  bool          `gorm:"default:false" json:"is_archived"`
}

// ─── Notification ──────────────────────────────────────────────────────────────

type Notification struct {
	Base
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Title     string    `gorm:"not null" json:"title"`
	Body      string    `json:"body"`
	Type      string    `gorm:"index" json:"type"` // e.g. "ADMISSION", "RESULT", "QUIZ", "PAYMENT"
	EntityID  string    `json:"entity_id,omitempty"` // related entity UUID
	IsRead    bool      `gorm:"default:false;index" json:"is_read"`
}

// ─── Announcement ──────────────────────────────────────────────────────────────

type Announcement struct {
	Base
	SchoolID       uuid.UUID     `gorm:"type:uuid;not null;index" json:"school_id"`
	School         School        `gorm:"foreignKey:SchoolID" json:"-"`
	PostedBy       uuid.UUID     `gorm:"type:uuid;not null;index" json:"posted_by"`
	Poster         User          `gorm:"foreignKey:PostedBy" json:"-"`
	Title          string        `gorm:"not null" json:"title"`
	Body           string        `json:"body"`
	TargetDivision DivisionScope `gorm:"type:varchar(20)" json:"target_division"`
	IsPublished    bool          `gorm:"default:false;index" json:"is_published"`
	IsArchived     bool          `gorm:"default:false" json:"is_archived"`
}

// ─── Admission Window ──────────────────────────────────────────────────────────

type AdmissionWindow struct {
	Base
	SchoolID                   uuid.UUID `gorm:"type:uuid;not null;index" json:"school_id"`
	School                     School    `gorm:"foreignKey:SchoolID" json:"-"`
	SessionID                  uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	Session                    AcademicSession `gorm:"foreignKey:SessionID" json:"-"`
	Divisions                  JSONSlice `gorm:"type:jsonb" json:"divisions"` // ["NURSERY","PRIMARY","SECONDARY"]
	OpenDate                   time.Time `json:"open_date"`
	CloseDate                  time.Time `json:"close_date"`
	MaxSlotsPerDivision        JSONMap   `gorm:"type:jsonb" json:"max_slots_per_division"` // {"NURSERY": 50}
	AppointmentCapacityPerSlot int       `gorm:"default:5" json:"appointment_capacity_per_slot"`
	IsActive                   bool      `gorm:"default:false;index" json:"is_active"`
}

// ─── Document Verification ─────────────────────────────────────────────────────

type DocumentVerification struct {
	Base
	DocRef        string    `gorm:"uniqueIndex;not null" json:"doc_ref"`
	DocType       DocType   `gorm:"type:varchar(30);not null" json:"doc_type"`
	EntityID      string    `gorm:"not null;index" json:"entity_id"` // UUID of the related entity
	GeneratedAt   time.Time `json:"generated_at"`
	VerifiedCount int       `gorm:"default:0" json:"verified_count"`
	IssuedBy      string    `json:"issued_by"` // role name of issuer
	StudentName   string    `json:"student_name"` // for display on verify page
}

// ─── Grading Scale ─────────────────────────────────────────────────────────────

// GradeScale defines a single grade band (e.g. A = 70–100).
type GradeScale struct {
	Base
	SchoolID  uuid.UUID `gorm:"type:uuid;not null;index" json:"school_id"`
	School    School    `gorm:"foreignKey:SchoolID" json:"-"`
	Grade     string    `gorm:"not null" json:"grade"` // e.g. "A"
	MinScore  float64   `gorm:"not null" json:"min_score"`
	MaxScore  float64   `gorm:"not null" json:"max_score"`
	Remark    string    `json:"remark"` // e.g. "Excellent"
	IsPassing bool      `gorm:"default:true" json:"is_passing"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
}

// ─── Admission Sequence ────────────────────────────────────────────────────────

// AdmissionSequence tracks the per-division per-year sequence counter for Admission IDs.
type AdmissionSequence struct {
	Base
	SchoolID  uuid.UUID     `gorm:"type:uuid;not null;index" json:"school_id"`
	Division  DivisionScope `gorm:"type:varchar(20);not null" json:"division"`
	Year      int           `gorm:"not null" json:"year"`
	LastSeq   int           `gorm:"default:0" json:"last_seq"`
}

// ─── Quiz Question Assignment ──────────────────────────────────────────────────

// QuizQuestionAssignment links a quiz to its approved question pool.
type QuizQuestionAssignment struct {
	Base
	QuizID     uuid.UUID `gorm:"type:uuid;not null;index" json:"quiz_id"`
	Quiz       Quiz      `gorm:"foreignKey:QuizID" json:"-"`
	QuestionID uuid.UUID `gorm:"type:uuid;not null;index" json:"question_id"`
	Question   QuizQuestion `gorm:"foreignKey:QuestionID" json:"-"`
	IsApproved bool      `gorm:"default:false" json:"is_approved"`
}

// AllModels returns all GORM models for AutoMigrate.
func AllModels() []interface{} {
	return []interface{}{
		&School{},
		&User{},
		&Division{},
		&AcademicSession{},
		&Term{},
		&Class{},
		&Subject{},
		&TeacherAssignment{},
		&Student{},
		&StudentClassHistory{},
		&Application{},
		&AppointmentLetter{},
		&AdmissionLetter{},
		&Attendance{},
		&ScoreStructure{},
		&ScoreEntry{},
		&AuditLog{},
		&Result{},
		&ResultSubject{},
		&Quiz{},
		&QuizQuestion{},
		&QuizAttempt{},
		&QuizQuestionAssignment{},
		&FeeStructure{},
		&FeePayment{},
		&FeeDiscount{},
		&Timetable{},
		&ActivityPost{},
		&Notification{},
		&Announcement{},
		&AdmissionWindow{},
		&DocumentVerification{},
		&GradeScale{},
		&AdmissionSequence{},
	}
}
