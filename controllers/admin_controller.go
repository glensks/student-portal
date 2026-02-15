package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"student-portal/config"
	"student-portal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// ===== USERS =====

// AdminGetUsers retrieves users by role or all users if no role is specified.
func AdminGetUsers(c *gin.Context) {
	role := c.Query("role")
	status := c.Query("status")

	query := `SELECT id, username, first_name, middle_name, surname, email, 
	          contact_number, role, status, created_at FROM users WHERE 1=1`
	args := []interface{}{}

	if role != "" {
		query += " AND role=?"
		args = append(args, role)
	}

	if status != "" {
		query += " AND status=?"
		args = append(args, status)
	}

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	users := []gin.H{}
	for rows.Next() {
		var id int
		var username, role, status string
		var firstName, middleName, surname, email, contactNumber sql.NullString
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &firstName, &middleName, &surname,
			&email, &contactNumber, &role, &status, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		users = append(users, gin.H{
			"id":             id,
			"username":       username,
			"first_name":     firstName.String,
			"middle_name":    middleName.String,
			"surname":        surname.String,
			"email":          email.String,
			"contact_number": contactNumber.String,
			"role":           role,
			"status":         status,
			"created_at":     createdAt,
		})
	}
	c.JSON(http.StatusOK, users)
}

// AdminCreateUser creates a new user (student, teacher, etc.) by admin.
func AdminCreateUser(c *gin.Context) {
	var input struct {
		Username      string `json:"username" binding:"required"`
		Password      string `json:"password" binding:"required"`
		FirstName     string `json:"first_name" binding:"required"`
		MiddleName    string `json:"middle_name"`
		Surname       string `json:"surname" binding:"required"`
		Email         string `json:"email" binding:"required,email"`
		ContactNumber string `json:"contact_number" binding:"required"`
		Role          string `json:"role" binding:"required"`
	}

	// Bind JSON body to input struct.
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log incoming data for debugging purposes.
	fmt.Println("Incoming data:", input)

	// Check if username already exists in the database.
	var count int
	if err := config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", input.Username).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If the username exists, return an error.
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	// Check if email already exists in the database.
	if err := config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email=?", input.Email).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If the email exists, return an error.
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	// Hash the password before saving it to the database.
	hash := utils.HashPassword(input.Password)

	// Insert new user into the database.
	_, err := config.DB.Exec(`INSERT INTO users 
		(username, password, first_name, middle_name, surname, email, contact_number, role, status) 
		VALUES (?,?,?,?,?,?,?,?,?)`,
		input.Username, hash, input.FirstName, input.MiddleName, input.Surname,
		input.Email, input.ContactNumber, input.Role, "active")

	if err != nil {
		// Log the error for debugging.
		fmt.Println("Error while inserting user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return success message.
	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// AdminEditUser allows the admin to update a user's details.
func AdminEditUser(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Username      string `json:"username" binding:"required"`
		Password      string `json:"password"`
		FirstName     string `json:"first_name" binding:"required"`
		MiddleName    string `json:"middle_name"`
		Surname       string `json:"surname" binding:"required"`
		Email         string `json:"email" binding:"required,email"`
		ContactNumber string `json:"contact_number" binding:"required"`
		Role          string `json:"role" binding:"required"`
	}

	// Bind the request body to the input struct.
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update user in the database.
	var err error
	if input.Password != "" {
		// If password is provided, hash it before updating.
		hash := utils.HashPassword(input.Password)
		_, err = config.DB.Exec(`UPDATE users SET username=?, password=?, first_name=?, 
			middle_name=?, surname=?, email=?, contact_number=?, role=? WHERE id=?`,
			input.Username, hash, input.FirstName, input.MiddleName, input.Surname,
			input.Email, input.ContactNumber, input.Role, id)
	} else {
		// If no password is provided, update other fields only.
		_, err = config.DB.Exec(`UPDATE users SET username=?, first_name=?, middle_name=?, 
			surname=?, email=?, contact_number=?, role=? WHERE id=?`,
			input.Username, input.FirstName, input.MiddleName, input.Surname,
			input.Email, input.ContactNumber, input.Role, id)
	}

	if err != nil {
		// Log the error for debugging.
		fmt.Println("Error while updating user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return success message.
	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

// AdminDeleteUser allows the admin to delete a user.
func AdminDeleteUser(c *gin.Context) {
	id := c.Param("id")
	_, err := config.DB.Exec("DELETE FROM users WHERE id=?", id)
	if err != nil {
		// Log the error for debugging.
		fmt.Println("Error while deleting user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// ===== ADDITIONAL ADMIN IT MANAGEMENT FUNCTIONS =====

// AdminResetUserPassword allows admin to reset a user's password
func AdminResetUserPassword(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the new password
	hash := utils.HashPassword(input.NewPassword)

	// Update password in database
	_, err := config.DB.Exec("UPDATE users SET password=? WHERE id=?", hash, id)
	if err != nil {
		fmt.Println("Error resetting password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// AdminToggleUserStatus allows admin to activate/deactivate user accounts
func AdminToggleUserStatus(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Status string `json:"status" binding:"required"` // "active" or "inactive"
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status value
	if input.Status != "active" && input.Status != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'active' or 'inactive'"})
		return
	}

	_, err := config.DB.Exec("UPDATE users SET status=? WHERE id=?", input.Status, id)
	if err != nil {
		fmt.Println("Error updating user status:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("User status updated to %s", input.Status)})
}

// AdminGetUserDetails retrieves detailed information about a specific user
func AdminGetUserDetails(c *gin.Context) {
	id := c.Param("id")

	var user struct {
		ID            int       `json:"id"`
		Username      string    `json:"username"`
		FirstName     string    `json:"first_name"`
		MiddleName    string    `json:"middle_name"`
		Surname       string    `json:"surname"`
		Email         string    `json:"email"`
		ContactNumber string    `json:"contact_number"`
		Role          string    `json:"role"`
		Status        string    `json:"status"`
		CreatedAt     time.Time `json:"created_at"`
	}

	var firstName, middleName, surname, email, contactNumber sql.NullString

	err := config.DB.QueryRow(
		`SELECT id, username, first_name, middle_name, surname, email, 
		 contact_number, role, status, created_at FROM users WHERE id=?`,
		id,
	).Scan(&user.ID, &user.Username, &firstName, &middleName, &surname, &email,
		&contactNumber, &user.Role, &user.Status, &user.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert sql.NullString to regular strings
	user.FirstName = firstName.String
	user.MiddleName = middleName.String
	user.Surname = surname.String
	user.Email = email.String
	user.ContactNumber = contactNumber.String

	c.JSON(http.StatusOK, user)
}

// AdminGetSystemStats provides system statistics (useful for monitoring)
func AdminGetSystemStats(c *gin.Context) {
	var stats struct {
		TotalUsers       int `json:"total_users"`
		ActiveUsers      int `json:"active_users"`
		InactiveUsers    int `json:"inactive_users"`
		StudentCount     int `json:"student_count"`
		TeacherCount     int `json:"teacher_count"`
		AdminCount       int `json:"admin_count"`
		ITTechCount      int `json:"it_tech_count"`
		TotalStudents    int `json:"total_students"`
		ApprovedStudents int `json:"approved_students"`
		PendingStudents  int `json:"pending_students"`
		RejectedStudents int `json:"rejected_students"`
	}

	// Get total users
	config.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)

	// Get active users
	config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE status='active'").Scan(&stats.ActiveUsers)

	// Get inactive users
	config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE status='inactive'").Scan(&stats.InactiveUsers)

	// Get counts by role
	config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role='student'").Scan(&stats.StudentCount)
	config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role='teacher'").Scan(&stats.TeacherCount)
	config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role='admin'").Scan(&stats.AdminCount)
	config.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role='it_tech'").Scan(&stats.ITTechCount)

	// Get student table stats
	config.DB.QueryRow("SELECT COUNT(*) FROM students").Scan(&stats.TotalStudents)
	config.DB.QueryRow("SELECT COUNT(*) FROM students WHERE status='approved'").Scan(&stats.ApprovedStudents)
	config.DB.QueryRow("SELECT COUNT(*) FROM students WHERE status='pending'").Scan(&stats.PendingStudents)
	config.DB.QueryRow("SELECT COUNT(*) FROM students WHERE status='rejected'").Scan(&stats.RejectedStudents)

	c.JSON(http.StatusOK, stats)
}

// AdminBulkUpdateStatus allows admin to update multiple users' status at once
func AdminBulkUpdateStatus(c *gin.Context) {
	var input struct {
		UserIDs []int  `json:"user_ids" binding:"required"`
		Status  string `json:"status" binding:"required"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	if input.Status != "active" && input.Status != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'active' or 'inactive'"})
		return
	}

	// Begin transaction
	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update each user
	for _, userID := range input.UserIDs {
		_, err := tx.Exec("UPDATE users SET status=? WHERE id=?", input.Status, userID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Bulk status update successful",
		"updated_count": len(input.UserIDs),
	})
}

// AdminSearchUsers allows searching users by username, name, or email
func AdminSearchUsers(c *gin.Context) {
	searchTerm := c.Query("q")

	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search term is required"})
		return
	}

	rows, err := config.DB.Query(
		`SELECT id, username, first_name, middle_name, surname, email, 
		 contact_number, role, status FROM users 
		 WHERE username LIKE ? OR first_name LIKE ? OR surname LIKE ? OR email LIKE ?`,
		"%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var id int
		var username, role, status string
		var firstName, middleName, surname, email, contactNumber sql.NullString

		if err := rows.Scan(&id, &username, &firstName, &middleName, &surname,
			&email, &contactNumber, &role, &status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		users = append(users, gin.H{
			"id":             id,
			"username":       username,
			"first_name":     firstName.String,
			"middle_name":    middleName.String,
			"surname":        surname.String,
			"email":          email.String,
			"contact_number": contactNumber.String,
			"role":           role,
			"status":         status,
		})
	}

	c.JSON(http.StatusOK, users)
}

// ===== STUDENTS TABLE MANAGEMENT =====

// AdminGetStudents retrieves all students from the students table with optional filtering
func AdminGetStudents(c *gin.Context) {
	status := c.Query("status")

	query := `SELECT id, student_id, first_name, middle_name, last_name, age, 
	          contact_number, email, address, status, created_at, profile_picture 
	          FROM students WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status=?"
		args = append(args, status)
	}

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var students []gin.H
	for rows.Next() {
		var id int
		var studentID, status string
		var firstName, middleName, lastName, email, contactNumber, address, profilePicture sql.NullString
		var age sql.NullInt64
		var createdAt time.Time

		if err := rows.Scan(&id, &studentID, &firstName, &middleName, &lastName,
			&age, &contactNumber, &email, &address, &status, &createdAt, &profilePicture); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		students = append(students, gin.H{
			"id":              id,
			"student_id":      studentID,
			"first_name":      firstName.String,
			"middle_name":     middleName.String,
			"last_name":       lastName.String,
			"age":             age.Int64,
			"contact_number":  contactNumber.String,
			"email":           email.String,
			"address":         address.String,
			"status":          status,
			"created_at":      createdAt,
			"profile_picture": profilePicture.String,
		})
	}
	c.JSON(http.StatusOK, students)
}

// AdminGetStudentDetails retrieves detailed information about a specific student
func AdminGetStudentDetails(c *gin.Context) {
	id := c.Param("id")

	var student struct {
		ID             int       `json:"id"`
		StudentID      string    `json:"student_id"`
		FirstName      string    `json:"first_name"`
		MiddleName     string    `json:"middle_name"`
		LastName       string    `json:"last_name"`
		Age            int       `json:"age"`
		ContactNumber  string    `json:"contact_number"`
		Email          string    `json:"email"`
		Address        string    `json:"address"`
		Status         string    `json:"status"`
		CreatedAt      time.Time `json:"created_at"`
		ProfilePicture string    `json:"profile_picture"`
	}

	var firstName, middleName, lastName, email, contactNumber, address, profilePicture sql.NullString
	var age sql.NullInt64

	err := config.DB.QueryRow(
		`SELECT id, student_id, first_name, middle_name, last_name, age, 
		 contact_number, email, address, status, created_at, profile_picture 
		 FROM students WHERE id=?`,
		id,
	).Scan(&student.ID, &student.StudentID, &firstName, &middleName, &lastName,
		&age, &contactNumber, &email, &address, &student.Status, &student.CreatedAt, &profilePicture)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert sql.NullString to regular strings
	student.FirstName = firstName.String
	student.MiddleName = middleName.String
	student.LastName = lastName.String
	student.Email = email.String
	student.ContactNumber = contactNumber.String
	student.Address = address.String
	student.ProfilePicture = profilePicture.String
	student.Age = int(age.Int64)

	c.JSON(http.StatusOK, student)
}

// AdminToggleStudentStatus allows admin to approve/pending/reject student accounts
func AdminToggleStudentStatus(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Status string `json:"status" binding:"required"` // "approved", "pending", "rejected"
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status value
	if input.Status != "approved" && input.Status != "pending" && input.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'approved', 'pending', or 'rejected'"})
		return
	}

	_, err := config.DB.Exec("UPDATE students SET status=? WHERE id=?", input.Status, id)
	if err != nil {
		fmt.Println("Error updating student status:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Student status updated to %s", input.Status)})
}

// AdminResetStudentPassword allows admin to reset a student's password
func AdminResetStudentPassword(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the new password
	hash := utils.HashPassword(input.NewPassword)

	// Update password in database
	_, err := config.DB.Exec("UPDATE students SET password=? WHERE id=?", hash, id)
	if err != nil {
		fmt.Println("Error resetting student password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Student password reset successfully"})
}

// AdminSearchStudents allows searching students by name, student_id, or email
func AdminSearchStudents(c *gin.Context) {
	searchTerm := c.Query("q")

	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search term is required"})
		return
	}

	rows, err := config.DB.Query(
		`SELECT id, student_id, first_name, middle_name, last_name, age, 
		 contact_number, email, address, status, profile_picture FROM students 
		 WHERE student_id LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR email LIKE ?`,
		"%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var students []gin.H
	for rows.Next() {
		var id int
		var studentID, status string
		var firstName, middleName, lastName, email, contactNumber, address, profilePicture sql.NullString
		var age sql.NullInt64

		if err := rows.Scan(&id, &studentID, &firstName, &middleName, &lastName,
			&age, &contactNumber, &email, &address, &status, &profilePicture); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		students = append(students, gin.H{
			"id":              id,
			"student_id":      studentID,
			"first_name":      firstName.String,
			"middle_name":     middleName.String,
			"last_name":       lastName.String,
			"age":             age.Int64,
			"contact_number":  contactNumber.String,
			"email":           email.String,
			"address":         address.String,
			"status":          status,
			"profile_picture": profilePicture.String,
		})
	}

	c.JSON(http.StatusOK, students)
}

// AdminDeleteStudent allows the admin to delete a student
func AdminDeleteStudent(c *gin.Context) {
	id := c.Param("id")
	_, err := config.DB.Exec("DELETE FROM students WHERE id=?", id)
	if err != nil {
		fmt.Println("Error while deleting student:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted"})
}

// AdminBulkUpdateStudentStatus allows admin to update multiple students' status at once
func AdminBulkUpdateStudentStatus(c *gin.Context) {
	var input struct {
		StudentIDs []int  `json:"student_ids" binding:"required"`
		Status     string `json:"status" binding:"required"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	if input.Status != "approved" && input.Status != "pending" && input.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'approved', 'pending', or 'rejected'"})
		return
	}

	// Begin transaction
	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update each student
	for _, studentID := range input.StudentIDs {
		_, err := tx.Exec("UPDATE students SET status=? WHERE id=?", input.Status, studentID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Bulk student status update successful",
		"updated_count": len(input.StudentIDs),
	})
}
