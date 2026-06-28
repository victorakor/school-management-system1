package models

// This file extends the Role constants defined in models.go with the
// missing roles from the LEAPS RBAC specification.
// Add these here to avoid editing the large models.go directly.

const (
	// RoleExamOfficer manages academic quality and examination integrity.
	RoleExamOfficer Role = "EXAM_OFFICER"

	// RoleAdmissionsOfficer processes applications and schedules interviews.
	RoleAdmissionsOfficer Role = "ADMISSIONS_OFFICER"

	// RoleClassTeacher is a Teacher who also manages a form/class.
	RoleClassTeacher Role = "CLASS_TEACHER"

	// RoleBlogManager handles school media and public-facing content only.
	RoleBlogManager Role = "BLOG_MANAGER"

	// RoleICTAdmin handles technical administration (passwords, backups, health).
	RoleICTAdmin Role = "ICT_ADMIN"
)
