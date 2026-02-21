package main

import (
	"os"
	"student-portal/config"
	"student-portal/controllers"
	"student-portal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// ---------------- CONNECT TO DATABASE ----------------
	config.ConnectDB()
	config.CreateDefaultUsers()

	// ---------------- CREATE GIN ROUTER ----------------
	r := gin.Default()

	// ---------------- FIXED CORS MIDDLEWARE ----------------
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ---------------- STATIC FILES ----------------
	r.Static("/assets", "./frontend")
	r.Static("/uploads", "./uploads")

	// ---------------- PUBLIC PAGES ----------------
	r.GET("/", func(c *gin.Context) { c.File("./frontend/login.html") })
	r.GET("/login", func(c *gin.Context) { c.File("./frontend/login.html") })
	r.GET("/login.html", func(c *gin.Context) { c.File("./frontend/login.html") })

	// ---------------- LOGIN API ----------------
	r.POST("/login", controllers.Login)
	r.POST("/register-student", controllers.RegisterStudent)
	r.GET("/subjects", controllers.FacultyGetSubjects)

	r.POST("/forgot-password", controllers.ForgotPassword)
	r.GET("/verify-reset-token", controllers.VerifyResetToken)
	r.POST("/reset-password", controllers.ResetPassword)
	r.GET("/reset-password", func(c *gin.Context) {
		c.File("./frontend/reset_password.html")
	})

	// ---------------- PROTECTED ROUTES ----------------
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())

	// ✅ FIXED: was using r.GET (unprotected), now correctly uses protected.GET
	protected.GET("/public/courses", controllers.FacultyGetCourses)
	protected.GET("/public/subjects", controllers.FacultyGetSubjects)

	// ---------------- ADMIN ROUTES ----------------
	admin := protected.Group("/admin")
	admin.Use(middleware.RoleOnly("admin"))
	admin.GET("/dashboard", func(c *gin.Context) { c.File("./frontend/admin.html") })

	// User CRUD
	admin.GET("/users", controllers.AdminGetUsers)
	admin.POST("/users", controllers.AdminCreateUser)
	admin.PUT("/users/:id", controllers.AdminEditUser)
	admin.DELETE("/users/:id", controllers.AdminDeleteUser)

	// IT Management
	admin.GET("/users/:id/details", controllers.AdminGetUserDetails)
	admin.POST("/users/:id/reset-password", controllers.AdminResetUserPassword)
	admin.PUT("/users/:id/status", controllers.AdminToggleUserStatus)
	admin.POST("/users/bulk-status", controllers.AdminBulkUpdateStatus)
	admin.GET("/search", controllers.AdminSearchUsers)
	admin.GET("/stats", controllers.AdminGetSystemStats)
	admin.GET("/students", controllers.AdminGetStudents)
	admin.GET("/students/:id/details", controllers.AdminGetStudentDetails)
	admin.PUT("/students/:id/status", controllers.AdminToggleStudentStatus)
	admin.POST("/students/:id/reset-password", controllers.AdminResetStudentPassword)
	admin.GET("/students/search", controllers.AdminSearchStudents)
	admin.DELETE("/students/:id", controllers.AdminDeleteStudent)

	// ---------------- TEACHER ROUTES ----------------
	teacher := protected.Group("/teacher")
	teacher.Use(middleware.RoleOnly("teacher"))
	teacher.GET("/dashboard", func(c *gin.Context) { c.File("./frontend/teacher.html") })
	teacher.GET("/classes", controllers.TeacherGetClasses)
	teacher.GET("/students", controllers.TeacherGetStudents)
	teacher.POST("/grades", controllers.TeacherSubmitGrade)
	teacher.GET("/grades", controllers.TeacherGetGrades)
	teacher.POST("/lessons", controllers.TeacherUploadLesson)
	teacher.GET("/lessons", controllers.TeacherGetLessonMaterials)
	teacher.GET("/lessons/:id/submissions", controllers.TeacherGetSubmissions)
	teacher.PUT("/lessons/:id", controllers.TeacherUpdateLesson)
	teacher.DELETE("/lessons/:id", controllers.TeacherDeleteLessonMaterial)
	teacher.GET("/submissions/pending", controllers.TeacherGetPendingSubmissions)
	teacher.POST("/submissions/:id/review", controllers.TeacherReviewSubmission)
	teacher.POST("/announcements", controllers.TeacherPostAnnouncement)
	teacher.GET("/announcements", controllers.TeacherGetAnnouncements)
	teacher.DELETE("/announcements/:id", controllers.TeacherDeleteAnnouncement)

	// ---------------- STUDENT ROUTES ----------------
	student := protected.Group("/student")
	student.Use(middleware.RoleOnly("student"))

	student.GET("/dashboard", func(c *gin.Context) {
		c.File("./frontend/student.html")
	})
	student.GET("/payments/me", controllers.StudentGetPaymentsMe)
	student.POST("/pay", controllers.StudentPayBill)
	student.POST("/payments/downpayment", controllers.StudentDownPayment)
	student.GET("/installments", controllers.StudentGetInstallments)
	student.GET("/schedule", controllers.StudentGetSchedule)
	student.POST("/documents/request", controllers.StudentRequestDocument)
	student.GET("/lessons", controllers.StudentGetLessons)
	student.POST("/submissions/upload", controllers.StudentUploadSubmission)
	student.GET("/submissions", controllers.StudentGetSubmissions)
	student.GET("/documents/requests", controllers.StudentGetDocumentRequests)
	student.GET("/grades", controllers.StudentGetGrades)
	student.GET("/profile", controllers.StudentGetProfile)
	student.PUT("/profile/update", controllers.StudentUpdateProfile)
	student.POST("/profile/change-password", controllers.StudentChangePassword)
	student.POST("/profile/upload-picture", controllers.StudentUploadProfilePicture)
	student.GET("/announcements", controllers.StudentGetAnnouncements)
	student.POST("/enroll", controllers.StudentSubmitEnrollment)
	student.GET("/enrollments", controllers.StudentGetEnrollments)

	// ---------------- REGISTRAR ROUTES ----------------
	registrar := protected.Group("/registrar")
	registrar.Use(middleware.RoleOnly("registrar"))
	registrar.GET("/dashboard", func(c *gin.Context) {
		c.File("./frontend/registrar.html")
	})
	registrar.GET("/students", controllers.RegistrarGetStudentsByStatus)
	registrar.POST("/approve-with-assessment", controllers.RegistrarApproveWithAssessment)
	registrar.POST("/announcements", controllers.RegistrarPostAnnouncement)
	registrar.GET("/announcements", controllers.RegistrarGetAnnouncements)
	registrar.DELETE("/announcements/:id", controllers.RegistrarDeleteAnnouncement)
	registrar.GET("/enrollment-applications", controllers.RegistrarGetEnrollmentApplications)
	registrar.POST("/enrollment-applications/approve", controllers.RegistrarApproveEnrollmentApplication)
	registrar.POST("/enrollment-applications/reject", controllers.RegistrarRejectEnrollmentApplication)

	// ✅ FIXED: was using r.POST (unprotected), now correctly uses registrar group
	registrar.POST("/student/status", controllers.RegistrarUpdateStudentStatus)

	// ---------------- CASHIER ROUTES ----------------
	cashier := protected.Group("/cashier")
	cashier.Use(middleware.RoleOnly("cashier"))
	cashier.GET("/dashboard", func(c *gin.Context) {
		c.File("./frontend/cashier.html")
	})
	cashier.GET("/pending-payments", controllers.CashierGetPendingPayments)
	cashier.POST("/approve-payment", controllers.CashierApprovePayment)

	// ---------------- RECORDS ROUTES ----------------
	records := protected.Group("/records")
	records.Use(middleware.RoleOnly("records"))
	records.GET("/dashboard", func(c *gin.Context) {
		c.File("./frontend/records.html")
	})
	records.GET("/me", controllers.RecordsMe)
	records.GET("/document-requests", controllers.RecordsGetDocumentRequests)
	records.GET("/document-requests/:id", controllers.RecordsGetDocumentRequestDetails)
	records.POST("/document-requests/:id", controllers.RecordsProcessDocumentRequest)
	records.GET("/grades", controllers.RecordsGetAllGrades)
	records.GET("/grades/student/:student_id", controllers.RecordsGetStudentGrades)
	records.POST("/grades/:grade_id/release", controllers.RecordsReleaseGrade)
	records.GET("/announcements", controllers.RecordsGetAnnouncements)
	records.GET("/announcements/:id", controllers.RecordsGetAnnouncementDetails)
	records.POST("/announcements", controllers.RecordsPostAnnouncement)
	records.PUT("/announcements/:id", controllers.RecordsUpdateAnnouncement)
	records.POST("/announcements/:id/toggle", controllers.RecordsToggleAnnouncement)
	records.DELETE("/announcements/:id", controllers.RecordsDeleteAnnouncement)

	// ---------------- FACULTY ROUTES ----------------
	faculty := protected.Group("/faculty")
	faculty.Use(middleware.RoleOnly("faculty"))
	faculty.GET("/dashboard", func(c *gin.Context) { c.File("./frontend/faculty.html") })
	faculty.GET("/me", controllers.FacultyMe)
	faculty.POST("/assign-teacher", controllers.FacultyAssignTeacher)
	faculty.GET("/teachers", controllers.FacultyGetTeachers)
	faculty.GET("/courses", controllers.FacultyGetCourses)
	faculty.POST("/courses", controllers.FacultyCreateCourse)
	faculty.GET("/subjects", controllers.FacultyGetSubjects)
	faculty.POST("/subjects", controllers.FacultyCreateSubject)
	faculty.POST("/school-year", controllers.FacultySetSchoolYear)

	// ---------------- TEST PING ----------------
	r.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })

	// ---------------- RUN SERVER ----------------
	r.Run(":" + os.Getenv("PORT"))
}
