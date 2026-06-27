// Package permissions provides a centralised role-permission map.
// ALL permission checks throughout the application MUST go through HasPermission().
// No permission logic is scattered inline through handlers.
package permissions

import "school-platform/internal/models"

// Permission is a named capability string.
type Permission string

const (
	PermManageSchoolSettings Permission = "manage_school_settings"
	PermManageUsers          Permission = "manage_users"
	PermManageStudents       Permission = "manage_students"
	PermManagePupils         Permission = "manage_pupils"
	PermManageTeachers       Permission = "manage_teachers"
	PermManageFinances       Permission = "manage_finances"
	PermManageResults        Permission = "manage_results"
	PermManageAttendance     Permission = "manage_attendance"
	PermManageQuizzes        Permission = "manage_quizzes"
	PermApproveQuizQuestions Permission = "approve_quiz_questions"
	PermSubmitQuizQuestions  Permission = "submit_quiz_questions"
	PermTakeQuiz             Permission = "take_quiz"
	PermPublishBlogPosts     Permission = "publish_blog_posts"
	PermManageAdmissions     Permission = "manage_admissions"
	PermViewReports          Permission = "view_reports"
	PermManageSessions       Permission = "manage_sessions"
	PermManageTerms          Permission = "manage_terms"
	PermManageSubjects       Permission = "manage_subjects"
	PermManageTimetable      Permission = "manage_timetable"
	PermCommunicateParents   Permission = "communicate_parents"
	PermManageClasses        Permission = "manage_classes"
	PermAssignTeachers       Permission = "assign_teachers"
	PermEnterScores          Permission = "enter_scores"
	PermApproveScores        Permission = "approve_scores"
	PermPublishResults       Permission = "publish_results"
	PermViewOwnChildFees     Permission = "view_own_child_fees"
	PermPostActivityFeed     Permission = "post_activity_feed"
	PermViewActivityFeed     Permission = "view_activity_feed"
	PermPromoteStudents      Permission = "promote_students"
)

// rolePermissions maps each role to its allowed permissions.
var rolePermissions = map[models.Role][]Permission{
	models.RoleOwner: {
		PermManageSchoolSettings,
		PermManageUsers,
		PermManageStudents,
		PermManagePupils,
		PermManageTeachers,
		PermManageFinances,
		PermManageResults,
		PermManageAttendance,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPublishBlogPosts,
		PermManageAdmissions,
		PermViewReports,
		PermManageSessions,
		PermManageTerms,
		PermManageSubjects,
		PermManageTimetable,
		PermCommunicateParents,
		PermManageClasses,
		PermAssignTeachers,
		PermEnterScores,
		PermApproveScores,
		PermPublishResults,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPromoteStudents,
	},

	models.RolePrincipal: {
		PermManageStudents,
		PermManageTeachers,
		PermManageResults,
		PermManageAttendance,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPublishBlogPosts,
		PermManageAdmissions,
		PermViewReports,
		PermManageTerms,
		PermManageSubjects,
		PermManageTimetable,
		PermCommunicateParents,
		PermManageClasses,
		PermAssignTeachers,
		PermApproveScores,
		PermPublishResults,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPromoteStudents,
	},

	models.RoleVicePrincipal: {
		PermManageStudents,
		PermManageTeachers,
		PermManageResults,
		PermManageAttendance,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPublishBlogPosts,
		PermManageAdmissions,
		PermViewReports,
		PermManageSubjects,
		PermManageTimetable,
		PermCommunicateParents,
		PermManageClasses,
		PermAssignTeachers,
		PermApproveScores,
		PermPublishResults,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPromoteStudents,
	},

	models.RoleHeadTeacher: {
		PermManagePupils,
		PermManageStudents,
		PermManageTeachers,
		PermManageResults,
		PermManageAttendance,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPublishBlogPosts,
		PermManageAdmissions,
		PermViewReports,
		PermManageTerms,
		PermManageSubjects,
		PermManageTimetable,
		PermCommunicateParents,
		PermManageClasses,
		PermAssignTeachers,
		PermApproveScores,
		PermPublishResults,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPromoteStudents,
	},

	models.RoleAsstHeadTeacher: {
		PermManagePupils,
		PermManageStudents,
		PermManageTeachers,
		PermManageResults,
		PermManageAttendance,
		PermManageQuizzes,
		PermApproveQuizQuestions,
		PermSubmitQuizQuestions,
		PermPublishBlogPosts,
		PermManageAdmissions,
		PermViewReports,
		PermManageSubjects,
		PermManageTimetable,
		PermCommunicateParents,
		PermManageClasses,
		PermAssignTeachers,
		PermApproveScores,
		PermPublishResults,
		PermPostActivityFeed,
		PermViewActivityFeed,
		PermPromoteStudents,
	},

	models.RoleBursar: {
		PermManageFinances,
		PermViewReports,
	},

	models.RoleTeacher: {
		PermManageAttendance,
		PermSubmitQuizQuestions,
		PermEnterScores,
		PermCommunicateParents,
		PermViewActivityFeed,
	},

	models.RoleStudent: {
		PermTakeQuiz,
		PermViewActivityFeed,
	},

	models.RolePupil: {
		PermTakeQuiz,
		PermViewActivityFeed,
	},

	models.RoleParent: {
		PermViewOwnChildFees,
		PermViewActivityFeed,
	},
}

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

// IsSeniorRole returns true for roles that can manage other staff.
func IsSeniorRole(role models.Role) bool {
	switch role {
	case models.RoleOwner, models.RolePrincipal, models.RoleVicePrincipal,
		models.RoleHeadTeacher, models.RoleAsstHeadTeacher:
		return true
	}
	return false
}

// IsAdminRole returns true for roles with administrative access.
func IsAdminRole(role models.Role) bool {
	switch role {
	case models.RoleOwner, models.RolePrincipal, models.RoleVicePrincipal,
		models.RoleHeadTeacher, models.RoleAsstHeadTeacher, models.RoleBursar:
		return true
	}
	return false
}

// CanPromoteStudents returns true for roles allowed to promote students.
func CanPromoteStudents(role models.Role) bool {
	return HasPermission(role, PermPromoteStudents)
}

// CanPublishResults returns true for roles allowed to publish results.
func CanPublishResults(role models.Role) bool {
	return HasPermission(role, PermPublishResults)
}
