package controllers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"student-portal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("secret")

// ========================== LOGIN ==========================
func Login(c *gin.Context) {
	var req struct {
		LoginID  string `json:"login_id" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var id int
	var username, hashedPassword, role, status string
	var studentID string

	// ---- check users table
	err := config.DB.QueryRow(`
			SELECT id, username, password, role
			FROM users
			WHERE username = ?
			LIMIT 1
		`, req.LoginID).Scan(&id, &username, &hashedPassword, &role)

	// ---- if not found, check students table
	if err == sql.ErrNoRows {
		var firstName, lastName string
		err = config.DB.QueryRow(`
				SELECT id, student_id, password, first_name, last_name, status
				FROM students
				WHERE student_id = ?
				LIMIT 1
			`, req.LoginID).Scan(&id, &studentID, &hashedPassword, &firstName, &lastName, &status)

		if err == nil {
			role = "student"
			username = firstName + " " + lastName

			if strings.ToLower(strings.TrimSpace(status)) != "approved" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "account not yet approved"})
				return
			}
		}
	}

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid login ID or password"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid login ID or password"})
		return
	}

	role = strings.ToLower(strings.TrimSpace(role))

	// ✅ JWT claims
	claims := jwt.MapClaims{
		"user_id": id,
		"name":    username,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	if role == "student" {
		claims["student_id"] = studentID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token creation failed"})
		return
	}

	// ---- safer cookie
	c.SetCookie("jwt", tokenString, 86400, "/", "", false, true)

	redirect := "/login"
	switch role {
	case "admin":
		redirect = "/admin/dashboard"
	case "teacher":
		redirect = "/teacher/dashboard"
	case "student":
		redirect = "/student/dashboard"
	case "registrar":
		redirect = "/registrar/dashboard"
	case "cashier":
		redirect = "/cashier/dashboard"
	case "records":
		redirect = "/records/dashboard"
	case "faculty":
		redirect = "/faculty/dashboard"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "login successful",
		"token":    tokenString,
		"role":     role,
		"redirect": redirect,
	})
}

// ===================== REGISTER STUDENT =====================

type StudentRegisterRequest struct {
	StudentID  string `json:"student_id"`
	Password   string `json:"password"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name"`
	Age        int    `json:"age"`

	ContactNumber string `json:"contact_number"`
	Email         string `json:"email"`
	Address       string `json:"address"`

	FatherFirstName  string `json:"father_first_name"`
	FatherMiddleName string `json:"father_middle_name"`
	FatherLastName   string `json:"father_last_name"`
	FatherOccupation string `json:"father_occupation"`
	FatherContact    string `json:"father_contact_number"`
	FatherAddress    string `json:"father_address"`

	MotherFirstName  string `json:"mother_first_name"`
	MotherMiddleName string `json:"mother_middle_name"`
	MotherLastName   string `json:"mother_last_name"`
	MotherOccupation string `json:"mother_occupation"`
	MotherContact    string `json:"mother_contact_number"`
	MotherAddress    string `json:"mother_address"`

	LastSchool     string   `json:"last_school_attended"`
	LastSchoolYear string   `json:"last_school_year"`
	Course         string   `json:"course"`
	Subjects       []string `json:"subjects"`
	YearLevel      string   `json:"year_level"`
	Semester       string   `json:"semester"`

	ScholarshipStatus string `json:"scholarship_status"`
}

func RegisterStudent(c *gin.Context) {
	var req StudentRegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Age == 0 {
		req.Age = 18
	}
	if req.YearLevel == "" {
		req.YearLevel = "1"
	}
	if req.Semester == "" {
		req.Semester = "1st"
	}
	if req.ScholarshipStatus == "" {
		req.ScholarshipStatus = "non-scholar"
	}

	subjectsStr := strings.Join(req.Subjects, ",")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// ---- students table
	res, err := tx.Exec(`
		INSERT INTO students (
			student_id, password, first_name, middle_name, last_name,
			age, contact_number, email, address, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		req.StudentID, hashedPassword, req.FirstName, req.MiddleName, req.LastName,
		req.Age, req.ContactNumber, req.Email, req.Address, "pending",
	)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	studentDBID, _ := res.LastInsertId()

	// ---- student_family table
	_, err = tx.Exec(`
		INSERT INTO student_family (
			student_id,
			father_first_name, father_middle_name, father_last_name,
			father_occupation, father_contact_number, father_address,
			mother_first_name, mother_middle_name, mother_last_name,
			mother_occupation, mother_contact_number, mother_address
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		studentDBID,
		req.FatherFirstName, req.FatherMiddleName, req.FatherLastName,
		req.FatherOccupation, req.FatherContact, req.FatherAddress,
		req.MotherFirstName, req.MotherMiddleName, req.MotherLastName,
		req.MotherOccupation, req.MotherContact, req.MotherAddress,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// ---- student_academic table
	_, err = tx.Exec(`
		INSERT INTO student_academic (
			student_id, last_school_attended, last_school_year,
			course, subjects, year_level, scholarship_status, total_units, semester
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		studentDBID,
		req.LastSchool, req.LastSchoolYear,
		req.Course, subjectsStr, req.YearLevel,
		req.ScholarshipStatus, len(req.Subjects)*3, req.Semester,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Student registered successfully. Awaiting approval.",
	})
}

// ===================== GET SUBJECTS =====================

func GetSubjectsForRegistration(c *gin.Context) {
	course := c.Query("course_id")
	year := c.Query("year_level")
	semester := c.Query("semester") // ✅ NEW: read semester from query

	query := "SELECT id, subject_name, code FROM subjects WHERE 1=1"
	args := []interface{}{}

	if course != "" {
		query += " AND course_id=?"
		args = append(args, course)
	}
	if year != "" {
		query += " AND year_level=?"
		args = append(args, year)
	}
	// ✅ Only filter by semester if one is selected
	if semester != "" {
		query += " AND semester=?"
		args = append(args, semester)
	}

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	subjects := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name, code string
		rows.Scan(&id, &name, &code)
		subjects = append(subjects, map[string]interface{}{
			"id":           id,
			"subject_name": name,
			"code":         code,
		})
	}

	c.JSON(http.StatusOK, subjects)
}
