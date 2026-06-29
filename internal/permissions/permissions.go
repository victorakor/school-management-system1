// Package permissions provides a centralised role-permission map.
// ALL permission checks throughout the application MUST go through HasPermission().
// No permission logic is scattered inline through handlers.
package permissions

import "school-platform/internal/models"

// Permission is a named capability string.
type Permission string

// ─── Permission constants ──────────────────────────────────────────────────────
//
// Naming convention: Perm<Action><Resource>
// Actions: Manage (full CRUD), View, Enter, Approve, Publish, Promote, Post, Take, Submit, Override

const (
	// School-wide administration
	PermManageSchoolSettings Permission = "manage_school_settings"
	PermManageUsers          Permission = "manage_users"
	PermManageSessions       Permission = "manage_sessions"
	PermManageTerms          Permission = "manage_terms"
	PermManageClasses        Permission = "manage_classes"
	PermManageSubjects       Permission = "manage_subjects"
	PermAssignTeachers       Permission = "assign_teachers"
	PermViewReports          Permission = "view_reports"

	// Student / pupil management
	PermManageStudents  Permission = "manage_students" // secondary
	PermManagePupils    Permission = "manage_pupils"   // nursery & primary
	PermPromoteStudents Permission = "promote_students"

	// Teacher management (view + assign — not account creation)
	PermManageTeachers Permission = "manage_teachers"

	// Scores / Results workflow
	PermEnterScores     Permission = "enter_scores"     // teachers: save draft scores
	PermApproveScores   Permission = "approve_scores"   // senior staff: approve/reject submissions
	PermManageResults   Permission = "manage_results"   // calculate remarks
	PermPublishResults  Permission = "publish_results"  // make results visible to students/parents
	PermOverrideResults Permission = "override_results" // owner only: bypass locks

	// Exam / Academic Officer specific
	PermConfigureAssessments Permission = "configure_assessments" // set score structures / periods
	PermValidateScores       Permission = "validate_scores"       // detect missing / inconsistent scores
	PermLockExamData         Permission = "lock_exam_data"        // lock after approval

	// Admissions
	PermManageAdmissions  Permission = "manage_admissions"  // full admissions management
	PermProcessAdmissions Permission = "process_admissions" // admissions officer: verify docs, schedule, generate letters
	PermApproveAdmissions Permission = "approve_admissions" // principal/head/owner: final approve/reject

	// Finance (scoped by DivisionScope on the user record)
	PermManageFinances   Permission = "manage_finances"     // bursar: create structure, record payments, discounts
	PermViewOwnChildFees Permission = "view_own_child_fees" // parent: read-only fee balance + receipts

	// Timetable
	PermManageTimetable Permission = "manage_timetable"

	// Attendance
	PermManageAttendance Permission = "manage_attendance"

	// Quiz
	PermManageQuizzes        Permission = "manage_quizzes"         // create, schedule, publish, close quizzes
	PermApproveQuizQuestions Permission = "approve_quiz_questions" // approve/flag submitted questions
	PermSubmitQuizQuestions  Permission = "submit_quiz_questions"  // teachers: submit questions to pool
	PermTakeQuiz             Permission = "take_quiz"              // students/pupils

	// Activity feed / blog / announcements
	PermPostActivityFeed  Permission = "post_activity_feed"  // create, edit, delete posts
	PermViewActivityFeed  Permission = "view_activity_feed"  // read posts
	PermPublishBlogPosts  Permission = "publish_blog_posts"  // publish announcements (alias kept for compat)
	PermManageBlogContent Permission = "manage_blog_content" // blog manager: full media & post management

	// Communication
	PermCommunicateParents Permission = "communicate_parents"

	// Behavioural / holistic assessments (class teacher only)
	PermEnterBehaviouralAssessment Permission = "enter_behavioural_assessment"

	// ICT / system administration (no academic or finance access)
	PermManageSystemConfig Permission = "manage_system_config" // passwords, backups, email/SMS config, health

	// Audit log access (Owner only — cannot be delegated)
	PermViewAuditLogs Permission = "view_audit_logs"
)

// ─── Role → Permission map ─────────────────────────────────────────────────────
//
// DIVISION SCOPE is enforced separately via CanAccessDivision().
// The permissions below say *what* a role can do; division scope says *where*.
//
// Key deviations from a naïve "more senior = more permissions" approach:
//   - HeadTeacher mirrors Principal but is restricted to Nursery/Primary by DivisionScope.
//   - ExamOfficer can configure assessments and lock data but CANNOT change scores.
//   - AdmissionsOfficer can process (verify/schedule/generate) but CANNOT approve.
//   - Bursar has only finance + view-reports; zero academic access.
//   - ClassTeacher extends Teacher with behavioural assessments and can recommend
//     promotion, but cannot publish results.
//   - BlogManager has zero academic, finance, or admissions access.
//   - ICTAdmin has zero academic, finance, admissions, or results access.

var rolePermissions = map[models.Role][]Permission{

	// ── School Owner (Proprietor) ──────────────────────────────────────────────
	// Full access to everything including overrides and audit.
	models.RoleOwner: {
		PermManageSchoolSettings,
		PermManageUsers,
		PermManageSessions,
		PermManageTerms,
		PermManageClasses,
		PermManageSubjects,
		PermAssignTeachers,
		PermViewReports,
		PermManageStudents,
		PermManagePupils,
		PermPromoteStudents,
		PermManageTeachers,
		PermEnterScores,
		PermApproveScores,
		PermManageResults,
		PermPublishResults,
		PermOverrideResults,
		PermConfigureAssessments,
		PermValidateScores,
		PermLockExamData,
		PermManageAdmissions,
		PermProcessAdmissions,
		PermApproveAdmissions,
		PermManageFinances,
		PermManageTimetable,
		PermManageAttendance,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPublishBlogPosts,
		PermManageBlogContent,
		PermCommunicateParents,
		PermEnterBehaviouralAssessment,
		PermManageSystemConfig,
		PermViewAuditLogs,
	},

	// ── Principal (Secondary scope) ───────────────────────────────────────────
	// Manages secondary school. Cannot create staff accounts (owner only).
	// Cannot directly edit teacher scores; can approve/reject and publish.
	models.RolePrincipal: {
		PermManageStudents, // secondary students only (enforced by DivisionScope)
		PermManageTeachers, // view + assign secondary teachers
		PermManageClasses,
		PermManageSubjects,
		PermAssignTeachers,
		PermApproveScores,
		PermManageResults,
		PermPublishResults,
		PermPromoteStudents,
		PermManageAttendance,
		PermManageAdmissions,
		PermApproveAdmissions,
		PermManageTimetable,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPublishBlogPosts,
		PermCommunicateParents,
		PermManageTerms,
		PermViewReports,
	},

	// ── Vice Principal (Secondary scope) ──────────────────────────────────────
	// Same scope as Principal. Cannot publish final results without Principal
	// approval in a dual-approval workflow (enforced at service layer).
	models.RoleVicePrincipal: {
		PermManageStudents,
		PermManageTeachers,
		PermManageClasses,
		PermManageSubjects,
		PermAssignTeachers,
		PermApproveScores,
		PermManageResults,
		PermPublishResults, // dual-approval check at service layer
		PermPromoteStudents,
		PermManageAttendance,
		PermManageAdmissions,
		PermApproveAdmissions,
		PermManageTimetable,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPublishBlogPosts,
		PermCommunicateParents,
		PermViewReports,
	},

	// ── Head Teacher (Nursery/Primary scope) ──────────────────────────────────
	// Mirrors Principal permissions but scoped to Nursery & Primary via DivisionScope.
	models.RoleHeadTeacher: {
		PermManagePupils, // nursery/primary pupils
		PermManageTeachers,
		PermManageClasses,
		PermManageSubjects,
		PermAssignTeachers,
		PermApproveScores,
		PermManageResults,
		PermPublishResults,
		PermPromoteStudents,
		PermManageAttendance,
		PermManageAdmissions,
		PermApproveAdmissions,
		PermManageTimetable,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPublishBlogPosts,
		PermCommunicateParents,
		PermManageTerms,
		PermViewReports,
	},

	// ── Assistant Head Teacher (Nursery/Primary scope) ────────────────────────
	// Supports the Head Teacher. Can review and monitor but cannot override
	// Head Teacher decisions and cannot publish results.
	models.RoleAsstHeadTeacher: {
		PermManagePupils,
		PermManageTeachers,
		PermManageClasses,
		PermManageSubjects,
		PermAssignTeachers,
		PermApproveScores,
		PermManageResults, // can calculate / add remarks; cannot publish
		PermPromoteStudents,
		PermManageAttendance,
		PermManageAdmissions,
		PermManageTimetable,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPublishBlogPosts,
		PermCommunicateParents,
		PermViewReports,
	},

	// ── Exam / Academic Officer ────────────────────────────────────────────────
	// Ensures academic quality and examination integrity.
	// Can configure assessment periods, validate completeness, and lock data.
	// CANNOT change teacher-entered scores or touch finances.
	models.RoleExamOfficer: {
		PermConfigureAssessments, // set score structures
		PermValidateScores,       // detect missing / inconsistent scores
		PermApproveScores,        // approve or return for correction
		PermManageResults,        // trigger computation; add admin remark
		PermLockExamData,         // lock after publication
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermViewReports,
	},

	// ── Admissions Officer ────────────────────────────────────────────────────
	// Processes applications through to the point of an approved decision.
	// CANNOT approve admissions independently — that requires Principal/Owner.
	models.RoleAdmissionsOfficer: {
		PermProcessAdmissions, // verify docs, schedule interviews, generate letters
		PermManageAdmissions,  // view all applications (read-heavy; updates limited to processing)
		PermViewReports,
	},

	// ── Bursar (Secondary) ────────────────────────────────────────────────────
	// Finance only. DivisionScope = SECONDARY.
	// ── Bursar (Primary/Nursery) ──────────────────────────────────────────────
	// Finance only. DivisionScope = PRIMARY.
	// Both share the same Role constant; division access is enforced by DivisionScope.
	models.RoleBursar: {
		PermManageFinances,
		PermViewReports,
	},

	// ── Teacher ───────────────────────────────────────────────────────────────
	// Scoped to assigned classes and subjects only (enforced at handler level).
	// Can enter scores and record attendance for their own students.
	// Cannot compute grades, compute positions, or publish results.
	models.RoleTeacher: {
		PermManageAttendance,
		PermSubmitQuizQuestions,
		PermEnterScores,
		PermCommunicateParents,
		PermViewActivityFeed,
	},

	// ── Class Teacher ─────────────────────────────────────────────────────────
	// Everything a Teacher can do plus holistic assessments, remarks,
	// attendance monitoring, and promotion recommendations.
	// CANNOT publish results or approve scores.
	models.RoleClassTeacher: {
		PermManageAttendance,
		PermSubmitQuizQuestions,
		PermEnterScores,
		PermCommunicateParents,
		PermViewActivityFeed,
		PermEnterBehaviouralAssessment, // psychomotor, affective, teacher remarks
		// fee status is read-only; enforced at handler level via view_own_child_fees logic
	},

	// ── Student (Secondary) ───────────────────────────────────────────────────
	models.RoleStudent: {
		PermTakeQuiz,
		PermViewActivityFeed,
	},

	// ── Pupil (Nursery/Primary) ───────────────────────────────────────────────
	models.RolePupil: {
		PermTakeQuiz,
		PermViewActivityFeed,
	},

	// ── Parent / Guardian ─────────────────────────────────────────────────────
	// Linked only to their own children. Read-only across all areas.
	models.RoleParent: {
		PermViewOwnChildFees,
		PermViewActivityFeed,
	},

	// ── Blog / Media Manager ──────────────────────────────────────────────────
	// School publicity only. Zero academic, finance, or admissions access.
	models.RoleBlogManager: {
		PermManageBlogContent,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPublishBlogPosts,
	},

	// ── ICT / System Administrator ────────────────────────────────────────────
	// Day-to-day technical administration. Cannot modify results, finances,
	// or approve admissions.
	models.RoleICTAdmin: {
		PermManageSystemConfig,
		PermViewReports, // system health / audit reports only
	},
}

// ─── Public helpers ────────────────────────────────────────────────────────────

// HasPermission returns true if the given role has the specified permission.
// This is the SINGLE entry point for all permission checks in the application.
func HasPermission(role models.Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a given role.
func GetPermissions(role models.Role) []Permission {
	return rolePermissions[role]
}

// CanAccessDivision enforces division scope rules.
// Returns true if the user's division scope allows access to the requested division.
func CanAccessDivision(userScope models.DivisionScope, requestedDivision models.DivisionScope) bool {
	if userScope == models.DivisionAll {
		return true
	}
	return userScope == requestedDivision
}

// IsSeniorRole returns true for roles that can manage other staff and approve scores.
func IsSeniorRole(role models.Role) bool {
	switch role {
	case models.RoleOwner, models.RolePrincipal, models.RoleVicePrincipal,
		models.RoleHeadTeacher, models.RoleAsstHeadTeacher, models.RoleExamOfficer:
		return true
	}
	return false
}

// IsAdminRole returns true for roles with administrative portal access
// (i.e. not students, pupils, or parents).
func IsAdminRole(role models.Role) bool {
	switch role {
	case models.RoleStudent, models.RolePupil, models.RoleParent:
		return false
	}
	return true
}

// IsFinanceRole returns true for roles with finance access.
func IsFinanceRole(role models.Role) bool {
	return HasPermission(role, PermManageFinances)
}

// CanPublishResults returns true for roles allowed to publish results.
func CanPublishResults(role models.Role) bool {
	return HasPermission(role, PermPublishResults)
}

// CanPromoteStudents returns true for roles allowed to promote students.
func CanPromoteStudents(role models.Role) bool {
	return HasPermission(role, PermPromoteStudents)
}

// CanApproveAdmissions returns true for roles that may give final admission decisions.
func CanApproveAdmissions(role models.Role) bool {
	return HasPermission(role, PermApproveAdmissions)
}

// AllRoles returns every defined role for validation / UI purposes.
func AllRoles() []models.Role {
	return []models.Role{
		models.RoleOwner,
		models.RolePrincipal,
		models.RoleVicePrincipal,
		models.RoleHeadTeacher,
		models.RoleAsstHeadTeacher,
		models.RoleExamOfficer,
		models.RoleAdmissionsOfficer,
		models.RoleBursar,
		models.RoleTeacher,
		models.RoleClassTeacher,
		models.RoleStudent,
		models.RolePupil,
		models.RoleParent,
		models.RoleBlogManager,
		models.RoleICTAdmin,
	}
}

// StaffRoles returns roles that can be created via the staff management UI
// (i.e. roles the Owner assigns, excluding student/pupil/parent self-registration).
func StaffRoles() []models.Role {
	return []models.Role{
		models.RolePrincipal,
		models.RoleVicePrincipal,
		models.RoleHeadTeacher,
		models.RoleAsstHeadTeacher,
		models.RoleExamOfficer,
		models.RoleAdmissionsOfficer,
		models.RoleBursar,
		models.RoleTeacher,
		models.RoleClassTeacher,
		models.RoleBlogManager,
		models.RoleICTAdmin,
	}
}
