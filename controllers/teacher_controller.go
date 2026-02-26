package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"student-portal/config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===== VIEW CLASSES (Assigned by Admin) =====
func TeacherGetClasses(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Teacher role required"})
		return
	}

	query := `
		SELECT 
			ts.id,
			s.subject_name,
			s.code as subject_code,
			c.course_name,
			c.code as course_code,
			ts.room,
			ts.day,
			ts.time_start,
			ts.time_end
		FROM teacher_subjects ts
		INNER JOIN subjects s ON ts.subject_id = s.id
		INNER JOIN courses c ON ts.course_id = c.id
		WHERE ts.teacher_id = ?
		ORDER BY 
			FIELD(ts.day, 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'),
			ts.time_start
	`

	rows, err := config.DB.Query(query, teacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch classes: " + err.Error()})
		return
	}
	defer rows.Close()

	var classes []gin.H

	for rows.Next() {
		var (
			id                                       int
			subject, subjectCode, course, courseCode string
			day, timeStart, timeEnd                  string
			roomNull                                 sql.NullString
		)

		if err := rows.Scan(&id, &subject, &subjectCode, &course, &courseCode, &roomNull, &day, &timeStart, &timeEnd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse class data: " + err.Error()})
			return
		}

		room := "Not assigned"
		if roomNull.Valid {
			room = roomNull.String
		}

		classes = append(classes, gin.H{
			"id":           id,
			"subject":      subject,
			"subject_code": subjectCode,
			"course":       course,
			"course_code":  courseCode,
			"room":         room,
			"day":          day,
			"time_start":   timeStart,
			"time_end":     timeEnd,
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading class data: " + err.Error()})
		return
	}

	if classes == nil {
		classes = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"classes": classes,
		"total":   len(classes),
	})
}

// ===== GET SINGLE CLASS DETAILS =====
func TeacherGetClassDetails(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	classID := c.Param("id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	query := `
		SELECT 
			ts.id,
			s.subject_name,
			s.code as subject_code,
			c.course_name,
			c.code as course_code,
			ts.room,
			ts.day,
			ts.time_start,
			ts.time_end
		FROM teacher_subjects ts
		INNER JOIN subjects s ON ts.subject_id = s.id
		INNER JOIN courses c ON ts.course_id = c.id
		WHERE ts.id = ? AND ts.teacher_id = ?
	`

	var (
		id                                       int
		subject, subjectCode, course, courseCode string
		day, timeStart, timeEnd                  string
		roomNull                                 sql.NullString
	)

	err := config.DB.QueryRow(query, classID, teacherID).Scan(
		&id, &subject, &subjectCode, &course, &courseCode, &roomNull, &day, &timeStart, &timeEnd,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	room := "Not assigned"
	if roomNull.Valid {
		room = roomNull.String
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           id,
		"subject":      subject,
		"subject_code": subjectCode,
		"course":       course,
		"course_code":  courseCode,
		"room":         room,
		"day":          day,
		"time_start":   timeStart,
		"time_end":     timeEnd,
	})
}

// ===== GET STUDENTS (only approved students) =====
func TeacherGetStudents(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	classID := c.Query("class_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	// FIX: Added AND st.status = 'approved' so only registrar-confirmed students appear
	query := `
		SELECT DISTINCT
			st.id,
			st.first_name,
			st.last_name,
			st.email,
			IFNULL(sa.year_level, 0)
		FROM teacher_subjects ts
		INNER JOIN student_academic sa
			ON FIND_IN_SET(ts.subject_id, sa.subjects) > 0
		INNER JOIN students st
			ON st.id = sa.student_id
		WHERE ts.teacher_id = ? AND ts.id = ?
		AND st.status = 'approved'
		ORDER BY st.last_name, st.first_name
	`

	rows, err := config.DB.Query(query, teacherID, classID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var students []gin.H

	for rows.Next() {
		var (
			id        int
			first     string
			last      string
			email     string
			yearLevel int
		)

		if err := rows.Scan(&id, &first, &last, &email, &yearLevel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		students = append(students, gin.H{
			"student_id": id,
			"full_name":  first + " " + last,
			"email":      email,
			"year_level": yearLevel,
		})
	}

	if students == nil {
		students = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"students": students,
		"total":    len(students),
	})
}

// ===== SUBMIT GRADE =====
func TeacherSubmitGrade(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Teacher role required"})
		return
	}

	var request struct {
		ClassID   int      `json:"class_id" binding:"required"`
		StudentID int      `json:"student_id" binding:"required"`
		Prelim    *float64 `json:"prelim"`
		Midterm   *float64 `json:"midterm"`
		Finals    *float64 `json:"finals"`
		Remarks   string   `json:"remarks"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate that the class belongs to this teacher
	var subjectID int
	err := config.DB.QueryRow(`
		SELECT subject_id FROM teacher_subjects 
		WHERE id = ? AND teacher_id = ?
	`, request.ClassID, teacherID).Scan(&subjectID)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	// FIX: Validate that the student is approved AND enrolled in this class
	var studentEnrolled bool
	err = config.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM student_academic sa
			INNER JOIN students st ON st.id = sa.student_id
			WHERE sa.student_id = ? 
			AND FIND_IN_SET(?, sa.subjects) > 0
			AND st.status = 'approved'
		)
	`, request.StudentID, subjectID).Scan(&studentEnrolled)

	if err != nil || !studentEnrolled {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found, not enrolled in this class, or not yet approved by the Registrar"})
		return
	}

	// Validate grades using Filipino GWA scale (1.0 = highest, 5.0 = failed)
	if request.Prelim != nil && (*request.Prelim < 1.0 || *request.Prelim > 5.0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Prelim grade must be between 1.0 and 5.0 (Filipino GWA scale)"})
		return
	}
	if request.Midterm != nil && (*request.Midterm < 1.0 || *request.Midterm > 5.0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Midterm grade must be between 1.0 and 5.0 (Filipino GWA scale)"})
		return
	}
	if request.Finals != nil && (*request.Finals < 1.0 || *request.Finals > 5.0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Finals grade must be between 1.0 and 5.0 (Filipino GWA scale)"})
		return
	}

	// Check if grade already exists
	var gradeID int
	err = config.DB.QueryRow(`
		SELECT id FROM grades 
		WHERE student_id = ? AND subject_id = ? AND teacher_id = ?
	`, request.StudentID, subjectID, teacherID).Scan(&gradeID)

	if err == nil {
		// Update existing grade
		_, err = config.DB.Exec(`
			UPDATE grades 
			SET prelim = ?, midterm = ?, finals = ?, remarks = ?, updated_at = NOW()
			WHERE id = ?
		`, request.Prelim, request.Midterm, request.Finals, request.Remarks, gradeID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update grade"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Grade updated successfully",
			"grade_id":   gradeID,
			"student_id": request.StudentID,
			"subject_id": subjectID,
		})
	} else {
		// Insert new grade
		result, err := config.DB.Exec(`
			INSERT INTO grades (student_id, subject_id, teacher_id, prelim, midterm, finals, remarks, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		`, request.StudentID, subjectID, teacherID, request.Prelim, request.Midterm, request.Finals, request.Remarks)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit grade"})
			return
		}

		newID, _ := result.LastInsertId()

		c.JSON(http.StatusCreated, gin.H{
			"message":    "Grade submitted successfully",
			"grade_id":   newID,
			"student_id": request.StudentID,
			"subject_id": subjectID,
		})
	}
}

// ===== GET GRADES FOR A CLASS (only approved students) =====
func TeacherGetGrades(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	classID := c.Query("class_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	// Get subject_id and verify teacher owns this class
	var subjectID int
	err := config.DB.QueryRow(`
		SELECT subject_id FROM teacher_subjects 
		WHERE id = ? AND teacher_id = ?
	`, classID, teacherID).Scan(&subjectID)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	// FIX: Added INNER JOIN students + AND st.status = 'approved'
	// so only registrar-approved students show up in the grades list
	query := `
		SELECT 
			st.id,
			st.first_name,
			st.last_name,
			st.email,
			IFNULL(sa.year_level, 0),
			g.prelim,
			g.midterm,
			g.finals,
			IFNULL(g.remarks, '')
		FROM student_academic sa
		INNER JOIN students st ON st.id = sa.student_id
		LEFT JOIN grades g 
			ON g.student_id = st.id 
			AND g.subject_id = ? 
			AND g.teacher_id = ?
		WHERE FIND_IN_SET(?, sa.subjects) > 0
		AND st.status = 'approved'
		ORDER BY st.last_name, st.first_name
	`

	rows, err := config.DB.Query(query, subjectID, teacherID, subjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var students []gin.H

	for rows.Next() {
		var (
			id        int
			first     string
			last      string
			email     string
			yearLevel int
			prelim    *float64
			midterm   *float64
			finals    *float64
			remarks   string
		)

		if err := rows.Scan(&id, &first, &last, &email, &yearLevel, &prelim, &midterm, &finals, &remarks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		students = append(students, gin.H{
			"student_id": id,
			"full_name":  first + " " + last,
			"email":      email,
			"year_level": yearLevel,
			"prelim":     prelim,
			"midterm":    midterm,
			"finals":     finals,
			"remarks":    remarks,
		})
	}

	if students == nil {
		students = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"students": students,
		"total":    len(students),
	})
}

// ===== UPLOAD LESSON =====
func TeacherUploadLesson(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Teacher role required"})
		return
	}

	classID := c.PostForm("class_id")
	title := c.PostForm("title")
	description := c.PostForm("description")
	materialType := c.PostForm("type")
	dueDateStr := c.PostForm("due_date")

	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if materialType != "video" && materialType != "image" && materialType != "document" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'video', 'image', or 'document'"})
		return
	}

	var dueDate sql.NullTime
	if dueDateStr != "" {
		formats := []string{"2006-01-02 15:04:05", "2006-01-02"}
		var parsedTime time.Time
		var parseErr error
		for _, format := range formats {
			parsedTime, parseErr = time.Parse(format, dueDateStr)
			if parseErr == nil {
				break
			}
		}
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due_date format. Use YYYY-MM-DD or YYYY-MM-DD HH:MM:SS"})
			return
		}
		if parsedTime.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Due date must be in the future"})
			return
		}
		dueDate = sql.NullTime{Time: parsedTime, Valid: true}
	}

	var subjectID int
	err := config.DB.QueryRow(`SELECT subject_id FROM teacher_subjects WHERE id = ? AND teacher_id = ?`, classID, teacherID).Scan(&subjectID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	allowedTypes := map[string][]string{
		"image":    {"image/jpeg", "image/jpg", "image/png", "image/gif"},
		"video":    {"video/mp4", "video/mpeg", "video/quicktime", "video/x-msvideo", "video/webm"},
		"document": {"application/pdf"},
	}

	fileHeader, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer fileHeader.Close()

	buffer := make([]byte, 512)
	_, err = fileHeader.Read(buffer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
		return
	}

	contentType := http.DetectContentType(buffer)

	validType := false
	for _, t := range allowedTypes[materialType] {
		if t == contentType {
			validType = true
			break
		}
	}
	if !validType {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid file type. Detected: %s, allowed: %v", contentType, allowedTypes[materialType]),
		})
		return
	}

	maxSize := int64(10 * 1024 * 1024)
	if materialType == "video" {
		maxSize = int64(100 * 1024 * 1024)
	}
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File too large. Max size: %dMB", maxSize/(1024*1024))})
		return
	}

	uploadDir := "./uploads/lessons"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%d_%s%s", teacherID, time.Now().Unix(), strings.ReplaceAll(uuid.New().String(), "-", ""), fileExt)
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	result, err := config.DB.Exec(`
		INSERT INTO lesson_materials
		(teacher_id, subject_id, class_id, title, description, type, file_name, file_path, file_size, due_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`, teacherID, subjectID, classID, title, description, materialType, fileName, filePath, file.Size, dueDate)
	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save lesson material: " + err.Error()})
		return
	}

	materialID, _ := result.LastInsertId()

	response := gin.H{
		"message":     "Lesson material uploaded successfully",
		"material_id": materialID,
		"title":       title,
		"type":        materialType,
		"file_name":   fileName,
		"file_size":   file.Size,
	}
	if dueDate.Valid {
		response["due_date"] = dueDate.Time.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusCreated, response)
}

// ===== GET LESSON MATERIALS FOR A CLASS =====
func TeacherGetLessonMaterials(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	classID := c.Query("class_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	var exists bool
	err := config.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM teacher_subjects WHERE id = ? AND teacher_id = ?)
	`, classID, teacherID).Scan(&exists)

	if err != nil || !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	query := `
		SELECT 
			lm.id,
			lm.title,
			lm.description,
			lm.type,
			lm.file_name,
			lm.file_path,
			lm.file_size,
			lm.due_date,
			lm.created_at,
			COUNT(DISTINCT ss.id) as total_submissions,
			COUNT(DISTINCT CASE WHEN ss.status = 'late' THEN ss.id END) as late_submissions
		FROM lesson_materials lm
		LEFT JOIN student_submissions ss ON lm.id = ss.material_id
		WHERE lm.teacher_id = ? AND lm.class_id = ?
		GROUP BY lm.id
		ORDER BY lm.created_at DESC
	`

	rows, err := config.DB.Query(query, teacherID, classID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var materials []gin.H

	for rows.Next() {
		var (
			id               int
			title            string
			description      string
			matType          string
			fileName         string
			filePath         string
			fileSize         int64
			dueDate          sql.NullString
			createdAt        string
			totalSubmissions int
			lateSubmissions  int
		)

		if err := rows.Scan(&id, &title, &description, &matType, &fileName, &filePath, &fileSize,
			&dueDate, &createdAt, &totalSubmissions, &lateSubmissions); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		material := gin.H{
			"id":                id,
			"title":             title,
			"description":       description,
			"type":              matType,
			"file_name":         fileName,
			"file_size":         fileSize,
			"created_at":        createdAt,
			"total_submissions": totalSubmissions,
			"late_submissions":  lateSubmissions,
			"download_url":      "/api/lessons/download/" + strconv.Itoa(id),
		}

		if dueDate.Valid {
			parsedTime, err := time.Parse("2006-01-02 15:04:05", dueDate.String)
			if err == nil {
				material["due_date"] = parsedTime.Format("2006-01-02 15:04:05")
				material["is_overdue"] = time.Now().After(parsedTime)
			} else {
				material["due_date"] = dueDate.String
				material["is_overdue"] = false
			}
		} else {
			material["due_date"] = nil
			material["is_overdue"] = false
		}

		materials = append(materials, material)
	}

	if materials == nil {
		materials = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"materials": materials,
		"total":     len(materials),
	})
}

// ===== GET SUBMISSIONS FOR A LESSON =====
func TeacherGetSubmissions(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	materialID := c.Param("id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var exists bool
	err := config.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM lesson_materials WHERE id = ? AND teacher_id = ?)
	`, materialID, teacherID).Scan(&exists)

	if err != nil || !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Material not found or not owned by you"})
		return
	}

	// FIX: Added INNER JOIN students + AND st.status = 'approved'
	// so submissions from non-approved students are excluded
	query := `
		SELECT 
			ss.id,
			st.id as student_id,
			st.first_name,
			st.last_name,
			st.email,
			ss.file_name,
			ss.file_path,
			ss.file_size,
			ss.submitted_at,
			ss.status,
			IFNULL(ss.remarks, '')
		FROM student_submissions ss
		INNER JOIN students st ON st.id = ss.student_id
		WHERE ss.material_id = ?
		AND st.status = 'approved'
		ORDER BY ss.submitted_at DESC
	`

	rows, err := config.DB.Query(query, materialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var submissions []gin.H

	for rows.Next() {
		var (
			id        int
			studentID int
			firstName string
			lastName  string
			email     string
			fileName  string
			filePath  string
			fileSize  int64
			submitted string
			status    string
			remarks   sql.NullString
		)

		if err := rows.Scan(&id, &studentID, &firstName, &lastName, &email,
			&fileName, &filePath, &fileSize, &submitted, &status, &remarks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		remarksValue := ""
		if remarks.Valid {
			remarksValue = remarks.String
		}

		cleanPath := strings.ReplaceAll(filePath, "\\", "/")
		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}

		submissions = append(submissions, gin.H{
			"id":           id,
			"student_id":   studentID,
			"student_name": firstName + " " + lastName,
			"email":        email,
			"file_name":    fileName,
			"file_path":    cleanPath,
			"download_url": cleanPath,
			"file_size":    fileSize,
			"submitted_at": submitted,
			"status":       status,
			"remarks":      remarksValue,
		})
	}

	if submissions == nil {
		submissions = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"total":       len(submissions),
	})
}

// ===== DELETE LESSON MATERIAL =====
func TeacherDeleteLessonMaterial(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	materialID := c.Param("id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var filePath string
	err := config.DB.QueryRow(`
		SELECT file_path FROM lesson_materials 
		WHERE id = ? AND teacher_id = ?
	`, materialID, teacherID).Scan(&filePath)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found or not owned by you"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var submissionPaths []string
	rows, _ := config.DB.Query(`SELECT file_path FROM student_submissions WHERE material_id = ?`, materialID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			rows.Scan(&path)
			submissionPaths = append(submissionPaths, path)
		}
	}

	_, err = config.DB.Exec(`DELETE FROM lesson_materials WHERE id = ?`, materialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete material"})
		return
	}

	os.Remove(filePath)
	for _, path := range submissionPaths {
		os.Remove(path)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lesson material deleted successfully"})
}

// ===== UPDATE LESSON MATERIAL =====
func TeacherUpdateLesson(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	materialID := c.Param("id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var exists bool
	err := config.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM lesson_materials WHERE id = ? AND teacher_id = ?)
	`, materialID, teacherID).Scan(&exists)

	if err != nil || !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Material not found or not owned by you"})
		return
	}

	title := c.PostForm("title")
	description := c.PostForm("description")
	dueDateStr := c.PostForm("due_date")

	var dueDate sql.NullTime
	if dueDateStr != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
		}

		var parsedTime time.Time
		var parseErr error

		for _, format := range formats {
			parsedTime, parseErr = time.Parse(format, dueDateStr)
			if parseErr == nil {
				break
			}
		}

		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due_date format"})
			return
		}

		dueDate = sql.NullTime{Time: parsedTime, Valid: true}
	}

	_, err = config.DB.Exec(`
		UPDATE lesson_materials 
		SET title = ?, description = ?, due_date = ?, updated_at = NOW()
		WHERE id = ?
	`, title, description, dueDate, materialID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update material"})
		return
	}

	response := gin.H{
		"message": "Lesson material updated successfully",
	}

	if dueDate.Valid {
		response["due_date"] = dueDate.Time.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusOK, response)
}

// ===== REVIEW SUBMISSION =====
func TeacherReviewSubmission(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	submissionID := c.Param("id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Teacher role required"})
		return
	}

	var request struct {
		Status  string   `json:"status" binding:"required"`
		Grade   *float64 `json:"grade"`
		Remarks string   `json:"remarks"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if request.Status != "accepted" && request.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'accepted' or 'rejected'"})
		return
	}

	if request.Grade != nil && (*request.Grade < 0 || *request.Grade > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Grade must be between 0 and 100"})
		return
	}

	// FIX: Also verify that the submitting student is approved
	var materialID int
	var studentID int
	var currentStatus string

	err := config.DB.QueryRow(`
		SELECT ss.material_id, ss.student_id, ss.status
		FROM student_submissions ss
		INNER JOIN lesson_materials lm ON ss.material_id = lm.id
		INNER JOIN students st ON st.id = ss.student_id
		WHERE ss.id = ? AND lm.teacher_id = ?
		AND st.status = 'approved'
	`, submissionID, teacherID).Scan(&materialID, &studentID, &currentStatus)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found, not owned by you, or student not yet approved"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	_, err = config.DB.Exec(`
		UPDATE student_submissions
		SET status = ?, remarks = ?, reviewed_at = NOW()
		WHERE id = ?
	`, request.Status, request.Remarks, submissionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update submission"})
		return
	}

	if request.Grade != nil && request.Status == "accepted" {
		var subjectID int
		err = config.DB.QueryRow(`
			SELECT subject_id FROM lesson_materials WHERE id = ?
		`, materialID).Scan(&subjectID)

		if err == nil {
			var gradeID int
			err = config.DB.QueryRow(`
				SELECT id FROM grades 
				WHERE student_id = ? AND subject_id = ? AND teacher_id = ?
			`, studentID, subjectID, teacherID).Scan(&gradeID)

			if err == nil {
				_, _ = config.DB.Exec(`
					UPDATE grades 
					SET remarks = CONCAT(IFNULL(remarks, ''), '\nSubmission Grade: ', ?), updated_at = NOW()
					WHERE id = ?
				`, *request.Grade, gradeID)
			}
		}
	}

	fmt.Printf("✅ Submission %s reviewed: status=%s, grade=%v\n", submissionID, request.Status, request.Grade)

	response := gin.H{
		"message":       "Submission reviewed successfully",
		"submission_id": submissionID,
		"status":        request.Status,
	}

	if request.Grade != nil {
		response["grade"] = *request.Grade
	}

	c.JSON(http.StatusOK, response)
}

// ===== GET PENDING SUBMISSIONS (only from approved students) =====
func TeacherGetPendingSubmissions(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	classID := c.Query("class_id")

	var query string
	var args []interface{}

	if classID != "" {
		// FIX: Added AND st.status = 'approved'
		query = `
			SELECT 
				ss.id,
				st.id as student_id,
				st.first_name,
				st.last_name,
				st.email,
				lm.id as material_id,
				lm.title as lesson_title,
				s.subject_name,
				ss.file_name,
				ss.file_path,
				ss.file_size,
				ss.submitted_at,
				ss.status,
				IFNULL(ss.remarks, '')
			FROM student_submissions ss
			INNER JOIN students st ON st.id = ss.student_id
			INNER JOIN lesson_materials lm ON lm.id = ss.material_id
			INNER JOIN subjects s ON lm.subject_id = s.id
			WHERE lm.teacher_id = ? 
			AND lm.class_id = ?
			AND ss.status IN ('pending', 'on-time', 'late')
			AND st.status = 'approved'
			ORDER BY ss.submitted_at DESC
		`
		args = []interface{}{teacherID, classID}
	} else {
		// FIX: Added AND st.status = 'approved'
		query = `
			SELECT 
				ss.id,
				st.id as student_id,
				st.first_name,
				st.last_name,
				st.email,
				lm.id as material_id,
				lm.title as lesson_title,
				s.subject_name,
				ss.file_name,
				ss.file_path,
				ss.file_size,
				ss.submitted_at,
				ss.status,
				IFNULL(ss.remarks, '')
			FROM student_submissions ss
			INNER JOIN students st ON st.id = ss.student_id
			INNER JOIN lesson_materials lm ON lm.id = ss.material_id
			INNER JOIN subjects s ON lm.subject_id = s.id
			WHERE lm.teacher_id = ?
			AND ss.status IN ('pending', 'on-time', 'late')
			AND st.status = 'approved'
			ORDER BY ss.submitted_at DESC
		`
		args = []interface{}{teacherID}
	}

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var submissions []gin.H

	for rows.Next() {
		var (
			id                         int
			studentID                  int
			firstName, lastName, email string
			materialID                 int
			lessonTitle, subjectName   string
			fileName, filePath         string
			fileSize                   int64
			submittedAt                string
			status                     string
			remarks                    string
		)

		if err := rows.Scan(&id, &studentID, &firstName, &lastName, &email,
			&materialID, &lessonTitle, &subjectName, &fileName, &filePath, &fileSize,
			&submittedAt, &status, &remarks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		cleanPath := strings.ReplaceAll(filePath, "\\", "/")
		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}

		submissions = append(submissions, gin.H{
			"submission_id": id,
			"student_id":    studentID,
			"student_name":  firstName + " " + lastName,
			"email":         email,
			"material_id":   materialID,
			"lesson_title":  lessonTitle,
			"subject":       subjectName,
			"file_name":     fileName,
			"file_path":     cleanPath,
			"file_size":     fileSize,
			"submitted_at":  submittedAt,
			"status":        status,
			"remarks":       remarks,
		})
	}

	if submissions == nil {
		submissions = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"total":       len(submissions),
	})
}

// ===== POST ANNOUNCEMENT =====
func TeacherPostAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	classID := c.PostForm("class_id")
	title := c.PostForm("title")
	content := c.PostForm("content")

	if classID == "" || title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id, title, and content are required"})
		return
	}

	var subjectID int
	err := config.DB.QueryRow(`
        SELECT subject_id FROM teacher_subjects WHERE id = ? AND teacher_id = ?
    `, classID, teacherID).Scan(&subjectID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	var imageName, imagePath sql.NullString
	var imageSize sql.NullInt64

	file, err := c.FormFile("image")
	if err == nil {
		allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"}

		fileHeader, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
			return
		}
		defer fileHeader.Close()

		buffer := make([]byte, 512)
		fileHeader.Read(buffer)
		contentType := http.DetectContentType(buffer)

		valid := false
		for _, t := range allowedTypes {
			if t == contentType {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image must be JPEG, PNG, GIF, or WebP"})
			return
		}
		if file.Size > int64(10*1024*1024) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image too large. Max 10MB"})
			return
		}

		uploadDir := "./uploads/announcements"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}

		fileExt := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("%d_%d_%s%s", teacherID, time.Now().Unix(),
			strings.ReplaceAll(uuid.New().String(), "-", ""), fileExt)
		savedPath := filepath.Join(uploadDir, fileName)

		if err := c.SaveUploadedFile(file, savedPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		imageName = sql.NullString{String: file.Filename, Valid: true}
		imagePath = sql.NullString{String: savedPath, Valid: true}
		imageSize = sql.NullInt64{Int64: file.Size, Valid: true}
	}

	result, err := config.DB.Exec(`
        INSERT INTO announcements (teacher_id, class_id, subject_id, title, content, image_name, image_path, image_size, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
    `, teacherID, classID, subjectID, title, content, imageName, imagePath, imageSize)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post announcement: " + err.Error()})
		return
	}

	newID, _ := result.LastInsertId()

	response := gin.H{
		"message":         "Announcement posted successfully",
		"announcement_id": newID,
		"title":           title,
	}
	if imagePath.Valid {
		response["image_url"] = "/" + strings.ReplaceAll(imagePath.String, "\\", "/")[2:]
	}

	c.JSON(http.StatusCreated, response)
}

// ===== GET ANNOUNCEMENTS FOR A CLASS =====
func TeacherGetAnnouncements(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	classID := c.Query("class_id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if classID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	var exists bool
	err := config.DB.QueryRow(`
        SELECT EXISTS(SELECT 1 FROM teacher_subjects WHERE id = ? AND teacher_id = ?)
    `, classID, teacherID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Class not found or not assigned to you"})
		return
	}

	rows, err := config.DB.Query(`
        SELECT id, title, content, image_name, image_path, image_size, created_at
        FROM announcements
        WHERE teacher_id = ? AND class_id = ?
        ORDER BY created_at DESC
    `, teacherID, classID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var announcements []gin.H
	for rows.Next() {
		var (
			id        int
			title     string
			content   string
			imageName sql.NullString
			imagePath sql.NullString
			imageSize sql.NullInt64
			createdAt string
		)
		if err := rows.Scan(&id, &title, &content, &imageName, &imagePath, &imageSize, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		item := gin.H{
			"id":         id,
			"title":      title,
			"content":    content,
			"created_at": createdAt,
			"image_url":  nil,
			"image_size": nil,
		}
		if imagePath.Valid {
			cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
			if strings.HasPrefix(cleanPath, "./") {
				cleanPath = cleanPath[2:]
			}
			item["image_url"] = "/" + cleanPath
			item["image_size"] = imageSize.Int64
		}
		announcements = append(announcements, item)
	}

	if announcements == nil {
		announcements = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"announcements": announcements,
		"total":         len(announcements),
	})
}

// ===== DELETE ANNOUNCEMENT =====
func TeacherDeleteAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetInt("user_id")
	announcementID := c.Param("id")

	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var filePath sql.NullString
	err := config.DB.QueryRow(`
        SELECT image_path FROM announcements WHERE id = ? AND teacher_id = ?
    `, announcementID, teacherID).Scan(&filePath)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Announcement not found or not owned by you"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = config.DB.Exec(`DELETE FROM announcements WHERE id = ?`, announcementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete announcement"})
		return
	}

	if filePath.Valid {
		os.Remove(filePath.String)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Announcement deleted successfully"})
}
