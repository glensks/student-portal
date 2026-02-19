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
	"github.com/jung-kurt/gofpdf"
)

func RecordsDashboard(c *gin.Context) {
	userID := c.GetInt("user_id")
	role := c.GetString("role")

	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "access denied",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome Records Officer",
		"user_id": userID,
	})
}

func RecordsMe(c *gin.Context) {
	userID := c.GetInt("user_id")

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome Records Officer",
		"user_id": userID,
	})
}

// ===================== GET ALL DOCUMENT REQUESTS =====================

func RecordsGetDocumentRequests(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Optional filter by status
	statusFilter := c.Query("status")

	query := `
		SELECT 
			dr.id,
			dr.student_id,
			IFNULL(s.student_id, 'N/A') as student_number,
			IFNULL(s.first_name, 'Unknown') as first_name,
			IFNULL(s.last_name, 'Student') as last_name,
			IFNULL(c.course_name, 'N/A') as course,
			IFNULL(sa.year_level, 'N/A') as year,
			dr.document_type,
			dr.purpose,
			dr.copies,
			dr.status,
			DATE_FORMAT(dr.requested_at, '%Y-%m-%d %H:%i:%s') as requested_at,
			DATE_FORMAT(dr.processed_at, '%Y-%m-%d %H:%i:%s') as processed_at,
			IFNULL(dr.notes, ''),
			IFNULL(dr.document_file, '')
		FROM document_requests dr
		LEFT JOIN students s ON dr.student_id = s.id
		LEFT JOIN student_academic sa ON s.id = sa.student_id
		LEFT JOIN courses c ON sa.course = c.id
	`

	args := []interface{}{}

	if statusFilter != "" {
		query += " WHERE dr.status = ?"
		args = append(args, statusFilter)
	}

	query += " ORDER BY dr.requested_at DESC"

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		fmt.Println("❌ Database query error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch requests"})
		return
	}
	defer rows.Close()

	var requests []gin.H

	for rows.Next() {
		var (
			id, studentID, copies                            int
			studentNumber, firstName, lastName, course, year string
			docType, purpose                                 string
			status, notes, docPath                           string
			requestedAt                                      string
			processedAt                                      *string
		)

		err := rows.Scan(
			&id, &studentID, &studentNumber, &firstName, &lastName,
			&course, &year, &docType, &purpose, &copies, &status, &requestedAt,
			&processedAt, &notes, &docPath,
		)

		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		studentName := firstName + " " + lastName

		requests = append(requests, gin.H{
			"request_id":     id,
			"student_id":     studentID,
			"student_number": studentNumber,
			"student_name":   studentName,
			"course":         course,
			"year":           year,
			"document_type":  docType,
			"purpose":        purpose,
			"copies":         copies,
			"status":         status,
			"requested_at":   requestedAt,
			"processed_at":   processedAt,
			"notes":          notes,
			"document_path":  docPath,
		})
	}

	fmt.Printf("✅ Found %d requests\n", len(requests))

	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
	})
}

// ===================== AUTO-GENERATE DOCUMENT FUNCTIONS =====================

type StudentDocumentData struct {
	RequestID         int
	StudentNumber     string
	FirstName         string
	MiddleName        string
	LastName          string
	Course            string
	YearLevel         string
	Email             string
	Address           string
	Semester          string
	ScholarshipStatus string
	Purpose           string
	DateRequested     string
}

func generateTranscriptOfRecords(data StudentDocumentData, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Official Header with better formatting
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 8, "THE UNIVERSITY OF MANILA")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "Sampaloc District in Manila, Philippines.")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 4, "Tel: 287355085 | https://www.facebook.com/UMOFFICIAL1913/")
	pdf.Ln(2)
	pdf.SetDrawColor(40, 145, 108)
	pdf.SetLineWidth(0.5)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(8)

	// Office designation
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, "OFFICE OF THE UNIVERSITY REGISTRAR")
	pdf.Ln(12)

	// Document Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "TRANSCRIPT OF RECORDS")
	pdf.Ln(15)

	// Student Information Section
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, "STUDENT INFORMATION")
	pdf.Ln(1)
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(45, 6, "Student Number:")
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, data.StudentNumber)
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(45, 6, "Name:")
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, fmt.Sprintf("%s %s %s", data.FirstName, data.MiddleName, data.LastName))
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(45, 6, "Course/Program:")
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, data.Course)
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(45, 6, "Year Level:")
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, data.YearLevel)
	pdf.Ln(10)

	// Academic Record Section
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, "ACADEMIC RECORD")
	pdf.Ln(1)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(4)

	// Grades table header
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(40, 145, 108)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(85, 8, "SUBJECT/COURSE", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "UNITS", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "GRADE", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "REMARKS", "1", 1, "C", true, 0, "")

	// Reset text color
	pdf.SetTextColor(0, 0, 0)

	// Fetch grades from database
	gradesQuery := `
		SELECT 
			s.subject_name,
			s.code as subject_code,
			CASE 
				WHEN g.prelim IS NOT NULL AND g.midterm IS NOT NULL AND g.finals IS NOT NULL 
				THEN ROUND((g.prelim + g.midterm + g.finals) / 3, 2)
				ELSE NULL
			END as average,
			IFNULL(g.remarks, 'INC') as remarks
		FROM grades g
		INNER JOIN subjects s ON g.subject_id = s.id
		INNER JOIN students st ON g.student_id = st.id
		WHERE st.student_id = ? AND g.is_released = TRUE
		ORDER BY s.subject_name
	`

	rows, err := config.DB.Query(gradesQuery, data.StudentNumber)
	if err != nil {
		return err
	}
	defer rows.Close()

	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(245, 245, 245)
	hasGrades := false
	rowCount := 0

	for rows.Next() {
		var subjectName, subjectCode, remarks string
		var average *float64

		err := rows.Scan(&subjectName, &subjectCode, &average, &remarks)
		if err != nil {
			continue
		}

		hasGrades = true
		gradeStr := "INC"
		if average != nil {
			gradeStr = fmt.Sprintf("%.2f", *average)
		}

		defaultUnits := "3.0"

		// Alternate row colors
		fill := rowCount%2 == 0
		pdf.CellFormat(85, 7, subjectName, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(25, 7, defaultUnits, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(30, 7, gradeStr, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(40, 7, remarks, "1", 1, "C", fill, 0, "")
		rowCount++
	}

	if !hasGrades {
		pdf.SetFont("Arial", "I", 10)
		pdf.Cell(0, 10, "No grades available or released at this time.")
		pdf.Ln(10)
	}

	// Footer section
	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 5, fmt.Sprintf("Document generated on: %s", time.Now().Format("January 02, 2006 at 3:04 PM")))
	pdf.Ln(4)
	pdf.Cell(0, 5, "*** This is an official computer-generated document. No signature required. ***")
	pdf.Ln(12)

	// Registrar signature line
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 5, "_________________________________________")
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(0, 5, "University Registrar")
	pdf.Ln(3)
	pdf.SetFont("Arial", "", 8)
	pdf.Cell(0, 4, "Office of the University Registrar")

	return pdf.OutputFileAndClose(outputPath)
}

func generateCertificateOfEnrollment(data StudentDocumentData, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 8, "THE UNIVERSITY OF MANILA")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "Sampaloc District in Manila, Philippines.")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 4, "Tel: 287355085 | https://www.facebook.com/UMOFFICIAL1913/")
	pdf.Ln(2)
	pdf.SetDrawColor(40, 145, 108)
	pdf.SetLineWidth(0.5)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(8)

	// Footer
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, "*** This is an official computer-generated document. No signature required. ***")

	return pdf.OutputFileAndClose(outputPath)
}

func generateGoodMoralCertificate(data StudentDocumentData, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 8, "THE UNIVERSITY OF MANILA")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "Sampaloc District in Manila, Philippines.")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 4, "Tel: 287355085 | https://www.facebook.com/UMOFFICIAL1913/")
	pdf.Ln(2)
	pdf.SetDrawColor(40, 145, 108)
	pdf.SetLineWidth(0.5)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(8)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, "OFFICE OF STUDENT AFFAIRS AND SERVICES")
	pdf.Ln(15)

	// Document Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "CERTIFICATE OF GOOD MORAL CHARACTER")
	pdf.Ln(18)

	// Body
	pdf.SetFont("Arial", "B", 11)
	pdf.MultiCell(0, 6, "TO WHOM IT MAY CONCERN:", "", "L", false)
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 11)
	certText := fmt.Sprintf("This is to certify that %s %s %s, Student Number %s, "+
		"a %s Year student of the %s program, has demonstrated GOOD MORAL CHARACTER "+
		"during their enrollment at The University of Manila.",
		data.FirstName, data.MiddleName, data.LastName, data.StudentNumber,
		data.YearLevel, data.Course)

	pdf.MultiCell(0, 7, certText, "", "J", false)
	pdf.Ln(8)

	pdf.MultiCell(0, 7, "Based on our records, the above-named student has no pending disciplinary cases "+
		"and has not violated any university rules, regulations, or policies.", "", "J", false)
	pdf.Ln(8)

	pdf.MultiCell(0, 7, fmt.Sprintf("This certification is being issued upon the request of the student for %s.",
		strings.ToLower(data.Purpose)), "", "J", false)
	pdf.Ln(15)

	// Date issued
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Issued this %s.", time.Now().Format("2nd day of January, 2006")))
	pdf.Ln(20)

	// Signature section
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "_________________________________________")
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 5, "Director, Office of Student Affairs and Services")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 4, "The University of Manila")
	pdf.Ln(15)

	// Footer
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, "*** This is an official computer-generated document. No signature required. ***")

	return pdf.OutputFileAndClose(outputPath)
}

func generateHonorableDismissal(data StudentDocumentData, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 8, "THE UNIVERSITY OF MANILA")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "Sampaloc District in Manila, Philippines.")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 4, "Tel: 287355085 | https://www.facebook.com/UMOFFICIAL1913/")
	pdf.Ln(2)
	pdf.SetDrawColor(40, 145, 108)
	pdf.SetLineWidth(0.5)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(8)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, "OFFICE OF THE UNIVERSITY REGISTRAR")
	pdf.Ln(15)

	// Document Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "CERTIFICATE OF HONORABLE DISMISSAL")
	pdf.Ln(18)

	// Body
	pdf.SetFont("Arial", "B", 11)
	pdf.MultiCell(0, 6, "TO WHOM IT MAY CONCERN:", "", "L", false)
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 11)
	certText := fmt.Sprintf("This is to certify that %s %s %s, bearing Student Number %s, "+
		"was a bona fide student of The University of Manila, enrolled in the %s program.",
		data.FirstName, data.MiddleName, data.LastName, data.StudentNumber, data.Course)

	pdf.MultiCell(0, 7, certText, "", "J", false)
	pdf.Ln(8)

	pdf.MultiCell(0, 7, "The above-named student is hereby granted HONORABLE DISMISSAL from The University of Manila. "+
		"The student has settled all financial obligations, returned all university property, and has no pending "+
		"accountabilities with the university. The student is eligible for transfer to another institution of higher learning.", "", "J", false)
	pdf.Ln(8)

	pdf.MultiCell(0, 7, fmt.Sprintf("This certificate is issued upon request for %s.",
		strings.ToLower(data.Purpose)), "", "J", false)
	pdf.Ln(15)

	// Date issued
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Issued this %s.", time.Now().Format("2nd day of January, 2006")))
	pdf.Ln(20)

	// Signature section
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "_________________________________________")
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 5, "University Registrar")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 4, "Office of the University Registrar")
	pdf.Ln(15)

	// Footer
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, "*** This is an official computer-generated document. No signature required. ***")

	return pdf.OutputFileAndClose(outputPath)
}

// ===================== PROCESS DOCUMENT REQUEST (AUTO-GENERATE) =====================

func RecordsProcessDocumentRequest(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"` // approved or rejected
		Notes  string `json:"notes"`                     // optional notes
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate status
	if req.Status != "approved" && req.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "status must be 'approved' or 'rejected'",
		})
		return
	}

	// Get request details
	var (
		currentStatus, documentType, purpose           string
		studentID                                      int
		studentNumber, firstName, middleName, lastName string
		course, yearLevel, email, address, semester    string
		scholarshipStatus                              string
	)

	err = config.DB.QueryRow(`
		SELECT 
			dr.status, dr.document_type, dr.purpose, dr.student_id,
			s.student_id, s.first_name, IFNULL(s.middle_name, ''), s.last_name,
			s.email, s.address,
			IFNULL(c.course_name, 'N/A'),
			IFNULL(sa.year_level, '1'),
			IFNULL(sa.semester, '1st'),
			IFNULL(sa.scholarship_status, 'non-scholar')
		FROM document_requests dr
		INNER JOIN students s ON dr.student_id = s.id
		LEFT JOIN student_academic sa ON s.id = sa.student_id
		LEFT JOIN courses c ON sa.course = c.id
		WHERE dr.id = ?
	`, requestID).Scan(
		&currentStatus, &documentType, &purpose, &studentID,
		&studentNumber, &firstName, &middleName, &lastName,
		&email, &address, &course, &yearLevel, &semester, &scholarshipStatus,
	)

	if err != nil {
		fmt.Println("❌ Query error:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	if currentStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request already processed",
		})
		return
	}

	var documentPath string

	// If approved, auto-generate document
	if req.Status == "approved" {
		// Create uploads directory if not exists
		uploadsDir := "./uploads/documents"
		if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create upload directory",
			})
			return
		}

		// Prepare student data
		studentData := StudentDocumentData{
			RequestID:         requestID,
			StudentNumber:     studentNumber,
			FirstName:         firstName,
			MiddleName:        middleName,
			LastName:          lastName,
			Course:            course,
			YearLevel:         yearLevel,
			Email:             email,
			Address:           address,
			Semester:          semester,
			ScholarshipStatus: scholarshipStatus,
			Purpose:           purpose,
			DateRequested:     time.Now().Format("2006-01-02"),
		}

		// Generate filename
		timestamp := time.Now().Unix()
		filename := fmt.Sprintf("%s_%s_%d.pdf",
			strings.ReplaceAll(strings.ToLower(documentType), " ", "_"),
			studentNumber,
			timestamp,
		)
		documentPath = filepath.Join(uploadsDir, filename)

		// Generate document based on type
		var genErr error
		switch strings.ToLower(documentType) {
		case "transcript of records", "tor":
			genErr = generateTranscriptOfRecords(studentData, documentPath)
		case "certificate of enrollment", "coe":
			genErr = generateCertificateOfEnrollment(studentData, documentPath)
		case "good moral certificate", "good moral":
			genErr = generateGoodMoralCertificate(studentData, documentPath)
		case "honorable dismissal":
			genErr = generateHonorableDismissal(studentData, documentPath)
		default:
			// For unknown document types, generate a generic certificate
			genErr = generateCertificateOfEnrollment(studentData, documentPath)
		}

		if genErr != nil {
			fmt.Println("❌ Document generation error:", genErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to generate document",
			})
			return
		}
	}

	// Update request
	_, err = config.DB.Exec(`
		UPDATE document_requests
		SET 
			status = ?,
			processed_at = NOW(),
			notes = ?,
			document_file = ?
		WHERE id = ?
	`, req.Status, req.Notes, documentPath, requestID)

	if err != nil {
		fmt.Println("❌ Update error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to process request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "request processed successfully",
		"request_id":    requestID,
		"status":        req.Status,
		"document_path": documentPath,
	})
}

// ===================== GET SINGLE DOCUMENT REQUEST DETAILS =====================
func RecordsGetDocumentRequestDetails(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_id"})
		return
	}

	fmt.Printf("🔍 Looking for request ID: %d\n", requestID)

	var (
		id, studentDBID, copies                          int
		studentNumber, firstName, lastName, course, year string
		docType, purpose                                 string
		status, notes, docPath                           string
		requestedAt                                      string
		processedAt                                      *string
	)

	err = config.DB.QueryRow(`
		SELECT 
			dr.id,
			dr.student_id,
			IFNULL(s.student_id, 'N/A') as student_number,
			IFNULL(s.first_name, 'Unknown'),
			IFNULL(s.last_name, 'Student'),
			IFNULL(c.course_name, 'N/A') as course,
			IFNULL(sa.year_level, 'N/A') as year,
			dr.document_type,
			dr.purpose,
			dr.copies,
			dr.status,
			DATE_FORMAT(dr.requested_at, '%Y-%m-%d %H:%i:%s') as requested_at,
			DATE_FORMAT(dr.processed_at, '%Y-%m-%d %H:%i:%s') as processed_at,
			IFNULL(dr.notes, ''),
			IFNULL(dr.document_file, '')
		FROM document_requests dr
		LEFT JOIN students s ON dr.student_id = s.id
		LEFT JOIN student_academic sa ON s.id = sa.student_id
		LEFT JOIN courses c ON sa.course = c.id
		WHERE dr.id = ?
	`, requestID).Scan(
		&id, &studentDBID, &studentNumber, &firstName, &lastName,
		&course, &year, &docType, &purpose, &copies, &status, &requestedAt,
		&processedAt, &notes, &docPath,
	)

	if err != nil {
		fmt.Println("❌ Query error:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	studentName := firstName + " " + lastName

	c.JSON(http.StatusOK, gin.H{
		"request_id":     id,
		"student_id":     studentDBID,
		"student_number": studentNumber,
		"student_name":   studentName,
		"course":         course,
		"year":           year,
		"document_type":  docType,
		"purpose":        purpose,
		"copies":         copies,
		"status":         status,
		"requested_at":   requestedAt,
		"processed_at":   processedAt,
		"notes":          notes,
		"document_path":  docPath,
	})
}

// ===================== GET ALL GRADES (FOR RECORDS OFFICER) =====================

func RecordsGetAllGrades(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Optional filters
	studentID := c.Query("student_id")
	subjectID := c.Query("subject_id")
	courseID := c.Query("course_id")
	yearLevel := c.Query("year_level")

	query := `
		SELECT 
			g.id as grade_id,
			st.id as student_db_id,
			IFNULL(st.student_id, 'N/A') as student_number,
			IFNULL(st.first_name, 'Unknown') as first_name,
			IFNULL(st.last_name, 'Student') as last_name,
			IFNULL(st.email, 'N/A') as email,
			IFNULL(c.course_name, 'N/A') as course,
			IFNULL(c.code, 'N/A') as course_code,
			IFNULL(sa.year_level, 0) as year_level,
			s.subject_name,
			s.code as subject_code,
			g.teacher_id,
			g.prelim,
			g.midterm,
			g.finals,
			CASE 
				WHEN g.prelim IS NOT NULL AND g.midterm IS NOT NULL AND g.finals IS NOT NULL 
				THEN ROUND((g.prelim + g.midterm + g.finals) / 3, 2)
				ELSE NULL
			END as average,
			IFNULL(g.remarks, '') as remarks,
			IFNULL(g.is_released, FALSE) as is_released,
			DATE_FORMAT(g.created_at, '%Y-%m-%d %H:%i:%s') as submitted_at,
			DATE_FORMAT(g.updated_at, '%Y-%m-%d %H:%i:%s') as updated_at,
			DATE_FORMAT(g.released_at, '%Y-%m-%d %H:%i:%s') as released_at
		FROM grades g
		INNER JOIN students st ON g.student_id = st.id
		INNER JOIN subjects s ON g.subject_id = s.id
		LEFT JOIN student_academic sa ON st.id = sa.student_id
		LEFT JOIN courses c ON sa.course = c.id
		WHERE 1=1
	`

	args := []interface{}{}

	// Apply filters
	if studentID != "" {
		query += " AND st.id = ?"
		args = append(args, studentID)
	}

	if subjectID != "" {
		query += " AND s.id = ?"
		args = append(args, subjectID)
	}

	if courseID != "" {
		query += " AND c.id = ?"
		args = append(args, courseID)
	}

	if yearLevel != "" {
		query += " AND sa.year_level = ?"
		args = append(args, yearLevel)
	}

	query += " ORDER BY st.last_name, st.first_name, s.subject_name"

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		fmt.Println("❌ Database query error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch grades"})
		return
	}
	defer rows.Close()

	var grades []gin.H

	for rows.Next() {
		var (
			gradeID, studentDBID, yearLevel, teacherID   int
			studentNumber, firstName, lastName, email    string
			course, courseCode, subjectName, subjectCode string
			prelim, midterm, finals, average             *float64
			remarks, submittedAt, updatedAt              string
			isReleased                                   bool
			releasedAt                                   *string
		)

		err := rows.Scan(
			&gradeID, &studentDBID, &studentNumber, &firstName, &lastName, &email,
			&course, &courseCode, &yearLevel, &subjectName, &subjectCode, &teacherID,
			&prelim, &midterm, &finals, &average, &remarks, &isReleased,
			&submittedAt, &updatedAt, &releasedAt,
		)

		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		studentName := firstName + " " + lastName

		// Get teacher name in a separate query
		var teacherName string
		err = config.DB.QueryRow(`
			SELECT IFNULL(username, CONCAT('Teacher #', id))
			FROM users 
			WHERE id = ? AND role = 'teacher'
		`, teacherID).Scan(&teacherName)

		if err != nil {
			teacherName = fmt.Sprintf("Teacher #%d", teacherID)
		}

		grades = append(grades, gin.H{
			"grade_id":       gradeID,
			"student_id":     studentDBID,
			"student_number": studentNumber,
			"student_name":   studentName,
			"email":          email,
			"course":         course,
			"course_code":    courseCode,
			"year_level":     yearLevel,
			"subject":        subjectName,
			"subject_code":   subjectCode,
			"teacher_name":   teacherName,
			"prelim":         prelim,
			"midterm":        midterm,
			"finals":         finals,
			"average":        average,
			"remarks":        remarks,
			"is_released":    isReleased,
			"submitted_at":   submittedAt,
			"updated_at":     updatedAt,
			"released_at":    releasedAt,
		})
	}

	if grades == nil {
		grades = []gin.H{}
	}

	fmt.Printf("✅ Found %d grades\n", len(grades))

	c.JSON(http.StatusOK, gin.H{
		"grades": grades,
		"total":  len(grades),
	})
}

// ===================== RELEASE GRADE =====================

func RecordsReleaseGrade(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	gradeIDStr := c.Param("grade_id")
	gradeID, err := strconv.Atoi(gradeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid grade_id"})
		return
	}

	// Get the action from request body
	var input struct {
		Action string `json:"action"` // "release" or "hold"
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if input.Action != "release" && input.Action != "hold" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'release' or 'hold'"})
		return
	}

	// Check if grade exists
	var exists bool
	err = config.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM grades WHERE id = ?)", gradeID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "grade not found"})
		return
	}

	// Update the is_released status
	isReleased := input.Action == "release"

	var err2 error
	if isReleased {
		_, err2 = config.DB.Exec(`
			UPDATE grades 
			SET is_released = ?,
			    released_at = NOW()
			WHERE id = ?
		`, isReleased, gradeID)
	} else {
		_, err2 = config.DB.Exec(`
			UPDATE grades 
			SET is_released = ?,
			    released_at = NULL
			WHERE id = ?
		`, isReleased, gradeID)
	}

	err = err2

	if err != nil {
		fmt.Println("❌ Update error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update grade status"})
		return
	}

	message := "Grade released successfully"
	if input.Action == "hold" {
		message = "Grade put on hold successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     message,
		"grade_id":    gradeID,
		"is_released": isReleased,
	})
}

// ===================== GET STUDENT GRADE DETAILS =====================

func RecordsGetStudentGrades(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	studentIDStr := c.Param("student_id")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student_id"})
		return
	}

	// Get student info
	var (
		studentNumber, firstName, lastName, email string
		course, courseCode                        string
		yearLevel                                 int
	)

	err = config.DB.QueryRow(`
		SELECT 
			IFNULL(st.student_id, 'N/A'),
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
	`, studentID).Scan(&studentNumber, &firstName, &lastName, &email, &course, &courseCode, &yearLevel)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// Get all grades for this student
	query := `
		SELECT 
			g.id as grade_id,
			s.subject_name,
			s.code as subject_code,
			g.teacher_id,
			g.prelim,
			g.midterm,
			g.finals,
			CASE 
				WHEN g.prelim IS NOT NULL AND g.midterm IS NOT NULL AND g.finals IS NOT NULL 
				THEN ROUND((g.prelim + g.midterm + g.finals) / 3, 2)
				ELSE NULL
			END as average,
			IFNULL(g.remarks, '') as remarks,
			IFNULL(g.is_released, FALSE) as is_released,
			DATE_FORMAT(g.created_at, '%Y-%m-%d %H:%i:%s') as submitted_at,
			DATE_FORMAT(g.updated_at, '%Y-%m-%d %H:%i:%s') as updated_at,
			DATE_FORMAT(g.released_at, '%Y-%m-%d %H:%i:%s') as released_at
		FROM grades g
		INNER JOIN subjects s ON g.subject_id = s.id
		WHERE g.student_id = ?
		ORDER BY s.subject_name
	`

	rows, err := config.DB.Query(query, studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch grades"})
		return
	}
	defer rows.Close()

	var grades []gin.H

	for rows.Next() {
		var (
			gradeID, teacherID               int
			subjectName, subjectCode         string
			prelim, midterm, finals, average *float64
			remarks, submittedAt, updatedAt  string
			isReleased                       bool
			releasedAt                       *string
		)

		err := rows.Scan(
			&gradeID, &subjectName, &subjectCode, &teacherID,
			&prelim, &midterm, &finals, &average, &remarks, &isReleased,
			&submittedAt, &updatedAt, &releasedAt,
		)

		if err != nil {
			continue
		}

		// Get teacher name
		var teacherName string
		err = config.DB.QueryRow(`
			SELECT IFNULL(username, CONCAT('Teacher #', id))
			FROM users 
			WHERE id = ? AND role = 'teacher'
		`, teacherID).Scan(&teacherName)

		if err != nil {
			teacherName = fmt.Sprintf("Teacher #%d", teacherID)
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
			"is_released":  isReleased,
			"submitted_at": submittedAt,
			"updated_at":   updatedAt,
			"released_at":  releasedAt,
		})
	}

	if grades == nil {
		grades = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"student": gin.H{
			"student_id":     studentID,
			"student_number": studentNumber,
			"name":           firstName + " " + lastName,
			"email":          email,
			"course":         course,
			"course_code":    courseCode,
			"year_level":     yearLevel,
		},
		"grades": grades,
		"total":  len(grades),
	})
}

// ===================== POST ANNOUNCEMENT =====================

func RecordsPostAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	recordsOfficerID := c.GetInt("user_id")

	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")
	priority := c.PostForm("priority")              // low, normal, high, urgent
	targetAudience := c.PostForm("target_audience") // all, students, teachers

	if title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and content are required"})
		return
	}

	// Set defaults
	if priority == "" {
		priority = "normal"
	}
	if targetAudience == "" {
		targetAudience = "all"
	}

	// Validate priority
	validPriorities := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
	if !validPriorities[priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
		return
	}

	// Validate target audience
	validAudiences := map[string]bool{"all": true, "students": true, "teachers": true}
	if !validAudiences[targetAudience] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target audience"})
		return
	}

	// Handle optional image upload
	var imageName, imagePath sql.NullString
	var imageSize sql.NullInt64

	file, err := c.FormFile("image")
	if err == nil {
		// Image was provided — validate it
		allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"}

		fileHeader, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read image"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "image must be JPEG, PNG, GIF, or WebP"})
			return
		}

		// 10MB max
		if file.Size > int64(10*1024*1024) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image too large. Max 10MB"})
			return
		}

		uploadDir := "./uploads/records_announcements"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
			return
		}

		fileExt := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("%d_%d_%s%s", recordsOfficerID, time.Now().Unix(),
			strings.ReplaceAll(uuid.New().String(), "-", ""), fileExt)
		savedPath := filepath.Join(uploadDir, fileName)

		if err := c.SaveUploadedFile(file, savedPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image"})
			return
		}

		imageName = sql.NullString{String: file.Filename, Valid: true}
		imagePath = sql.NullString{String: savedPath, Valid: true}
		imageSize = sql.NullInt64{Int64: file.Size, Valid: true}
	}

	result, err := config.DB.Exec(`
		INSERT INTO records_announcements 
		(records_officer_id, title, content, image_name, image_path, image_size, priority, target_audience, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, NOW(), NOW())
	`, recordsOfficerID, title, content, imageName, imagePath, imageSize, priority, targetAudience)

	if err != nil {
		fmt.Println("❌ Insert error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to post announcement"})
		return
	}

	newID, _ := result.LastInsertId()

	response := gin.H{
		"message":         "announcement posted successfully",
		"announcement_id": newID,
		"title":           title,
		"priority":        priority,
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

// ===================== GET ALL ANNOUNCEMENTS =====================

func RecordsGetAnnouncements(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Optional filters
	targetFilter := c.Query("target_audience")
	priorityFilter := c.Query("priority")
	activeFilter := c.Query("is_active")

	query := `
		SELECT 
			ra.id,
			ra.records_officer_id,
			u.username as officer_name,
			ra.title,
			ra.content,
			ra.image_name,
			ra.image_path,
			ra.image_size,
			ra.priority,
			ra.target_audience,
			ra.is_active,
			DATE_FORMAT(ra.created_at, '%Y-%m-%d %H:%i:%s') as created_at,
			DATE_FORMAT(ra.updated_at, '%Y-%m-%d %H:%i:%s') as updated_at
		FROM records_announcements ra
		INNER JOIN users u ON ra.records_officer_id = u.id
		WHERE 1=1
	`

	args := []interface{}{}

	if targetFilter != "" {
		query += " AND ra.target_audience = ?"
		args = append(args, targetFilter)
	}

	if priorityFilter != "" {
		query += " AND ra.priority = ?"
		args = append(args, priorityFilter)
	}

	if activeFilter != "" {
		query += " AND ra.is_active = ?"
		args = append(args, activeFilter == "true")
	}

	query += " ORDER BY ra.created_at DESC"

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		fmt.Println("❌ Database query error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch announcements"})
		return
	}
	defer rows.Close()

	var announcements []gin.H

	for rows.Next() {
		var (
			id, recordsOfficerID        int
			officerName, title, content string
			imageName, imagePath        sql.NullString
			imageSize                   sql.NullInt64
			priority, targetAudience    string
			isActive                    bool
			createdAt, updatedAt        string
		)

		err := rows.Scan(
			&id, &recordsOfficerID, &officerName, &title, &content,
			&imageName, &imagePath, &imageSize, &priority, &targetAudience,
			&isActive, &createdAt, &updatedAt,
		)

		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		announcement := gin.H{
			"id":                 id,
			"records_officer_id": recordsOfficerID,
			"officer_name":       officerName,
			"title":              title,
			"content":            content,
			"priority":           priority,
			"target_audience":    targetAudience,
			"is_active":          isActive,
			"created_at":         createdAt,
			"updated_at":         updatedAt,
			"image_url":          nil,
			"image_size":         nil,
		}

		if imagePath.Valid {
			cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
			if strings.HasPrefix(cleanPath, "./") {
				cleanPath = cleanPath[2:]
			}
			announcement["image_url"] = "/" + cleanPath
			announcement["image_size"] = imageSize.Int64
		}

		announcements = append(announcements, announcement)
	}

	if announcements == nil {
		announcements = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"announcements": announcements,
		"total":         len(announcements),
	})
}

// ===================== GET SINGLE ANNOUNCEMENT =====================

func RecordsGetAnnouncementDetails(c *gin.Context) {
	role := c.GetString("role")
	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	announcementID := c.Param("id")

	var (
		id, recordsOfficerID        int
		officerName, title, content string
		imageName, imagePath        sql.NullString
		imageSize                   sql.NullInt64
		priority, targetAudience    string
		isActive                    bool
		createdAt, updatedAt        string
	)

	err := config.DB.QueryRow(`
		SELECT 
			ra.id,
			ra.records_officer_id,
			u.username as officer_name,
			ra.title,
			ra.content,
			ra.image_name,
			ra.image_path,
			ra.image_size,
			ra.priority,
			ra.target_audience,
			ra.is_active,
			DATE_FORMAT(ra.created_at, '%Y-%m-%d %H:%i:%s') as created_at,
			DATE_FORMAT(ra.updated_at, '%Y-%m-%d %H:%i:%s') as updated_at
		FROM records_announcements ra
		INNER JOIN users u ON ra.records_officer_id = u.id
		WHERE ra.id = ?
	`, announcementID).Scan(
		&id, &recordsOfficerID, &officerName, &title, &content,
		&imageName, &imagePath, &imageSize, &priority, &targetAudience,
		&isActive, &createdAt, &updatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	response := gin.H{
		"id":                 id,
		"records_officer_id": recordsOfficerID,
		"officer_name":       officerName,
		"title":              title,
		"content":            content,
		"priority":           priority,
		"target_audience":    targetAudience,
		"is_active":          isActive,
		"created_at":         createdAt,
		"updated_at":         updatedAt,
		"image_url":          nil,
		"image_size":         nil,
	}

	if imagePath.Valid {
		cleanPath := strings.ReplaceAll(imagePath.String, "\\", "/")
		if strings.HasPrefix(cleanPath, "./") {
			cleanPath = cleanPath[2:]
		}
		response["image_url"] = "/" + cleanPath
		response["image_size"] = imageSize.Int64
	}

	c.JSON(http.StatusOK, response)
}

// ===================== UPDATE ANNOUNCEMENT =====================

func RecordsUpdateAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	recordsOfficerID := c.GetInt("user_id")

	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	announcementID := c.Param("id")

	// Verify ownership
	var exists bool
	err := config.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM records_announcements WHERE id = ? AND records_officer_id = ?)
	`, announcementID, recordsOfficerID).Scan(&exists)

	if err != nil || !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "announcement not found or not owned by you"})
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")
	priority := c.PostForm("priority")
	targetAudience := c.PostForm("target_audience")

	if title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and content are required"})
		return
	}

	// Validate priority if provided
	if priority != "" {
		validPriorities := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
		if !validPriorities[priority] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
			return
		}
	}

	// Validate target audience if provided
	if targetAudience != "" {
		validAudiences := map[string]bool{"all": true, "students": true, "teachers": true}
		if !validAudiences[targetAudience] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target audience"})
			return
		}
	}

	// Update database
	_, err = config.DB.Exec(`
		UPDATE records_announcements 
		SET title = ?, content = ?, priority = ?, target_audience = ?, updated_at = NOW()
		WHERE id = ?
	`, title, content, priority, targetAudience, announcementID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update announcement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "announcement updated successfully",
	})
}

// ===================== TOGGLE ANNOUNCEMENT STATUS =====================

func RecordsToggleAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	recordsOfficerID := c.GetInt("user_id")

	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	announcementID := c.Param("id")

	// Verify ownership
	var currentStatus bool
	err := config.DB.QueryRow(`
		SELECT is_active FROM records_announcements 
		WHERE id = ? AND records_officer_id = ?
	`, announcementID, recordsOfficerID).Scan(&currentStatus)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "announcement not found or not owned by you"})
		return
	}

	newStatus := !currentStatus

	_, err = config.DB.Exec(`
		UPDATE records_announcements 
		SET is_active = ?, updated_at = NOW()
		WHERE id = ?
	`, newStatus, announcementID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle announcement"})
		return
	}

	statusText := "deactivated"
	if newStatus {
		statusText = "activated"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("announcement %s successfully", statusText),
		"is_active": newStatus,
	})
}

// ===================== DELETE ANNOUNCEMENT =====================

func RecordsDeleteAnnouncement(c *gin.Context) {
	role := c.GetString("role")
	recordsOfficerID := c.GetInt("user_id")

	if role != "records" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	announcementID := c.Param("id")

	var filePath sql.NullString
	err := config.DB.QueryRow(`
		SELECT image_path FROM records_announcements 
		WHERE id = ? AND records_officer_id = ?
	`, announcementID, recordsOfficerID).Scan(&filePath)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found or not owned by you"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = config.DB.Exec(`DELETE FROM records_announcements WHERE id = ?`, announcementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete announcement"})
		return
	}

	// Delete image file if exists
	if filePath.Valid {
		os.Remove(filePath.String)
	}

	c.JSON(http.StatusOK, gin.H{"message": "announcement deleted successfully"})
}
