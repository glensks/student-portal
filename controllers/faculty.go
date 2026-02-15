package controllers

import (
	"fmt"
	"net/http"
	"student-portal/config"

	"github.com/gin-gonic/gin"
)

func FacultyDashboard(c *gin.Context) {
	c.File("./frontend/faculty.html") // adjust path as needed
}

// FacultyMe: returns profile info for frontend
func FacultyMe(c *gin.Context) {
	userID := c.GetInt("user_id")

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome Faculty",
		"user_id": userID,
	})
}

// ===== COURSES =====
func FacultyCreateCourse(c *gin.Context) {
	var req struct {
		CourseName string `json:"course_name" binding:"required"`
		Code       string `json:"code" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := config.DB.Exec("INSERT INTO courses (course_name, code) VALUES (?, ?)", req.CourseName, req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"message": "Course created", "id": id})
}

func FacultyGetCourses(c *gin.Context) {
	rows, err := config.DB.Query("SELECT id, course_name, code FROM courses")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var courses []map[string]interface{}
	for rows.Next() {
		var id int
		var name, code string
		rows.Scan(&id, &name, &code)
		courses = append(courses, map[string]interface{}{
			"id":          id,
			"course_name": name,
			"code":        code,
		})
	}
	c.JSON(http.StatusOK, courses)
}

// ===== SUBJECTS =====
func FacultyCreateSubject(c *gin.Context) {
	var req struct {
		SubjectName string `json:"subject_name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		CourseID    int    `json:"course_id" binding:"required"`
		YearLevel   int    `json:"year_level" binding:"required"`
		Semester    string `json:"semester" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := config.DB.Exec(
		"INSERT INTO subjects (subject_name, code, course_id, year_level, semester) VALUES (?,?,?,?,?)",
		req.SubjectName, req.Code, req.CourseID, req.YearLevel, req.Semester,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"message": "Subject created", "id": id})
}

func FacultyGetSubjects(c *gin.Context) {
	courseID := c.Query("course_id")
	yearLevel := c.Query("year_level")
	semester := c.Query("semester")

	query := "SELECT id, subject_name, code, course_id, year_level, COALESCE(semester, '') FROM subjects"
	args := []interface{}{}
	conditions := []string{}

	if courseID != "" {
		conditions = append(conditions, "course_id=?")
		args = append(args, courseID)
	}
	if yearLevel != "" {
		conditions = append(conditions, "year_level=?")
		args = append(args, yearLevel)
	}
	if semester != "" {
		conditions = append(conditions, "semester=?")
		args = append(args, semester)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var subjects []map[string]interface{}
	for rows.Next() {
		var id, cID, year int
		var name, code, sem string
		rows.Scan(&id, &name, &code, &cID, &year, &sem)
		subjects = append(subjects, map[string]interface{}{
			"id":           id,
			"subject_name": name,
			"code":         code,
			"course_id":    cID,
			"year_level":   year,
			"semester":     sem,
		})
	}
	c.JSON(http.StatusOK, subjects)
}

// ===== TEACHER ASSIGNMENTS =====
func FacultyAssignTeacher(c *gin.Context) {
	type Schedule struct {
		SubjectID int    `json:"subject_id" binding:"required"`
		CourseID  int    `json:"course_id" binding:"required"`
		Room      string `json:"room" binding:"required"`
		Day       string `json:"day" binding:"required"`
		StartTime string `json:"start_time" binding:"required"`
		EndTime   string `json:"end_time" binding:"required"`
	}

	var req struct {
		TeacherID int        `json:"teacher_id" binding:"required"`
		Schedules []Schedule `json:"schedules" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// DEBUG: Log what we received
	fmt.Printf("DEBUG - Teacher ID: %d\n", req.TeacherID)
	for i, s := range req.Schedules {
		fmt.Printf("DEBUG - Schedule %d: SubjectID=%d, CourseID=%d, Room='%s', Day=%s, Start=%s, End=%s\n",
			i, s.SubjectID, s.CourseID, s.Room, s.Day, s.StartTime, s.EndTime)
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO teacher_subjects (teacher_id, subject_id, course_id, room, day, time_start, time_end) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stmt.Close()

	for _, s := range req.Schedules {
		// Validate room is not empty
		if s.Room == "" {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Room cannot be empty"})
			return
		}

		// DEBUG: Log the exact insert
		fmt.Printf("DEBUG - Inserting: teacher_id=%d, subject_id=%d, course_id=%d, room='%s', day=%s, start=%s, end=%s\n",
			req.TeacherID, s.SubjectID, s.CourseID, s.Room, s.Day, s.StartTime, s.EndTime)

		if _, err := stmt.Exec(req.TeacherID, s.SubjectID, s.CourseID, s.Room, s.Day, s.StartTime, s.EndTime); err != nil {
			tx.Rollback()
			// DEBUG: Log the error
			fmt.Printf("DEBUG - Insert error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Teacher assigned successfully"})
}

func FacultyGetTeachers(c *gin.Context) {
	rows, err := config.DB.Query("SELECT id, username, email FROM users WHERE role = 'teacher'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var teachers []map[string]interface{}
	for rows.Next() {
		var id int
		var username, email string
		rows.Scan(&id, &username, &email)
		teachers = append(teachers, map[string]interface{}{
			"id":       id,
			"username": username,
			"email":    email,
		})
	}
	c.JSON(http.StatusOK, teachers)
}

func FacultySetSchoolYear(c *gin.Context) {
	var req struct {
		Year     string `json:"year" binding:"required"`
		Semester string `json:"semester" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Deactivate old years
	if _, err := config.DB.Exec("UPDATE school_year SET is_active = FALSE"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err := config.DB.Exec(
		"INSERT INTO school_year (year, semester, is_active) VALUES (?, ?, TRUE)",
		req.Year, req.Semester,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "School year set"})
}
