package controllers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"student-portal/config"
	"time"

	"github.com/gin-gonic/gin"
)

// ===================== STUDENT VIEW ALL PAYMENTS =====================

// StudentGetPaymentsMe returns payments for the logged-in student (from JWT)
func StudentGetPaymentsMe(c *gin.Context) {

	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student not authenticated"})
		return
	}

	// ===== STUDENT + ACADEMIC (UPDATED: Added first_name, last_name, profile_picture)
	var (
		studentDBID       int
		firstName         string
		lastName          string
		profilePicture    string // ⬅️ NEW
		yearLevel         int
		totalUnits        int
		scholarshipStatus string
		subjectIDs        string
	)

	err := config.DB.QueryRow(`
		SELECT 
			st.id,
			st.first_name,
			st.last_name,
			IFNULL(st.profile_picture, ''),    -- ⬅️ NEW
			IFNULL(sa.year_level, 0),
			IFNULL(sa.total_units, 0),
			IFNULL(sa.scholarship_status, ''),
			IFNULL(sa.subjects, '')
		FROM students st
		LEFT JOIN student_academic sa ON sa.student_id = st.id
		WHERE st.student_id = ?
	`, studentStrID).Scan(
		&studentDBID,
		&firstName,
		&lastName,
		&profilePicture, // ⬅️ NEW
		&yearLevel,
		&totalUnits,
		&scholarshipStatus,
		&subjectIDs,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Construct full name
	studentName := firstName + " " + lastName

	// ===== SUBJECTS
	var subjects []gin.H
	if subjectIDs != "" {
		rows, _ := config.DB.Query(`
			SELECT subject_name FROM subjects WHERE FIND_IN_SET(id, ?)
		`, subjectIDs)
		defer rows.Close()

		for rows.Next() {
			var name string
			rows.Scan(&name)
			subjects = append(subjects, gin.H{"subject_name": name})
		}
	}

	// ===== PAYMENTS
	rows, err := config.DB.Query(`
		SELECT 
			id,
			IFNULL(payment_method, ''),
			IFNULL(semester, ''),
			IFNULL(school_year, ''),
			IFNULL(total_amount, 0),
			IFNULL(amount_paid, 0),
			IFNULL(status, 'unpaid')
		FROM student_payments
		WHERE student_id = ?
		ORDER BY id DESC
	`, studentDBID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payments"})
		return
	}
	defer rows.Close()

	var payments []gin.H

	for rows.Next() {
		var (
			id, totalAmount, amountPaid          int
			method, semester, schoolYear, status string
		)

		rows.Scan(
			&id,
			&method,
			&semester,
			&schoolYear,
			&totalAmount,
			&amountPaid,
			&status,
		)

		// FEES (DISPLAY ONLY)
		var fees []gin.H
		feeRows, _ := config.DB.Query(`
			SELECT fee_name, amount FROM payment_fees WHERE payment_id = ?
		`, id)

		for feeRows.Next() {
			var n string
			var a int
			feeRows.Scan(&n, &a)
			fees = append(fees, gin.H{
				"fee_name": n,
				"amount":   a,
			})
		}
		feeRows.Close()

		payments = append(payments, gin.H{
			"payment_id":   id,
			"semester":     semester,
			"school_year":  schoolYear,
			"total_amount": totalAmount,
			"amount_paid":  amountPaid,
			"remaining":    totalAmount - amountPaid,
			"status":       status,
			"other_fees":   fees,
		})
	}

	// Normalize profile picture path
	cleanProfilePic := strings.ReplaceAll(profilePicture, "\\", "/")
	if strings.HasPrefix(cleanProfilePic, "./") {
		cleanProfilePic = cleanProfilePic[2:]
	}

	// ===== RESPONSE (UPDATED: Added student_name and profile_picture)
	c.JSON(http.StatusOK, gin.H{
		"student_name":       studentName,
		"profile_picture":    cleanProfilePic, // ⬅️ NEW
		"student_id":         studentStrID,
		"year_level":         yearLevel,
		"total_units":        totalUnits,
		"scholarship_status": scholarshipStatus,
		"subjects":           subjects,
		"payments":           payments,
	})
}

// ===================== STUDENT PAY BILL (PENDING APPROVAL) =====================

func StudentPayBill(c *gin.Context) {
	var req struct {
		PaymentID     int    `json:"payment_id"`
		Amount        int    `json:"amount"`
		PaymentMethod string `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	if req.PaymentMethod == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_method required"})
		return
	}

	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var studentDBID int
	err := config.DB.QueryRow(`SELECT id FROM students WHERE student_id = ?`, studentStrID).Scan(&studentDBID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student not found"})
		return
	}

	var totalAmount, amountPaid int
	err = config.DB.QueryRow(`
		SELECT total_amount, amount_paid
		FROM student_payments
		WHERE id = ? AND student_id = ?
	`, req.PaymentID, studentDBID).Scan(&totalAmount, &amountPaid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	newAmountPaid := amountPaid + req.Amount
	if newAmountPaid > totalAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment exceeds remaining balance"})
		return
	}

	// SET STATUS TO PENDING FOR CASHIER APPROVAL
	_, err = config.DB.Exec(`
		UPDATE student_payments
		SET
			amount_paid = ?,
			status = 'pending',
			payment_method = ?
		WHERE id = ? AND student_id = ?
	`, newAmountPaid, req.PaymentMethod, req.PaymentID, studentDBID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "payment submitted, waiting for cashier approval",
		"amount_paid": newAmountPaid,
		"remaining":   totalAmount - newAmountPaid,
		"status":      "pending",
	})
}

// ===================== STUDENT DOWNPAYMENT (PENDING APPROVAL) =====================

func StudentDownPayment(c *gin.Context) {
	var req struct {
		PaymentID     int    `json:"payment_id"`
		DownPayment   int    `json:"down_payment"`
		PaymentMethod string `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.DownPayment <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid downpayment amount"})
		return
	}

	if req.PaymentMethod == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_method required"})
		return
	}

	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// get student db id
	var studentDBID int
	err := config.DB.QueryRow(
		`SELECT id FROM students WHERE student_id = ?`,
		studentStrID,
	).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "student not found"})
		return
	}

	// get payment info
	var totalAmount, amountPaid int
	err = config.DB.QueryRow(`
		SELECT total_amount, amount_paid
		FROM student_payments
		WHERE id = ? AND student_id = ?
	`, req.PaymentID, studentDBID).Scan(&totalAmount, &amountPaid)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	if amountPaid > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment already made"})
		return
	}

	if req.DownPayment >= totalAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "downpayment must be less than total tuition"})
		return
	}

	remaining := totalAmount - req.DownPayment

	// ⭐ SET TO PENDING - WAIT FOR CASHIER APPROVAL
	_, err = config.DB.Exec(`
		UPDATE student_payments
		SET
			amount_paid = ?,
			status = 'pending',
			payment_method = ?
		WHERE id = ? AND student_id = ?
	`, req.DownPayment, req.PaymentMethod, req.PaymentID, studentDBID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit downpayment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "downpayment submitted, waiting for cashier approval",
		"total_amount": totalAmount,
		"paid":         req.DownPayment,
		"remaining":    remaining,
		"status":       "pending",
	})
}

// ===================== STUDENT GET INSTALLMENTS =====================

func StudentGetInstallments(c *gin.Context) {

	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// ================= STUDENT ID =================
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// ================= PAYMENT ID =================
	paymentIDStr := c.Query("payment_id")
	paymentID, err := strconv.Atoi(paymentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment_id"})
		return
	}

	// ================= PAYMENT INFO =================
	var (
		totalAmount float64
		amountPaid  float64
		remaining   float64
		status      string
	)

	err = config.DB.QueryRow(`
		SELECT 
			total_amount,
			amount_paid,
			(total_amount - amount_paid),
			status
		FROM student_payments
		WHERE id = ? AND student_id = ?
	`, paymentID, studentDBID).Scan(
		&totalAmount,
		&amountPaid,
		&remaining,
		&status,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	// ================= ACADEMIC (FIXED) =================
	var totalUnits int
	var subjectIDs string

	err = config.DB.QueryRow(`
		SELECT 
			IFNULL(total_units, 0),
			IFNULL(subjects, '')
		FROM student_academic
		WHERE student_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, studentDBID).Scan(&totalUnits, &subjectIDs)

	if err != nil {
		totalUnits = 0
		subjectIDs = ""
	}

	// ================= SUBJECTS =================
	subjects := []gin.H{}

	if subjectIDs != "" {
		rows, err := config.DB.Query(`
			SELECT subject_name
			FROM subjects
			WHERE FIND_IN_SET(id, ?)
			ORDER BY subject_name ASC
		`, subjectIDs)

		if err == nil {
			defer rows.Close()

			for rows.Next() {
				var name string
				rows.Scan(&name)

				subjects = append(subjects, gin.H{
					"subject_name": name,
				})
			}
		}
	}

	// ================= NOT APPROVED =================
	if status != "approved" && status != "partial" {
		c.JSON(http.StatusOK, gin.H{
			"payment_id":   paymentID,
			"total_amount": totalAmount,
			"amount_paid":  amountPaid,
			"remaining":    remaining,
			"status":       status,
			"total_units":  totalUnits,
			"subjects":     subjects,
			"installments": []gin.H{},
		})
		return
	}

	// ================= INSTALLMENTS =================
	instRows, err := config.DB.Query(`
		SELECT term, amount, status, paid_at
		FROM student_installments
		WHERE payment_id = ?
		ORDER BY FIELD(term,'prelim','midterm','finals')
	`, paymentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load installments"})
		return
	}
	defer instRows.Close()

	installments := []gin.H{}

	for instRows.Next() {
		var term, instStatus string
		var amount float64
		var paidAt *string

		instRows.Scan(&term, &amount, &instStatus, &paidAt)

		installments = append(installments, gin.H{
			"term":    term,
			"amount":  amount,
			"status":  instStatus,
			"paid_at": paidAt,
		})
	}

	// ================= RESPONSE =================
	c.JSON(http.StatusOK, gin.H{
		"payment_id":   paymentID,
		"total_amount": totalAmount,
		"amount_paid":  amountPaid,
		"remaining":    remaining,
		"status":       status,
		"total_units":  totalUnits,
		"subjects":     subjects,
		"installments": installments,
	})
}

type StudentSchedule struct {
	Subject string `json:"subject"`
	Day     string `json:"day"`
	Time    string `json:"time"`
	Room    string `json:"room"`
	Teacher string `json:"teacher"`
}

func StudentGetSchedule(c *gin.Context) {

	// ================= AUTH =================
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// ================= GET students.id =================
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student not found"})
		return
	}

	// ================= GET ACADEMIC RECORD =================
	var subjects string
	var courseID int

	err = config.DB.QueryRow(`
		SELECT subjects, course
		FROM student_academic
		WHERE student_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, studentDBID).Scan(&subjects, &courseID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "academic record not found"})
		return
	}

	// ================= GET SCHEDULE =================
	rows, err := config.DB.Query(`
		SELECT
			sub.subject_name,
			ts.day,
			CONCAT(
				TIME_FORMAT(ts.time_start, '%h:%i %p'),
				' - ',
				TIME_FORMAT(ts.time_end, '%h:%i %p')
			) AS time_range,
			ts.room,
			u.username AS instructor
		FROM teacher_subjects ts
		JOIN subjects sub ON ts.subject_id = sub.id
		JOIN users u ON ts.teacher_id = u.id
		WHERE FIND_IN_SET(ts.subject_id, ?)
		  AND ts.course_id = ?
		  AND u.role = 'teacher'
		ORDER BY ts.day, ts.time_start
	`, subjects, courseID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var schedule []gin.H

	for rows.Next() {
		var subject, day, timeRange, room, instructor string

		err := rows.Scan(&subject, &day, &timeRange, &room, &instructor)
		if err != nil {
			continue
		}

		schedule = append(schedule, gin.H{
			"subject":    subject,
			"day":        day,
			"time":       timeRange,
			"room":       room,
			"instructor": instructor,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"schedule": schedule,
	})
}

func StudentRequestDocument(c *gin.Context) {
	// Log the incoming request
	fmt.Println("📥 Document Request received")

	var req struct {
		DocumentType string `json:"document_type"`
		Purpose      string `json:"purpose"`
		Copies       int    `json:"copies"`
	}

	// Parse JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("❌ JSON binding error:", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format. Please check your input.",
		})
		return
	}

	// Validate required fields
	if req.DocumentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "document_type is required",
		})
		return
	}

	if req.Purpose == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "purpose is required",
		})
		return
	}

	// Default copies to 1
	if req.Copies <= 0 {
		req.Copies = 1
	}

	// Validate copies range
	if req.Copies > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 10 copies allowed per request",
		})
		return
	}

	// Get authenticated student
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		fmt.Println("❌ No student_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "not authenticated",
		})
		return
	}

	fmt.Printf("✅ Student ID: %s\n", studentStrID)

	// Get student DB ID
	var studentDBID int
	err := config.DB.QueryRow(`
        SELECT id FROM students WHERE student_id = ?
    `, studentStrID).Scan(&studentDBID)

	if err != nil {
		fmt.Println("❌ Student not found in database:", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "student not found",
		})
		return
	}

	fmt.Printf("✅ Student DB ID: %d\n", studentDBID)
	fmt.Printf("📋 Request: Type=%s, Purpose=%s, Copies=%d\n",
		req.DocumentType, req.Purpose, req.Copies)

	// Insert document request
	result, err := config.DB.Exec(`
        INSERT INTO document_requests (student_id, document_type, purpose, copies, status, requested_at)
        VALUES (?, ?, ?, ?, 'pending', NOW())
    `, studentDBID, req.DocumentType, req.Purpose, req.Copies)

	if err != nil {
		fmt.Println("❌ Database insert error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to submit request. Please try again later.",
		})
		return
	}

	requestID, _ := result.LastInsertId()

	fmt.Printf("✅ Request created with ID: %d\n", requestID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "document request submitted successfully",
		"request_id": requestID,
		"status":     "pending",
	})
}

// ===================== STUDENT VIEW DOCUMENT REQUESTS =====================

func StudentGetDocumentRequests(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Get student DB ID
	var studentDBID int
	err := config.DB.QueryRow(`
        SELECT id FROM students WHERE student_id = ?
    `, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Get all document requests (UPDATED: Added document_file field)
	rows, err := config.DB.Query(`
        SELECT 
            id,
            document_type,
            purpose,
            copies,
            status,
            DATE_FORMAT(requested_at, '%Y-%m-%d %H:%i:%s') as requested_at,
            DATE_FORMAT(processed_at, '%Y-%m-%d %H:%i:%s') as processed_at,
            IFNULL(notes, ''),
            IFNULL(document_file, '')
        FROM document_requests
        WHERE student_id = ?
        ORDER BY requested_at DESC
    `, studentDBID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch requests"})
		return
	}
	defer rows.Close()

	var requests []gin.H

	for rows.Next() {
		var (
			id, copies                               int
			docType, purpose, status, notes, docPath string
			requestedAt                              string
			processedAt                              *string
		)

		// Scan with document_file included
		rows.Scan(&id, &docType, &purpose, &copies, &status, &requestedAt, &processedAt, &notes, &docPath)

		// Normalize path for URL usage
		cleanPath := strings.ReplaceAll(docPath, "\\", "/")

		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}

		requests = append(requests, gin.H{
			"request_id":    id,
			"document_type": docType,
			"purpose":       purpose,
			"copies":        copies,
			"status":        status,
			"requested_at":  requestedAt,
			"processed_at":  processedAt,
			"notes":         notes,
			"document_path": cleanPath, // ⬅️ FIXED: Clean path without "./"
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
	})
}

func StudentGetGrades(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Get student DB ID
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Get student info
	var firstName, lastName, email string
	var course, courseCode string
	var yearLevel int

	err = config.DB.QueryRow(`
		SELECT 
			IFNULL(st.first_name, 'Unknown'),
			IFNULL(st.last_name, 'Student'),
			IFNULL(st.email, 'N/A'),
			IFNULL(c.course_name, 'N/A'),
			IFNULL(c.code, 'N/A'),
			IFNULL(sa.year_level, 0)
		FROM students st
		LEFT JOIN student_academic sa ON st.id = sa.student_id
		LEFT JOIN courses c ON sa.course = c.id
		WHERE st.id = ?
	`, studentDBID).Scan(&firstName, &lastName, &email, &course, &courseCode, &yearLevel)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch student info"})
		return
	}

	// Get ONLY RELEASED grades
	rows, err := config.DB.Query(`
		SELECT 
			g.id as grade_id,
			s.subject_name,
			s.code as subject_code,
			u.username as teacher_name,  
			g.prelim,
			g.midterm,
			g.finals,
			CASE 
				WHEN g.prelim IS NOT NULL AND g.midterm IS NOT NULL AND g.finals IS NOT NULL 
				THEN ROUND((g.prelim + g.midterm + g.finals) / 3, 2)
				ELSE NULL
			END as average,
			IFNULL(g.remarks, '') as remarks,
			DATE_FORMAT(g.released_at, '%Y-%m-%d %H:%i:%s') as released_at
		FROM grades g
		INNER JOIN subjects s ON g.subject_id = s.id
		INNER JOIN users u ON g.teacher_id = u.id 
		WHERE g.student_id = ? 
		AND g.is_released = TRUE
		ORDER BY s.subject_name
	`, studentDBID)

	if err != nil {
		fmt.Println("❌ Database query error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch grades"})
		return
	}
	defer rows.Close()

	var grades []gin.H

	for rows.Next() {
		var (
			gradeID                               int
			subjectName, subjectCode, teacherName string
			prelim, midterm, finals, average      *float64
			remarks                               string
			releasedAt                            *string
		)

		err := rows.Scan(
			&gradeID, &subjectName, &subjectCode, &teacherName,
			&prelim, &midterm, &finals, &average,
			&remarks, &releasedAt,
		)

		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		grades = append(grades, gin.H{
			"grade_id":     gradeID,
			"subject":      subjectName,
			"subject_code": subjectCode,
			"teacher_name": teacherName,
			"prelim":       prelim,
			"midterm":      midterm,
			"finals":       finals,
			"average":      average,
			"remarks":      remarks,
			"released_at":  releasedAt,
		})
	}

	if grades == nil {
		grades = []gin.H{}
	}

	// Calculate GPA
	var totalGradePoints float64
	var gradeCount int
	for _, grade := range grades {
		if avg, ok := grade["average"].(*float64); ok && avg != nil {
			totalGradePoints += *avg
			gradeCount++
		}
	}

	var gpa float64
	if gradeCount > 0 {
		gpa = totalGradePoints / float64(gradeCount)
	}

	fmt.Printf("✅ Found %d released grades for student %s\n", len(grades), studentStrID)

	c.JSON(http.StatusOK, gin.H{
		"student": gin.H{
			"student_id":  studentStrID,
			"name":        firstName + " " + lastName,
			"email":       email,
			"course":      course,
			"course_code": courseCode,
			"year_level":  yearLevel,
		},
		"grades":       grades,
		"total_grades": len(grades),
		"gpa":          gpa,
	})
}

func StudentGetProfile(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Get student DB ID and basic info (UPDATED: Added profile_picture)
	var studentDBID int
	var firstName, lastName, email, contactNumber, address, profilePicture string

	err := config.DB.QueryRow(`
		SELECT 
			id,
			IFNULL(first_name, ''),
			IFNULL(last_name, ''),
			IFNULL(email, ''),
			IFNULL(contact_number, ''),
			IFNULL(address, ''),
			IFNULL(profile_picture, '')
		FROM students
		WHERE student_id = ?
	`, studentStrID).Scan(
		&studentDBID,
		&firstName,
		&lastName,
		&email,
		&contactNumber,
		&address,
		&profilePicture,
	)

	if err != nil {
		fmt.Println("❌ Error fetching profile:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Get academic info
	var course, courseCode string
	var yearLevel int

	err = config.DB.QueryRow(`
		SELECT 
			IFNULL(c.course_name, 'N/A'),
			IFNULL(c.code, 'N/A'),
			IFNULL(sa.year_level, 0)
		FROM student_academic sa
		LEFT JOIN courses c ON sa.course = c.id
		WHERE sa.student_id = ?
		ORDER BY sa.id DESC
		LIMIT 1
	`, studentDBID).Scan(&course, &courseCode, &yearLevel)

	if err != nil {
		course = "N/A"
		courseCode = "N/A"
		yearLevel = 0
	}

	// Normalize profile picture path
	cleanProfilePic := strings.ReplaceAll(profilePicture, "\\", "/")
	if strings.HasPrefix(cleanProfilePic, "./") {
		cleanProfilePic = cleanProfilePic[2:]
	}

	c.JSON(http.StatusOK, gin.H{
		"student_id":      studentStrID,
		"student_name":    firstName + " " + lastName,
		"first_name":      firstName,
		"last_name":       lastName,
		"email":           email,
		"contact_number":  contactNumber,
		"address":         address,
		"profile_picture": cleanProfilePic, // ⬅️ NEW
		"course":          course,
		"course_code":     courseCode,
		"year_level":      yearLevel,
	})
}

func StudentUploadProfilePicture(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Get student DB ID
	var studentDBID int
	var oldProfilePic string
	err := config.DB.QueryRow(`
		SELECT id, IFNULL(profile_picture, '') FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID, &oldProfilePic)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("profile_picture")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 5MB)"})
		return
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
	}

	// Open file to check content type
	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read file"})
		return
	}
	defer fileContent.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = fileContent.Read(buffer)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read file content"})
		return
	}

	contentType := http.DetectContentType(buffer)
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type (only jpg, png, gif allowed)"})
		return
	}

	// Create uploads directory if it doesn't exist
	uploadsDir := "./uploads/profile_pictures"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		fmt.Println("❌ Error creating directory:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create upload directory"})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf("student_%s_%d%s", studentStrID, time.Now().Unix(), ext)
	filePath := filepath.Join(uploadsDir, newFilename)

	// Save the file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		fmt.Println("❌ Error saving file:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save file"})
		return
	}

	// Delete old profile picture if exists
	if oldProfilePic != "" {
		oldPath := "./" + oldProfilePic
		if _, err := os.Stat(oldPath); err == nil {
			os.Remove(oldPath)
			fmt.Printf("🗑️ Deleted old profile picture: %s\n", oldPath)
		}
	}

	// Update database
	_, err = config.DB.Exec(`
		UPDATE students SET profile_picture = ? WHERE id = ?
	`, filePath, studentDBID)

	if err != nil {
		fmt.Println("❌ Database update error:", err)
		// Clean up uploaded file if DB update fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update profile"})
		return
	}

	// Normalize path for response
	cleanPath := strings.ReplaceAll(filePath, "\\", "/")
	if strings.HasPrefix(cleanPath, "./") {
		cleanPath = cleanPath[2:]
	}

	fmt.Printf("✅ Profile picture uploaded for student %s: %s\n", studentStrID, cleanPath)

	c.JSON(http.StatusOK, gin.H{
		"message":         "profile picture uploaded successfully",
		"profile_picture": cleanPath,
	})
}

func StudentUpdateProfile(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req struct {
		FirstName     string `json:"first_name"`
		LastName      string `json:"last_name"`
		Email         string `json:"email"`
		ContactNumber string `json:"contact_number"`
		Address       string `json:"address"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	// Validate required fields
	if req.FirstName == "" || req.LastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first name and last name are required"})
		return
	}

	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	// Validate email format (basic)
	if !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		return
	}

	// Get student DB ID
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Update profile
	_, err = config.DB.Exec(`
		UPDATE students
		SET 
			first_name = ?,
			last_name = ?,
			email = ?,
			contact_number = ?,
			address = ?
		WHERE id = ?
	`, req.FirstName, req.LastName, req.Email, req.ContactNumber,
		req.Address, studentDBID)

	if err != nil {
		fmt.Println("❌ Update error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	fmt.Printf("✅ Profile updated for student %s\n", studentStrID)

	c.JSON(http.StatusOK, gin.H{
		"message":      "profile updated successfully",
		"student_name": req.FirstName + " " + req.LastName,
		"email":        req.Email,
	})
}

// ===================== STUDENT CHANGE PASSWORD =====================

func StudentChangePassword(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Validate password length
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "new password must be at least 6 characters",
		})
		return
	}

	// Get student DB ID and current password
	var studentDBID int
	var currentPassword string

	err := config.DB.QueryRow(`
		SELECT id, password FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID, &currentPassword)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Verify current password
	// Note: If you're using hashed passwords, use bcrypt.CompareHashAndPassword
	// For now, assuming plain text comparison (should be hashed in production!)
	if currentPassword != req.CurrentPassword {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "current password is incorrect",
		})
		return
	}

	// Update password
	// TODO: In production, hash the password using bcrypt before storing
	_, err = config.DB.Exec(`
		UPDATE students SET password = ? WHERE id = ?
	`, req.NewPassword, studentDBID)

	if err != nil {
		fmt.Println("❌ Password update error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to change password",
		})
		return
	}

	fmt.Printf("✅ Password changed for student %s\n", studentStrID)

	c.JSON(http.StatusOK, gin.H{
		"message": "password changed successfully",
	})
}

// ===================== STUDENT VIEW LESSONS =====================

func StudentGetLessons(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Get student DB ID
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Get student's enrolled subjects
	var subjectIDs string
	err = config.DB.QueryRow(`
		SELECT IFNULL(subjects, '')
		FROM student_academic
		WHERE student_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, studentDBID).Scan(&subjectIDs)

	if err != nil || subjectIDs == "" {
		c.JSON(http.StatusOK, gin.H{
			"lessons": []gin.H{},
			"message": "no subjects enrolled",
		})
		return
	}

	// Get lessons for enrolled subjects - CORRECTED TO USE lesson_materials
	rows, err := config.DB.Query(`
		SELECT 
			lm.id,
			s.subject_name,
			s.code as subject_code,
			lm.title,
			IFNULL(lm.description, ''),
			IFNULL(lm.file_path, ''),
			lm.type as file_type,
			DATE_FORMAT(lm.created_at, '%Y-%m-%d %H:%i:%s') as uploaded_at,
			u.username as teacher_name,
			lm.due_date
		FROM lesson_materials lm
		INNER JOIN subjects s ON lm.subject_id = s.id
		INNER JOIN users u ON lm.teacher_id = u.id
		WHERE FIND_IN_SET(lm.subject_id, ?)
		ORDER BY lm.created_at DESC
	`, subjectIDs)

	if err != nil {
		fmt.Println("❌ Error fetching lessons:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch lessons"})
		return
	}
	defer rows.Close()

	var lessons []gin.H

	for rows.Next() {
		var (
			id                                           int
			subjectName, subjectCode, title, description string
			filePath, fileType, uploadedAt, teacherName  string
			dueDate                                      *string
		)

		err := rows.Scan(
			&id, &subjectName, &subjectCode, &title,
			&description, &filePath, &fileType,
			&uploadedAt, &teacherName, &dueDate,
		)

		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		// Normalize file path
		cleanPath := strings.ReplaceAll(filePath, "\\", "/")
		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}

		lesson := gin.H{
			"lesson_id":    id,
			"subject":      subjectName,
			"subject_code": subjectCode,
			"title":        title,
			"description":  description,
			"file_path":    cleanPath,
			"file_type":    fileType,
			"uploaded_at":  uploadedAt,
			"teacher_name": teacherName,
		}

		if dueDate != nil {
			lesson["due_date"] = *dueDate
		}

		lessons = append(lessons, lesson)
	}

	if lessons == nil {
		lessons = []gin.H{}
	}

	fmt.Printf("✅ Found %d lessons for student %s\n", len(lessons), studentStrID)

	c.JSON(http.StatusOK, gin.H{
		"lessons":       lessons,
		"total_lessons": len(lessons),
	})
}

// ===================== STUDENT UPLOAD SUBMISSION (CORRECTED) =====================

func StudentUploadSubmission(c *gin.Context) {

	// ================= AUTH =================
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// ================= GET STUDENT DB ID =================
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// ================= VALIDATE LESSON ID =================
	materialStr := strings.TrimSpace(c.PostForm("lesson_id"))
	fmt.Println("📥 Received lesson_id:", materialStr)

	if materialStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lesson_id is required"})
		return
	}

	materialID, err := strconv.Atoi(materialStr)
	if err != nil || materialID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	// ================= OTHER FORM DATA =================
	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	// ================= VERIFY LESSON EXISTS =================
	// ================= VERIFY LESSON EXISTS =================
	var subjectID int
	var dueDateStr *string

	err = config.DB.QueryRow(`
    SELECT subject_id, due_date
    FROM lesson_materials
    WHERE id = ?
`, materialID).Scan(&subjectID, &dueDateStr)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "lesson material not found"})
		return
	}

	if err != nil {
		fmt.Println("DB lesson lookup error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Parse due_date string → *time.Time
	var dueDate *time.Time
	if dueDateStr != nil && *dueDateStr != "" {
		for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
			if t, parseErr := time.Parse(layout, *dueDateStr); parseErr == nil {
				dueDate = &t
				break
			}
		}
	}

	// ================= CHECK ENROLLMENT =================
	var enrolled bool

	err = config.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM student_academic
			WHERE student_id = ?
			AND FIND_IN_SET(?, subjects) > 0
		)
	`, studentDBID, subjectID).Scan(&enrolled)

	if err != nil || !enrolled {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not enrolled in this subject",
		})
		return
	}

	// ================= FILE VALIDATION =================
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 10MB)"})
		return
	}

	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
		"text/plain":      true,
	}

	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read file"})
		return
	}
	defer fileContent.Close()

	buffer := make([]byte, 512)
	fileContent.Read(buffer)

	contentType := http.DetectContentType(buffer)

	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid file type",
		})
		return
	}

	// ================= SAVE FILE =================
	uploadDir := "./uploads/student_submissions"
	os.MkdirAll(uploadDir, os.ModePerm)

	ext := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf(
		"submission_%s_%d_%d%s",
		studentStrID,
		materialID,
		time.Now().Unix(),
		ext,
	)

	filePath := filepath.Join(uploadDir, newFilename)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file save failed"})
		return
	}

	// ================= LATE CHECK =================
	status := "on-time"
	if dueDate != nil && time.Now().After(*dueDate) {
		status = "late"
	}

	// ================= CLEAN REMARKS =================
	remarks := title
	if description != "" {
		remarks += " - " + description
	}

	// ================= INSERT DB =================
	result, err := config.DB.Exec(`
		INSERT INTO student_submissions
		(student_id, material_id, file_name, file_path, file_size, submitted_at, status, remarks)
		VALUES (?, ?, ?, ?, ?, NOW(), ?, ?)
	`, studentDBID, materialID, file.Filename, filePath, file.Size, status, remarks)

	if err != nil {
		os.Remove(filePath)
		fmt.Println("DB insert error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "submission save failed"})
		return
	}

	submissionID, _ := result.LastInsertId()

	// ================= RESPONSE =================
	cleanPath := strings.ReplaceAll(filePath, "\\", "/")
	if strings.HasPrefix(cleanPath, "./") {
		cleanPath = cleanPath[2:]
	}

	fmt.Printf("✅ Submission saved: student=%s lesson=%d\n", studentStrID, materialID)

	c.JSON(http.StatusOK, gin.H{
		"message":       "submission uploaded successfully",
		"submission_id": submissionID,
		"title":         title,
		"file_path":     cleanPath,
		"status":        status,
	})
}

// ===================== STUDENT VIEW OWN SUBMISSIONS (FIXED) =====================

func StudentGetSubmissions(c *gin.Context) {
	studentStrID := c.GetString("student_id")
	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Get student DB ID
	var studentDBID int
	err := config.DB.QueryRow(`
		SELECT id FROM students WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Optional: filter by material_id (lesson_id in frontend)
	materialID := c.Query("lesson_id")

	var query string
	var args []interface{}

	if materialID != "" {
		query = `
			SELECT 
				ss.id,
				lm.title as lesson_title,
				s.subject_name,
				ss.file_name,
				IFNULL(ss.remarks, ''),
				ss.file_path,
				ss.status,
				DATE_FORMAT(ss.submitted_at, '%Y-%m-%d %H:%i:%s') as submitted_at
			FROM student_submissions ss
			INNER JOIN lesson_materials lm ON ss.material_id = lm.id
			INNER JOIN subjects s ON lm.subject_id = s.id
			WHERE ss.student_id = ? AND ss.material_id = ?
			ORDER BY ss.submitted_at DESC
		`
		args = []interface{}{studentDBID, materialID}
	} else {
		query = `
			SELECT 
				ss.id,
				lm.title as lesson_title,
				s.subject_name,
				ss.file_name,
				IFNULL(ss.remarks, ''),
				ss.file_path,
				ss.status,
				DATE_FORMAT(ss.submitted_at, '%Y-%m-%d %H:%i:%s') as submitted_at
			FROM student_submissions ss
			INNER JOIN lesson_materials lm ON ss.material_id = lm.id
			INNER JOIN subjects s ON lm.subject_id = s.id
			WHERE ss.student_id = ?
			ORDER BY ss.submitted_at DESC
		`
		args = []interface{}{studentDBID}
	}

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		fmt.Println("❌ Error fetching submissions:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch submissions"})
		return
	}
	defer rows.Close()

	var submissions []gin.H

	for rows.Next() {
		var (
			id                                           int
			lessonTitle, subjectName, title, description string
			filePath, status, submittedAt                string
		)

		err := rows.Scan(
			&id, &lessonTitle, &subjectName, &title,
			&description, &filePath, &status,
			&submittedAt,
		)

		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		// Normalize file path
		cleanPath := strings.ReplaceAll(filePath, "\\", "/")
		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}

		submissions = append(submissions, gin.H{
			"submission_id": id,
			"lesson_title":  lessonTitle,
			"subject":       subjectName,
			"title":         title,
			"description":   description,
			"file_path":     cleanPath,
			"status":        status, // ✅ FIXED: Now includes status
			"submitted_at":  submittedAt,
		})
	}

	if submissions == nil {
		submissions = []gin.H{}
	}

	fmt.Printf("✅ Found %d submissions for student %s\n", len(submissions), studentStrID)

	c.JSON(http.StatusOK, gin.H{
		"submissions":       submissions,
		"total_submissions": len(submissions),
	})
}

// ===================== STUDENT GET ANNOUNCEMENTS =====================

// ===================== STUDENT GET ANNOUNCEMENTS (UPDATED: Added Records Announcements) =====================

func StudentGetAnnouncements(c *gin.Context) {
	role := c.GetString("role")
	studentStrID := c.GetString("student_id")

	// Validate student authentication
	if role != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Student role required"})
		return
	}

	if studentStrID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Get student DB ID
	var studentDBID int
	var studentStatus string
	err := config.DB.QueryRow(`
		SELECT id, IFNULL(status, 'pending') 
		FROM students 
		WHERE student_id = ?
	`, studentStrID).Scan(&studentDBID, &studentStatus)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	// Get student's enrolled subjects for teacher announcements
	var subjectIDs string
	err = config.DB.QueryRow(`
		SELECT IFNULL(subjects, '')
		FROM student_academic
		WHERE student_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, studentDBID).Scan(&subjectIDs)

	if err != nil {
		subjectIDs = "" // No subjects enrolled yet
	}

	var allAnnouncements []gin.H

	// ===== 1. GET TEACHER ANNOUNCEMENTS =====
	if subjectIDs != "" {
		teacherQuery := `
			SELECT 
				a.id,
				a.title,
				a.content,
				s.subject_name,
				s.code as subject_code,
				u.username as teacher_name,
				a.image_name,
				a.image_path,
				a.image_size,
				DATE_FORMAT(a.created_at, '%Y-%m-%d %H:%i:%s') as created_at,
				'teacher' as source
			FROM announcements a
			INNER JOIN subjects s ON a.subject_id = s.id
			INNER JOIN users u ON a.teacher_id = u.id
			WHERE FIND_IN_SET(a.subject_id, ?)
			ORDER BY a.created_at DESC
			LIMIT 25
		`

		rows, err := config.DB.Query(teacherQuery, subjectIDs)
		if err == nil {
			defer rows.Close()

			for rows.Next() {
				var (
					id                                   int
					title, content, subject, subjectCode string
					teacherName                          string
					imageName, imagePath                 sql.NullString
					imageSize                            sql.NullInt64
					createdAt                            string
					source                               string
				)

				err := rows.Scan(
					&id, &title, &content, &subject, &subjectCode,
					&teacherName, &imageName, &imagePath, &imageSize, &createdAt, &source,
				)

				if err != nil {
					fmt.Println("❌ Row scan error:", err)
					continue
				}

				announcement := gin.H{
					"id":           id,
					"title":        title,
					"content":      content,
					"subject":      subject,
					"subject_code": subjectCode,
					"author":       teacherName,
					"author_type":  "Teacher",
					"created_at":   createdAt,
					"source":       source,
					"image_url":    nil,
					"image_size":   nil,
				}

				// Normalize image path
				if imagePath.Valid {
					cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
					if strings.HasPrefix(cleanPath, "./") {
						cleanPath = cleanPath[2:]
					}
					announcement["image_url"] = "/" + cleanPath
					announcement["image_size"] = imageSize.Int64
				}

				allAnnouncements = append(allAnnouncements, announcement)
			}
		}
	}

	// ===== 2. GET REGISTRAR ANNOUNCEMENTS =====
	registrarQuery := `
		SELECT 
			id,
			title,
			content,
			target_audience,
			image_name,
			image_path,
			image_size,
			DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') as created_at
		FROM registrar_announcements
		WHERE target_audience = 'all' OR target_audience = ?
		ORDER BY created_at DESC
		LIMIT 25
	`

	// Determine student's target audience category
	targetAudience := studentStatus // 'pending', 'approved', or 'enrolled'
	if targetAudience == "" {
		targetAudience = "pending"
	}

	rows, err := config.DB.Query(registrarQuery, targetAudience)
	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var (
				id                        int
				title, content, targetAud string
				imageName, imagePath      sql.NullString
				imageSize                 sql.NullInt64
				createdAt                 string
			)

			err := rows.Scan(
				&id, &title, &content, &targetAud,
				&imageName, &imagePath, &imageSize, &createdAt,
			)

			if err != nil {
				fmt.Println("❌ Row scan error:", err)
				continue
			}

			announcement := gin.H{
				"id":              id,
				"title":           title,
				"content":         content,
				"subject":         "University Announcement",
				"subject_code":    "ADMIN",
				"author":          "Registrar's Office",
				"author_type":     "Registrar",
				"target_audience": targetAud,
				"created_at":      createdAt,
				"source":          "registrar",
				"image_url":       nil,
				"image_size":      nil,
			}

			// Normalize image path
			if imagePath.Valid {
				cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
				if strings.HasPrefix(cleanPath, "./") {
					cleanPath = cleanPath[2:]
				}
				announcement["image_url"] = "/" + cleanPath
				announcement["image_size"] = imageSize.Int64
			}

			allAnnouncements = append(allAnnouncements, announcement)
		}
	}

	// ===== 3. GET RECORDS OFFICER ANNOUNCEMENTS (NEW) =====
	recordsQuery := `
		SELECT 
			id,
			title,
			content,
			priority,
			target_audience,
			image_name,
			image_path,
			image_size,
			DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') as created_at
		FROM records_announcements
		WHERE is_active = TRUE 
		AND (target_audience = 'all' OR target_audience = 'students')
		ORDER BY created_at DESC
		LIMIT 25
	`

	recordsRows, err := config.DB.Query(recordsQuery)
	if err == nil {
		defer recordsRows.Close()

		for recordsRows.Next() {
			var (
				id                                  int
				title, content, priority, targetAud string
				imageName, imagePath                sql.NullString
				imageSize                           sql.NullInt64
				createdAt                           string
			)

			err := recordsRows.Scan(
				&id, &title, &content, &priority, &targetAud,
				&imageName, &imagePath, &imageSize, &createdAt,
			)

			if err != nil {
				fmt.Println("❌ Row scan error:", err)
				continue
			}

			announcement := gin.H{
				"id":              id,
				"title":           title,
				"content":         content,
				"subject":         "Records Office",
				"subject_code":    "RECORDS",
				"author":          "Records Office",
				"author_type":     "Records",
				"priority":        priority,
				"target_audience": targetAud,
				"created_at":      createdAt,
				"source":          "records",
				"image_url":       nil,
				"image_size":      nil,
			}

			// Normalize image path
			if imagePath.Valid {
				cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
				if strings.HasPrefix(cleanPath, "./") {
					cleanPath = cleanPath[2:]
				}
				announcement["image_url"] = "/" + cleanPath
				announcement["image_size"] = imageSize.Int64
			}

			allAnnouncements = append(allAnnouncements, announcement)
		}
	}

	fmt.Printf("✅ Found %d total announcements for student %s\n", len(allAnnouncements), studentStrID)

	if allAnnouncements == nil {
		allAnnouncements = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"announcements": allAnnouncements,
		"total":         len(allAnnouncements),
	})
}

// Logout endpoint
func Logout(c *gin.Context) {
	// Clear the session cookie
	c.SetCookie(
		"session_token",
		"",
		-1, // MaxAge -1 deletes the cookie
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}
