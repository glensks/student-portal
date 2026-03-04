package controllers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"student-portal/config"
	"student-portal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

/* =======================
STUDENT STRUCTS
======================= */

type StudentSubject struct {
	SubjectCode string `json:"subject_code"`
	SubjectName string `json:"subject_name"`
}

type Student struct {
	ID                int              `json:"id"`         // ✅ EXPOSED — frontend uses this to identify pending students
	StudentID         string           `json:"student_id"` // NULL for pending students
	FirstName         string           `json:"first_name"`
	LastName          string           `json:"last_name"`
	FullName          string           `json:"full_name"`
	Course            string           `json:"course"`
	YearLevel         string           `json:"year_level"`
	Semester          string           `json:"semester"`
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
	Semester       string       `json:"semester"`
	Scholarship    string       `json:"scholarship_status"`
	TotalUnits     int          `json:"total_units"`
	OtherFees      []PaymentFee `json:"other_fees"`
	OtherFeesTotal int          `json:"other_fees_total"`
	TotalAmount    int          `json:"total_amount"`
	TotalPaid      int          `json:"total_paid"`
	Status         string       `json:"status"`
}

type ApproveWithAssessmentRequest struct {
	// For pending students (no student_id yet), frontend sends the DB row id as a string
	StudentDBID string       `json:"student_db_id"`
	Semester    string       `json:"semester"`
	SchoolYear  string       `json:"school_year"`
	OtherFees   []PaymentFee `json:"other_fees"`
}

/* =======================
HELPERS
======================= */

// generateStudentID generates a unique student ID in the format YYYY-NNNNN
// e.g. 2025-00001, 2025-00002, etc.
func generateStudentID(tx *sql.Tx) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("%d-", year)

	var count int
	err := tx.QueryRow(
		`SELECT COUNT(*) FROM students WHERE student_id LIKE ?`,
		prefix+"%",
	).Scan(&count)
	if err != nil {
		return "", err
	}

	// Keep incrementing until a non-taken ID is found (handles gaps/deletions)
	for {
		candidate := fmt.Sprintf("%d-%05d", year, count+1)
		var exists int
		tx.QueryRow(
			`SELECT COUNT(*) FROM students WHERE student_id = ?`,
			candidate,
		).Scan(&exists)
		if exists == 0 {
			return candidate, nil
		}
		count++
	}
}

// generateDefaultPassword returns a random 8-character password.
// Uses uppercase letters + digits, excludes ambiguous chars (0/O, 1/I/L).
func generateDefaultPassword() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	result := make([]byte, 8)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

/* =======================
STUDENT LISTING
======================= */

func RegistrarGetStudentsByStatus(c *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))

	rows, err := config.DB.Query(`
		SELECT 
			st.id,
			IFNULL(st.student_id,''),
			st.first_name,
			st.last_name,
			IFNULL(c.course_name,''),
			IFNULL(sa.year_level,''),
			IFNULL(sa.scholarship_status,''),
			st.status,
			IFNULL(sa.total_units,0),
			IFNULL(sa.semester,''),
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
			&s.Semester,
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
					`SELECT IFNULL(code,''), subject_name FROM subjects WHERE id=?`,
					strings.TrimSpace(id),
				).Scan(&subj.SubjectCode, &subj.SubjectName)
				s.Subjects = append(s.Subjects, subj)
			}
		}

		students = append(students, s)
	}

	c.JSON(200, gin.H{"students": students})
}

/* ====================================================
APPROVE WITH ASSESSMENT
(Initial Enrollment — new student registered via portal)

Flow:
  1. Frontend sends student.id (DB row id) — NOT student_id (which is NULL for pending)
  2. Look up student by DB row id (st.id = ?)
  3. Generate Student ID (YYYY-NNNNN format)
  4. Generate random default password, bcrypt hash it
  5. Save student_id + password to students table
  6. Create billing record
  7. Send email with credentials + billing statement
==================================================== */

func RegistrarApproveWithAssessment(c *gin.Context) {
	var req ApproveWithAssessmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if req.StudentDBID == "" {
		c.JSON(400, gin.H{"error": "student_db_id is required"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var studentDBID, totalUnits int
	var scholarshipStatus, studentEmail, firstName, lastName string

	// ===== GET STUDENT INFO BY DB ROW ID =====
	// Pending students have student_id = NULL in DB.
	// The frontend sends s.id (the DB row id) as student_db_id.
	err = tx.QueryRow(`
		SELECT st.id,
		       IFNULL(sa.total_units, 0),
		       IFNULL(sa.scholarship_status, ''),
		       st.email,
		       st.first_name,
		       st.last_name
		FROM students st
		LEFT JOIN student_academic sa ON sa.student_id = st.id
		WHERE st.id = ?
	`, req.StudentDBID).Scan(
		&studentDBID,
		&totalUnits,
		&scholarshipStatus,
		&studentEmail,
		&firstName,
		&lastName,
	)

	if err != nil {
		tx.Rollback()
		c.JSON(404, gin.H{"error": "Student not found"})
		return
	}

	// ===== GENERATE STUDENT ID =====
	generatedStudentID, err := generateStudentID(tx)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to generate student ID: " + err.Error()})
		return
	}

	// ===== GENERATE DEFAULT PASSWORD =====
	rawPassword, err := generateDefaultPassword()
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to generate password"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}

	// ===== COMPUTE TUITION =====
	tuition := 800 * totalUnits
	if strings.ToLower(strings.TrimSpace(scholarshipStatus)) == "scholar" {
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

	// ===== INSERT OTHER FEES =====
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
		UPDATE student_payments SET total_amount = ? WHERE id = ?
	`, totalAmount, paymentID)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// ===== APPROVE STUDENT + ASSIGN STUDENT ID + SET PASSWORD =====
	_, err = tx.Exec(`
		UPDATE students
		SET status     = 'approved',
		    student_id = ?,
		    password   = ?
		WHERE id = ?
	`, generatedStudentID, string(hashedPassword), studentDBID)

	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Failed to update student: " + err.Error()})
		return
	}

	tx.Commit()

	// ===== BUILD & SEND EMAIL =====
	otherFeesRows := ""
	for _, fee := range req.OtherFees {
		if fee.Amount > 0 {
			otherFeesRows += fmt.Sprintf(`
				<tr>
					<td style="padding:8px 12px;color:#555;border-top:1px solid #f0f0f0;">%s</td>
					<td style="padding:8px 12px;text-align:right;color:#555;border-top:1px solid #f0f0f0;">&#8369;%d</td>
				</tr>`, fee.FeeName, fee.Amount)
		}
	}

	emailBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Enrollment Approved</title>
</head>
<body style="margin:0;padding:0;background-color:#f0f4f8;font-family:'Segoe UI',Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f0f4f8;padding:40px 20px;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0"
	style="background:#ffffff;border-radius:12px;overflow:hidden;
	       box-shadow:0 4px 20px rgba(0,0,0,0.08);max-width:600px;width:100%%;">

	<!-- HEADER -->
	<tr>
		<td style="background:linear-gradient(135deg,#1b4332,#2d6a4f);padding:36px 40px;text-align:center;">
			<div style="font-size:52px;margin-bottom:12px;">🎓</div>
			<h1 style="margin:0;color:#ffffff;font-size:26px;font-weight:700;letter-spacing:0.3px;">
				Enrollment Approved!
			</h1>
			<p style="margin:10px 0 0;color:#95d5b2;font-size:14px;line-height:1.5;">
				Welcome to the University of Manila — your application has been approved.
			</p>
		</td>
	</tr>

	<!-- GREETING -->
	<tr>
		<td style="padding:32px 40px 0;">
			<p style="margin:0;font-size:15px;color:#333;line-height:1.7;">
				Dear <strong>%s %s</strong>,<br><br>
				Congratulations! Your enrollment application has been reviewed and approved by the
				Registrar's Office. Your student account has been created — please find your
				<strong>login credentials</strong> and <strong>billing statement</strong> below.
			</p>
		</td>
	</tr>

	<!-- CREDENTIALS CARD -->
	<tr>
		<td style="padding:24px 40px 0;">
			<table width="100%%" cellpadding="0" cellspacing="0"
				style="background:#f0fff4;border:2px solid #52b788;border-radius:10px;overflow:hidden;">
				<tr>
					<td style="padding:12px 18px;background:#d1fae5;font-size:12px;font-weight:700;
						color:#065f46;text-transform:uppercase;letter-spacing:0.8px;">
						🔐 Your Student Account Credentials
					</td>
				</tr>
				<tr>
					<td>
						<table width="100%%" cellpadding="0" cellspacing="0">
							<tr>
								<td style="padding:18px;border-bottom:1px solid #d1fae5;width:50%%;">
									<span style="color:#888;font-size:11px;display:block;margin-bottom:6px;
										text-transform:uppercase;letter-spacing:0.5px;">Student ID</span>
									<strong style="font-size:24px;color:#1b4332;font-family:monospace;
										letter-spacing:0.06em;">%s</strong>
								</td>
								<td style="padding:18px;border-bottom:1px solid #d1fae5;
									border-left:1px solid #d1fae5;">
									<span style="color:#888;font-size:11px;display:block;margin-bottom:6px;
										text-transform:uppercase;letter-spacing:0.5px;">Default Password</span>
									<strong style="font-size:24px;color:#1b4332;font-family:monospace;
										letter-spacing:0.15em;">%s</strong>
								</td>
							</tr>
							<tr>
								<td colspan="2" style="padding:12px 18px;background:#ecfdf5;">
									<p style="margin:0;font-size:12px;color:#065f46;line-height:1.7;">
										⚠️ <strong>Important:</strong> Log in using the credentials above and
										<strong>change your password immediately</strong> after your first login.
										Do not share these credentials with anyone.
									</p>
								</td>
							</tr>
						</table>
					</td>
				</tr>
			</table>
		</td>
	</tr>

	<!-- ENROLLMENT DETAILS -->
	<tr>
		<td style="padding:24px 40px 0;">
			<table width="100%%" cellpadding="0" cellspacing="0"
				style="background:#f5f9ff;border:1px solid #dce8fb;border-radius:10px;overflow:hidden;">
				<tr>
					<td style="padding:12px 18px;background:#e3eefb;font-size:12px;font-weight:700;
						color:#1565c0;text-transform:uppercase;letter-spacing:0.8px;">
						📋 Enrollment Information
					</td>
				</tr>
				<tr>
					<td>
						<table width="100%%" cellpadding="0" cellspacing="0">
							<tr>
								<td style="padding:12px 18px;font-size:14px;color:#444;
									border-bottom:1px solid #e8f0fb;width:50%%;">
									<span style="color:#888;font-size:12px;display:block;margin-bottom:2px;">SEMESTER</span>
									<strong>%s</strong>
								</td>
								<td style="padding:12px 18px;font-size:14px;color:#444;
									border-bottom:1px solid #e8f0fb;">
									<span style="color:#888;font-size:12px;display:block;margin-bottom:2px;">SCHOOL YEAR</span>
									<strong>%s</strong>
								</td>
							</tr>
							<tr>
								<td colspan="2" style="padding:12px 18px;font-size:14px;color:#444;">
									<span style="color:#888;font-size:12px;display:block;margin-bottom:2px;">TOTAL UNITS ENROLLED</span>
									<strong>%d units</strong>
								</td>
							</tr>
						</table>
					</td>
				</tr>
			</table>
		</td>
	</tr>

	<!-- BILLING STATEMENT -->
	<tr>
		<td style="padding:24px 40px 0;">
			<p style="margin:0 0 10px;font-size:12px;font-weight:700;color:#888;
				text-transform:uppercase;letter-spacing:0.8px;">💰 Billing Statement</p>
			<table width="100%%" cellpadding="0" cellspacing="0"
				style="border:1px solid #e0e0e0;border-radius:10px;overflow:hidden;">
				<tr style="background:#f9f9f9;">
					<td style="padding:11px 12px;font-size:12px;font-weight:700;color:#777;
						border-bottom:1px solid #e8e8e8;text-transform:uppercase;letter-spacing:0.5px;">
						Description
					</td>
					<td style="padding:11px 12px;font-size:12px;font-weight:700;color:#777;
						border-bottom:1px solid #e8e8e8;text-align:right;text-transform:uppercase;letter-spacing:0.5px;">
						Amount
					</td>
				</tr>
				<tr>
					<td style="padding:12px;color:#444;font-size:14px;">
						Tuition Fee
						<span style="font-size:12px;color:#999;display:block;">%d units</span>
					</td>
					<td style="padding:12px;text-align:right;color:#444;font-size:14px;">&#8369;%d</td>
				</tr>
				%s
				<tr><td colspan="2" style="padding:0;border-top:2px solid #e3eefb;"></td></tr>
				<tr style="background:#f0f7ff;">
					<td style="padding:14px 12px;font-size:16px;font-weight:700;color:#1565c0;">Total Amount Due</td>
					<td style="padding:14px 12px;font-size:16px;font-weight:700;color:#1565c0;text-align:right;">
						&#8369;%d
					</td>
				</tr>
			</table>
		</td>
	</tr>

	<!-- PAYMENT STATUS -->
	<tr>
		<td style="padding:20px 40px 0;text-align:center;">
			<table cellpadding="0" cellspacing="0" style="display:inline-table;margin:auto;">
				<tr>
					<td style="background:#fff8e1;border:1px solid #ffe082;border-radius:25px;
						padding:10px 28px;font-size:13px;font-weight:700;color:#f57f17;
						text-transform:uppercase;letter-spacing:0.8px;">
						⏱ Payment Status: Unpaid
					</td>
				</tr>
			</table>
		</td>
	</tr>

	<!-- REMINDER -->
	<tr>
		<td style="padding:24px 40px;">
			<table width="100%%" cellpadding="0" cellspacing="0"
				style="background:#fff3e0;border-left:4px solid #fb8c00;border-radius:0 8px 8px 0;">
				<tr>
					<td style="padding:14px 18px;font-size:13px;color:#555;line-height:1.7;">
						🔔 <strong>Reminder:</strong> Please settle your balance at the Cashier's Office or
						through the Student Portal before the payment deadline. Failure to pay on time may
						result in your enrollment being placed on hold.
					</td>
				</tr>
			</table>
		</td>
	</tr>

	<!-- FOOTER -->
	<tr>
		<td style="background:#f5f7fa;padding:20px 40px;text-align:center;border-top:1px solid #e8e8e8;">
			<p style="margin:0 0 4px;font-size:12px;color:#aaa;">
				This is an automated notification from the
				<strong>University of Manila Student Portal</strong>.
				Please do not reply to this email.
			</p>
			<p style="margin:0;font-size:12px;color:#aaa;">
				For concerns or inquiries, please visit or contact the Registrar's Office.
			</p>
		</td>
	</tr>

</table>
</td></tr>
</table>
</body>
</html>
	`,
		firstName, lastName, // greeting
		generatedStudentID, // credentials: student ID
		rawPassword,        // credentials: default password
		req.Semester,       // enrollment: semester
		req.SchoolYear,     // enrollment: school year
		totalUnits,         // enrollment: units
		totalUnits,         // billing: units label
		tuition,            // billing: tuition amount
		otherFeesRows,      // billing: other fees rows
		totalAmount,        // billing: total
	)

	// Run in goroutine but log any error so Railway logs show exact failure
	go func() {
		if err := utils.SendEmail(
			studentEmail,
			"🎓 Enrollment Approved – Your Student ID & Billing Statement",
			emailBody,
		); err != nil {
			log.Printf("❌ APPROVAL EMAIL FAILED to %s: %v", studentEmail, err)
		} else {
			log.Printf("✅ APPROVAL EMAIL SENT to %s", studentEmail)
		}
	}()

	c.JSON(200, gin.H{
		"message":              "Student approved with assessment",
		"generated_student_id": generatedStudentID,
		"payment_id":           paymentID,
		"semester":             req.Semester,
		"school_year":          req.SchoolYear,
		"tuition":              tuition,
		"other_fees":           otherTotal,
		"total_amount":         totalAmount,
		"amount_paid":          0,
		"status":               "unpaid",
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

	validAudiences := map[string]bool{
		"all": true, "pending": true, "approved": true, "enrolled": true,
	}
	if !validAudiences[targetAudience] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_audience. Must be: all, pending, approved, or enrolled"})
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
        INSERT INTO registrar_announcements
            (registrar_id, title, content, target_audience, image_name, image_path, image_size, created_at, updated_at)
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

	targetAudience := c.Query("target_audience")
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

/* ==========================
RE-ENROLLMENT APPLICATIONS
========================== */

// GET /registrar/enrollment-applications?status=pending
func RegistrarGetEnrollmentApplications(c *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status == "" {
		status = "pending"
	}

	rows, err := config.DB.Query(`
        SELECT 
            ea.id,
            ea.student_id,
            st.student_id as student_str_id,
            st.first_name,
            st.last_name,
            IFNULL(c.course_name, ''),
            IFNULL(c.id, 0),
            ea.year_level,
            ea.semester,
            ea.academic_year,
            IFNULL(ea.scholarship_status, ''),
            ea.total_units,
            ea.subjects,
            ea.status,
            IFNULL(ea.remarks, ''),
            DATE_FORMAT(ea.applied_at, '%Y-%m-%d %H:%i:%s')
        FROM enrollment_applications ea
        JOIN students st ON st.id = ea.student_id
        LEFT JOIN courses c ON c.id = ea.course_id
        WHERE ea.status = ?
        ORDER BY ea.applied_at DESC
    `, status)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type EnrollApp struct {
		EnrollmentID      int              `json:"enrollment_id"`
		StudentDBID       int              `json:"-"`
		StudentID         string           `json:"student_id"`
		FirstName         string           `json:"first_name"`
		LastName          string           `json:"last_name"`
		FullName          string           `json:"full_name"`
		Course            string           `json:"course"`
		CourseID          int              `json:"course_id"`
		YearLevel         int              `json:"year_level"`
		Semester          string           `json:"semester"`
		AcademicYear      string           `json:"academic_year"`
		ScholarshipStatus string           `json:"scholarship_status"`
		TotalUnits        int              `json:"total_units"`
		Subjects          []StudentSubject `json:"subjects"`
		Status            string           `json:"status"`
		Remarks           string           `json:"remarks"`
		AppliedAt         string           `json:"applied_at"`
	}

	var apps []EnrollApp

	for rows.Next() {
		var app EnrollApp
		var subjectsStr string

		err := rows.Scan(
			&app.EnrollmentID,
			&app.StudentDBID,
			&app.StudentID,
			&app.FirstName,
			&app.LastName,
			&app.Course,
			&app.CourseID,
			&app.YearLevel,
			&app.Semester,
			&app.AcademicYear,
			&app.ScholarshipStatus,
			&app.TotalUnits,
			&subjectsStr,
			&app.Status,
			&app.Remarks,
			&app.AppliedAt,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		app.FullName = app.FirstName + " " + app.LastName
		app.Subjects = []StudentSubject{}

		if strings.TrimSpace(subjectsStr) != "" {
			ids := strings.Split(subjectsStr, ",")
			for _, id := range ids {
				var subj StudentSubject
				config.DB.QueryRow(
					`SELECT IFNULL(code,''), subject_name FROM subjects WHERE id=?`,
					strings.TrimSpace(id),
				).Scan(&subj.SubjectCode, &subj.SubjectName)
				app.Subjects = append(app.Subjects, subj)
			}
		}

		apps = append(apps, app)
	}

	if apps == nil {
		apps = []EnrollApp{}
	}

	c.JSON(200, gin.H{"applications": apps, "total": len(apps)})
}

// POST /registrar/enrollment-applications/approve
// Re-enrollment: student already HAS credentials — only send billing email
func RegistrarApproveEnrollmentApplication(c *gin.Context) {
	var req struct {
		EnrollmentID int          `json:"enrollment_id"`
		OtherFees    []PaymentFee `json:"other_fees"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var studentDBID, totalUnits, courseID int
	var scholarshipStatus, semester, academicYear, subjectsStr string
	var studentEmail, studentID, firstName, lastName string

	err = tx.QueryRow(`
        SELECT 
            ea.student_id,
            ea.total_units,
            ea.scholarship_status,
            ea.semester,
            ea.academic_year,
            ea.subjects,
            ea.course_id,
            st.email,
            st.student_id as student_str_id,
            st.first_name,
            st.last_name
        FROM enrollment_applications ea
        JOIN students st ON st.id = ea.student_id
        WHERE ea.id = ? AND ea.status = 'pending'
    `, req.EnrollmentID).Scan(
		&studentDBID,
		&totalUnits,
		&scholarshipStatus,
		&semester,
		&academicYear,
		&subjectsStr,
		&courseID,
		&studentEmail,
		&studentID,
		&firstName,
		&lastName,
	)

	if err != nil {
		tx.Rollback()
		c.JSON(404, gin.H{"error": "Enrollment application not found or already processed"})
		return
	}

	// Compute tuition
	tuition := 800 * totalUnits
	if strings.ToLower(scholarshipStatus) == "scholar" {
		tuition = 500 * totalUnits
	}

	// Create payment
	res, err := tx.Exec(`
        INSERT INTO student_payments
            (student_id, total_amount, amount_paid, status, semester, school_year)
        VALUES (?, 0, 0, 'unpaid', ?, ?)
    `, studentDBID, semester, academicYear)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	paymentID, _ := res.LastInsertId()

	// Insert other fees
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

	totalAmount := tuition + otherTotal
	tx.Exec(`UPDATE student_payments SET total_amount = ? WHERE id = ?`, totalAmount, paymentID)

	// Update student_academic
	_, err = tx.Exec(`
        UPDATE student_academic 
        SET semester = ?, subjects = ?, total_units = ?, scholarship_status = ?, course = ?
        WHERE student_id = ?
    `, semester, subjectsStr, totalUnits, scholarshipStatus, courseID, studentDBID)
	if err != nil {
		tx.Exec(`
            INSERT INTO student_academic (student_id, semester, subjects, total_units, scholarship_status, course)
            VALUES (?, ?, ?, ?, ?, ?)
        `, studentDBID, semester, subjectsStr, totalUnits, scholarshipStatus, courseID)
	}

	// Mark application approved
	tx.Exec(`UPDATE enrollment_applications SET status = 'approved', processed_at = NOW() WHERE id = ?`, req.EnrollmentID)
	tx.Exec(`UPDATE students SET status = 'approved' WHERE id = ?`, studentDBID)

	tx.Commit()

	// ===== SEND RE-ENROLLMENT BILLING EMAIL =====
	otherFeesRows := ""
	for _, fee := range req.OtherFees {
		if fee.Amount > 0 {
			otherFeesRows += fmt.Sprintf(`
				<tr>
					<td style="padding:8px 12px;color:#555;border-top:1px solid #f0f0f0;">%s</td>
					<td style="padding:8px 12px;text-align:right;color:#555;border-top:1px solid #f0f0f0;">&#8369;%d</td>
				</tr>`, fee.FeeName, fee.Amount)
		}
	}

	reEnrollEmailBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>Re-Enrollment Approved</title></head>
<body style="margin:0;padding:0;background-color:#f0f4f8;font-family:'Segoe UI',Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f0f4f8;padding:40px 20px;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0"
	style="background:#ffffff;border-radius:12px;overflow:hidden;
	       box-shadow:0 4px 20px rgba(0,0,0,0.08);max-width:600px;width:100%%;">
	<tr>
		<td style="background:linear-gradient(135deg,#1b4332,#2d6a4f);padding:36px 40px;text-align:center;">
			<div style="font-size:52px;margin-bottom:12px;">✅</div>
			<h1 style="margin:0;color:#ffffff;font-size:26px;font-weight:700;">Re-Enrollment Approved!</h1>
			<p style="margin:10px 0 0;color:#95d5b2;font-size:14px;">Your re-enrollment has been approved.</p>
		</td>
	</tr>
	<tr>
		<td style="padding:32px 40px 0;">
			<p style="margin:0;font-size:15px;color:#333;line-height:1.7;">
				Dear <strong>%s %s</strong>,<br><br>
				Your re-enrollment for <strong>%s — %s</strong> has been approved by the Registrar's Office.
				Please review your billing statement below and settle your balance before the payment deadline.
			</p>
		</td>
	</tr>
	<tr>
		<td style="padding:24px 40px 0;">
			<p style="margin:0 0 10px;font-size:12px;font-weight:700;color:#888;
				text-transform:uppercase;letter-spacing:0.8px;">💰 Billing Statement</p>
			<table width="100%%" cellpadding="0" cellspacing="0"
				style="border:1px solid #e0e0e0;border-radius:10px;overflow:hidden;">
				<tr style="background:#f9f9f9;">
					<td style="padding:11px 12px;font-size:12px;font-weight:700;color:#777;
						border-bottom:1px solid #e8e8e8;text-transform:uppercase;">Description</td>
					<td style="padding:11px 12px;font-size:12px;font-weight:700;color:#777;
						border-bottom:1px solid #e8e8e8;text-align:right;text-transform:uppercase;">Amount</td>
				</tr>
				<tr>
					<td style="padding:12px;color:#444;font-size:14px;">
						Tuition Fee<span style="font-size:12px;color:#999;display:block;">%d units</span>
					</td>
					<td style="padding:12px;text-align:right;color:#444;font-size:14px;">&#8369;%d</td>
				</tr>
				%s
				<tr><td colspan="2" style="padding:0;border-top:2px solid #e3eefb;"></td></tr>
				<tr style="background:#f0f7ff;">
					<td style="padding:14px 12px;font-size:16px;font-weight:700;color:#1565c0;">Total Amount Due</td>
					<td style="padding:14px 12px;font-size:16px;font-weight:700;color:#1565c0;text-align:right;">&#8369;%d</td>
				</tr>
			</table>
		</td>
	</tr>
	<tr>
		<td style="padding:20px 40px 0;text-align:center;">
			<table cellpadding="0" cellspacing="0" style="display:inline-table;margin:auto;">
				<tr>
					<td style="background:#fff8e1;border:1px solid #ffe082;border-radius:25px;
						padding:10px 28px;font-size:13px;font-weight:700;color:#f57f17;
						text-transform:uppercase;letter-spacing:0.8px;">
						⏱ Payment Status: Unpaid
					</td>
				</tr>
			</table>
		</td>
	</tr>
	<tr>
		<td style="padding:24px 40px;">
			<table width="100%%" cellpadding="0" cellspacing="0"
				style="background:#fff3e0;border-left:4px solid #fb8c00;border-radius:0 8px 8px 0;">
				<tr>
					<td style="padding:14px 18px;font-size:13px;color:#555;line-height:1.7;">
						🔔 <strong>Reminder:</strong> Please settle your balance before the payment deadline.
					</td>
				</tr>
			</table>
		</td>
	</tr>
	<tr>
		<td style="background:#f5f7fa;padding:20px 40px;text-align:center;border-top:1px solid #e8e8e8;">
			<p style="margin:0;font-size:12px;color:#aaa;">
				Automated notification from the University of Manila Student Portal. Do not reply.
			</p>
		</td>
	</tr>
</table>
</td></tr>
</table>
</body>
</html>`,
		firstName, lastName,
		semester, academicYear,
		totalUnits, tuition,
		otherFeesRows,
		totalAmount,
	)

	go func() {
		if err := utils.SendEmail(
			studentEmail,
			"✅ Re-Enrollment Approved – Billing Statement",
			reEnrollEmailBody,
		); err != nil {
			log.Printf("❌ RE-ENROLL EMAIL FAILED to %s: %v", studentEmail, err)
		} else {
			log.Printf("✅ RE-ENROLL EMAIL SENT to %s", studentEmail)
		}
	}()

	c.JSON(200, gin.H{
		"message":       "Re-enrollment approved successfully",
		"payment_id":    paymentID,
		"enrollment_id": req.EnrollmentID,
		"student_id":    studentID,
		"semester":      semester,
		"school_year":   academicYear,
		"tuition":       tuition,
		"other_fees":    otherTotal,
		"total_amount":  totalAmount,
		"status":        "unpaid",
	})
}

// POST /registrar/enrollment-applications/reject
func RegistrarRejectEnrollmentApplication(c *gin.Context) {
	var req struct {
		EnrollmentID int    `json:"enrollment_id"`
		Remarks      string `json:"remarks"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	_, err := config.DB.Exec(`
        UPDATE enrollment_applications 
        SET status = 'rejected', remarks = ?, processed_at = NOW()
        WHERE id = ? AND status = 'pending'
    `, req.Remarks, req.EnrollmentID)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to reject application"})
		return
	}

	c.JSON(200, gin.H{"message": "Enrollment application rejected"})
}

// POST /registrar/student/status
// Works for both pending (no student_id yet → match by DB id) and existing students
func RegistrarUpdateStudentStatus(c *gin.Context) {
	var req struct {
		StudentID string `json:"student_id"` // DB row id (as string) for pending students
		Action    string `json:"action"`
		Remarks   string `json:"remarks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	var newStatus string
	switch strings.ToLower(req.Action) {
	case "reject":
		newStatus = "rejected"
	case "approve":
		newStatus = "approved"
	default:
		c.JSON(400, gin.H{"error": "Invalid action"})
		return
	}

	// Try matching by DB row id first (pending students), then by student_id column
	result, err := config.DB.Exec(`
        UPDATE students SET status = ? WHERE id = ? OR student_id = ?
    `, newStatus, req.StudentID, req.StudentID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(404, gin.H{"error": "Student not found"})
		return
	}

	c.JSON(200, gin.H{"message": "Student status updated to " + newStatus})
}
