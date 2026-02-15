package controllers

import (
	"net/http"
	"student-portal/config"

	"github.com/gin-gonic/gin"
)

// ===================== CASHIER GET STUDENT PAYMENTS =====================
// Search payments by student ID
// ===================== CASHIER VIEW PENDING PAYMENTS =====================

func CashierGetPendingPayments(c *gin.Context) {

	studentID := c.Query("student_id")
	if studentID == "" {
		c.JSON(400, gin.H{"error": "student_id required"})
		return
	}

	// 1️⃣ get student db id
	var studentDBID int
	err := config.DB.QueryRow(`
        SELECT id FROM students WHERE student_id = ?
    `, studentID).Scan(&studentDBID)
	if err != nil {
		c.JSON(404, gin.H{"error": "student not found"})
		return
	}

	// 2️⃣ get subjects (⭐ DITO MO IDADAGDAG)
	var subjectIDs string
	_ = config.DB.QueryRow(`
        SELECT IFNULL(subjects,'')
        FROM student_academic
        WHERE student_id = ?
    `, studentDBID).Scan(&subjectIDs)

	var subjects []string
	if subjectIDs != "" {
		rows, _ := config.DB.Query(`
            SELECT subject_name
            FROM subjects
            WHERE FIND_IN_SET(id, ?)
        `, subjectIDs)
		defer rows.Close()

		for rows.Next() {
			var name string
			rows.Scan(&name)
			subjects = append(subjects, name)
		}
	}

	// 3️⃣ get pending payments
	rows, err := config.DB.Query(`
        SELECT id, total_amount, amount_paid, payment_method, status
        FROM student_payments
        WHERE student_id = ? AND status = 'pending'
    `, studentDBID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch pending payments"})
		return
	}
	defer rows.Close()

	var pending []gin.H
	for rows.Next() {
		var id, total, paid int
		var method, status string

		rows.Scan(&id, &total, &paid, &method, &status)

		pending = append(pending, gin.H{
			"payment_id":     id,
			"total_amount":   total,
			"amount_paid":    paid,
			"remaining":      total - paid,
			"payment_method": method,
			"status":         status,
			"subjects":       subjects, // ⭐ HERE
		})
	}

	c.JSON(200, gin.H{
		"pending": pending,
	})
}

// ===================== CASHIER APPROVE PAYMENT =====================

func CashierApprovePayment(c *gin.Context) {
	var req struct {
		PaymentID int `json:"payment_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	db := config.DB

	var studentID, totalAmount, amountPaid int
	err := db.QueryRow(`
		SELECT student_id, total_amount, amount_paid
		FROM student_payments
		WHERE id = ? AND status = 'pending'
	`, req.PaymentID).Scan(&studentID, &totalAmount, &amountPaid)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pending payment not found"})
		return
	}

	// ✅ FULL PAYMENT
	if amountPaid >= totalAmount {
		_, err := db.Exec(`
			UPDATE student_payments
			SET status = 'paid'
			WHERE id = ?
		`, req.PaymentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve payment"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "payment fully approved",
			"status":  "paid",
		})
		return
	}

	// ✅ DOWNPAYMENT
	remaining := float64(totalAmount - amountPaid)

	_, err = db.Exec(`
		UPDATE student_payments
		SET status = 'partial'
		WHERE id = ?
	`, req.PaymentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update payment"})
		return
	}

	err = CreateInstallmentsAfterDownpayment(req.PaymentID, remaining)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create installments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "downpayment approved & installments created",
		"status":  "partial",
	})
}

func CreateInstallmentsAfterDownpayment(paymentID int, remaining float64) error {
	db := config.DB

	terms := []string{"prelim", "midterm", "finals"}
	baseAmount := remaining / 3

	var totalInserted float64 = 0

	for i, term := range terms {
		amount := baseAmount

		// Adjust last term to fix floating point issue
		if i == len(terms)-1 {
			amount = remaining - totalInserted
		}

		_, err := db.Exec(`
			INSERT INTO student_installments
			(payment_id, term, amount)
			VALUES (?, ?, ?)
		`, paymentID, term, amount)

		if err != nil {
			return err
		}

		totalInserted += amount
	}

	return nil
}
