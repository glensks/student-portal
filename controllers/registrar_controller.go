package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"student-portal/config"
	"student-portal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

/* =======================
STUDENT STRUCTS
======================= */

type StudentSubject struct {
	SubjectCode string `json:"subject_code"`
	SubjectName string `json:"subject_name"`
}

type Student struct {
	ID                int              `json:"-"`
	StudentID         string           `json:"student_id"`
	FirstName         string           `json:"first_name"`
	LastName          string           `json:"last_name"`
	FullName          string           `json:"full_name"`
	Course            string           `json:"course"`
	YearLevel         string           `json:"year_level"`
	Semester          string           `json:"semester"` // ✅ INCLUDED
	ScholarshipStatus string           `json:"scholarship_status"`
	Status            string           `json:"status"`
	Subjects          []StudentSubject `json:"subjects"`
	TotalUnits        int              `json:"total_units"`
}

/* =======================
PAYMENT STRUCTS
======================= */

type PaymentFee struct {
	FeeName string `json:"fee_name"`
	Amount  int    `json:"amount"`
}

type PaymentDetails struct {
	PaymentID      int          `json:"payment_id"`
	StudentID      string       `json:"student_id"`
	FullName       string       `json:"full_name"`
	Semester       string       `json:"semester"` // ✅ INCLUDED
	Scholarship    string       `json:"scholarship_status"`
	TotalUnits     int          `json:"total_units"`
	OtherFees      []PaymentFee `json:"other_fees"`
	OtherFeesTotal int          `json:"other_fees_total"`
	TotalAmount    int          `json:"total_amount"`
	TotalPaid      int          `json:"total_paid"`
	Status         string       `json:"status"`
}

type ApproveWithAssessmentRequest struct {
	StudentID  string       `json:"student_id"`
	Semester   string       `json:"semester"`    // ✅ REQUIRED
	SchoolYear string       `json:"school_year"` // ✅ REQUIRED
	OtherFees  []PaymentFee `json:"other_fees"`
}

/* =======================
STUDENT LISTING
======================= */

func RegistrarGetStudentsByStatus(c *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))

	rows, err := config.DB.Query(`
		SELECT 
			st.id,
			st.student_id,
			st.first_name,
			st.last_name,
			IFNULL(c.course_name,''),
			IFNULL(sa.year_level,''),
			IFNULL(sa.scholarship_status,''),
			st.status,
			IFNULL(sa.total_units,0),
			IFNULL(sa.semester,''),     -- ✅ SEMESTER
			IFNULL(sa.subjects,'')
		FROM students st
		LEFT JOIN student_academic sa ON sa.student_id = st.id
		LEFT JOIN courses c ON c.id = sa.course
		WHERE LOWER(st.status) = ?
	`, status)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var students []Student

	for rows.Next() {
		var s Student
		var subjectsStr string

		err := rows.Scan(
			&s.ID,
			&s.StudentID,
			&s.FirstName,
			&s.LastName,
			&s.Course,
			&s.YearLevel,
			&s.ScholarshipStatus,
			&s.Status,
			&s.TotalUnits,
			&s.Semester, // ✅ SCANNED
			&subjectsStr,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		s.FullName = s.FirstName + " " + s.LastName
		s.Subjects = []StudentSubject{}

		if strings.TrimSpace(subjectsStr) != "" {
			ids := strings.Split(subjectsStr, ",")
			for _, id := range ids {
				var subj StudentSubject
				config.DB.QueryRow(
					`SELECT code, subject_name FROM subjects WHERE id=?`,
					strings.TrimSpace(id),
				).Scan(&subj.SubjectCode, &subj.SubjectName)

				s.Subjects = append(s.Subjects, subj)
			}
		}

		students = append(students, s)
	}

	c.JSON(200, gin.H{"students": students})
}

/* =======================
APPROVE WITH ASSESSMENT
======================= */

func RegistrarApproveWithAssessment(c *gin.Context) {
	var req ApproveWithAssessmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var studentDBID, totalUnits int
	var scholarshipStatus, studentEmail string

	// ===== GET STUDENT INFO =====
	err = tx.QueryRow(`
		SELECT st.id,
		       IFNULL(sa.total_units,0),
		       IFNULL(sa.scholarship_status,''),
		       st.email
		FROM students st
		LEFT JOIN student_academic sa ON sa.student_id = st.id
		WHERE st.student_id = ?
	`, req.StudentID).Scan(
		&studentDBID,
		&totalUnits,
		&scholarshipStatus,
		&studentEmail,
	)

	if err != nil {
		tx.Rollback()
		c.JSON(404, gin.H{"error": "Student not found"})
		return
	}

	// ===== COMPUTE TUITION =====
	tuition := 800 * totalUnits
	if strings.ToLower(scholarshipStatus) == "scholar" {
		tuition = 500 * totalUnits
	}

	// ===== CREATE PAYMENT RECORD =====
	res, err := tx.Exec(`
		INSERT INTO student_payments
		    (student_id, total_amount, amount_paid, status, semester, school_year)
		VALUES (?, 0, 0, 'unpaid', ?, ?)
	`, studentDBID, req.Semester, req.SchoolYear)

	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	paymentID, _ := res.LastInsertId()

	// ===== OTHER FEES =====
	otherTotal := 0

	for _, fee := range req.OtherFees {
		if fee.Amount <= 0 {
			continue
		}

		_, err := tx.Exec(`
			INSERT INTO payment_fees (payment_id, fee_name, amount)
			VALUES (?, ?, ?)
		`, paymentID, fee.FeeName, fee.Amount)

		if err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		otherTotal += fee.Amount
	}

	// ===== FINAL TOTAL =====
	totalAmount := tuition + otherTotal

	_, err = tx.Exec(`
		UPDATE student_payments
		SET total_amount = ?
		WHERE id = ?
	`, totalAmount, paymentID)

	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// ===== APPROVE STUDENT =====
	_, err = tx.Exec(`
		UPDATE students
		SET status='approved'
		WHERE id=?
	`, studentDBID)

	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	// ===== EMAIL =====
	emailBody := fmt.Sprintf(`
	<h2>Enrollment Approved</h2>
	<p><b>Semester:</b> %s</p>
	<p><b>Total Units:</b> %d</p>
	<p><b>Tuition:</b> ₱%d</p>
	<p><b>Other Fees:</b> ₱%d</p>
	<p><b>Total Amount Payable:</b> ₱%d</p>
	<p>Status: UNPAID</p>
	`,
		req.Semester,
		totalUnits,
		tuition,
		otherTotal,
		totalAmount,
	)

	go utils.SendEmail(
		studentEmail,
		"Student Billing Statement",
		emailBody,
	)

	// ===== RESPONSE =====
	c.JSON(200, gin.H{
		"message":      "Student approved with assessment",
		"payment_id":   paymentID,
		"semester":     req.Semester, // ✅ RETURNED
		"school_year":  req.SchoolYear,
		"tuition":      tuition,
		"other_fees":   otherTotal,
		"total_amount": totalAmount,
		"amount_paid":  0,
		"status":       "unpaid",
	})
}

// ===== POST REGISTRAR ANNOUNCEMENT =====
func RegistrarPostAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	registrarID := c.GetInt("user_id")

	if role != "registrar" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")
	targetAudience := c.PostForm("target_audience")

	if title == "" || content == "" || targetAudience == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, content, and target_audience are required"})
		return
	}

	// Validate target_audience
	validAudiences := map[string]bool{
		"all":      true,
		"pending":  true,
		"approved": true,
		"enrolled": true,
	}
	if !validAudiences[targetAudience] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_audience. Must be: all, pending, approved, or enrolled"})
		return
	}

	// Handle optional image
	var imageName, imagePath sql.NullString
	var imageSize sql.NullInt64

	file, err := c.FormFile("image")
	if err == nil {
		// Image was provided — validate it
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
		if file.Size > int64(10*1024*1024) { // 10MB max
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image too large. Max 10MB"})
			return
		}

		uploadDir := "./uploads/registrar-announcements"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}

		fileExt := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("%d_%d_%s%s", registrarID, time.Now().Unix(),
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
        INSERT INTO registrar_announcements (registrar_id, title, content, target_audience, image_name, image_path, image_size, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
    `, registrarID, title, content, targetAudience, imageName, imagePath, imageSize)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post announcement: " + err.Error()})
		return
	}

	newID, _ := result.LastInsertId()

	response := gin.H{
		"message":         "Announcement posted successfully",
		"announcement_id": newID,
		"title":           title,
		"target_audience": targetAudience,
	}
	if imagePath.Valid {
		cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}
		response["image_url"] = "/" + cleanPath
	}

	c.JSON(http.StatusCreated, response)
}

// ===== GET REGISTRAR ANNOUNCEMENTS =====
func RegistrarGetAnnouncements(c *gin.Context) {
	role := c.GetString("role")

	if role != "registrar" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	targetAudience := c.Query("target_audience") // Optional filter

	query := `
        SELECT id, title, content, target_audience, image_name, image_path, image_size, created_at
        FROM registrar_announcements
    `

	args := []interface{}{}

	if targetAudience != "" {
		query += ` WHERE target_audience = ?`
		args = append(args, targetAudience)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var announcements []gin.H
	for rows.Next() {
		var (
			id             int
			title          string
			content        string
			targetAudience string
			imageName      sql.NullString
			imagePath      sql.NullString
			imageSize      sql.NullInt64
			createdAt      string
		)
		if err := rows.Scan(&id, &title, &content, &targetAudience, &imageName, &imagePath, &imageSize, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		item := gin.H{
			"id":              id,
			"title":           title,
			"content":         content,
			"target_audience": targetAudience,
			"created_at":      createdAt,
			"image_url":       nil,
			"image_size":      nil,
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

// ===== DELETE REGISTRAR ANNOUNCEMENT =====
func RegistrarDeleteAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	registrarID := c.GetInt("user_id")
	announcementID := c.Param("id")

	if role != "registrar" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var filePath sql.NullString
	err := config.DB.QueryRow(`
        SELECT image_path FROM registrar_announcements WHERE id = ? AND registrar_id = ?
    `, announcementID, registrarID).Scan(&filePath)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Announcement not found or not owned by you"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = config.DB.Exec(`DELETE FROM registrar_announcements WHERE id = ?`, announcementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete announcement"})
		return
	}

	if filePath.Valid {
		os.Remove(filePath.String)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Announcement deleted successfully"})
}
