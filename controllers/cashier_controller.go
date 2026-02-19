package controllers

import (
	"fmt"
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

	// 2️⃣ get subjects
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
			"subjects":       subjects,
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

	// ✅ PARTIAL PAYMENT — update status to partial
	_, err = db.Exec(`
		UPDATE student_payments
		SET status = 'partial'
		WHERE id = ?
	`, req.PaymentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update payment"})
		return
	}

	remaining := float64(totalAmount - amountPaid)

	// ✅ Check kung may existing installments na bago mag-create
	// (ibig sabihin, nagbayad na ng isang term ang student bago i-approve ng cashier)
	var existingInstallmentCount int
	db.QueryRow(`
		SELECT COUNT(*) FROM student_installments WHERE payment_id = ?
	`, req.PaymentID).Scan(&existingInstallmentCount)

	hadExistingInstallments := existingInstallmentCount > 0

	// ✅ Create installment rows para sa missing terms
	err = CreateInstallmentsAfterDownpayment(req.PaymentID, remaining)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create installments"})
		return
	}

	// ✅ ONLY mark installments as paid kung may existing installment payment na ang student
	// BAGO pa i-approve ng cashier ang downpayment.
	//
	// BAKIT? Kung bagong downpayment lang (walang installment payments pa),
	// ang amountPaid ay ang downpayment amount — hindi dapat mag-trigger ng
	// "prelim paid" kahit malaki ang downpayment kumpara sa per-term amount.
	if hadExistingInstallments {
		err = MarkPaidInstallments(req.PaymentID, float64(amountPaid))
		if err != nil {
			fmt.Println("⚠️ Warning: could not mark paid installments:", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "payment approved & installments updated",
		"status":  "partial",
	})
}

// CreateInstallmentsAfterDownpayment creates installment rows ONLY for terms
// that do not already exist in student_installments for this payment.
// This prevents duplicate entries when a student has already paid a term
// (e.g. Prelim) before the cashier approves the downpayment.
func CreateInstallmentsAfterDownpayment(paymentID int, remaining float64) error {
	db := config.DB

	allTerms := []string{"prelim", "midterm", "finals"}

	// 1️⃣ Check which terms already exist for this payment
	existingRows, err := db.Query(`
		SELECT term FROM student_installments
		WHERE payment_id = ?
	`, paymentID)
	if err != nil {
		return err
	}
	defer existingRows.Close()

	existingTerms := make(map[string]bool)
	for existingRows.Next() {
		var term string
		existingRows.Scan(&term)
		existingTerms[term] = true
	}

	// 2️⃣ Filter to only terms that are missing
	var missingTerms []string
	for _, t := range allTerms {
		if !existingTerms[t] {
			missingTerms = append(missingTerms, t)
		}
	}

	// Nothing to create — all terms already recorded
	if len(missingTerms) == 0 {
		return nil
	}

	// 3️⃣ Split remaining balance equally across missing terms only
	baseAmount := remaining / float64(len(missingTerms))
	var totalInserted float64

	for i, term := range missingTerms {
		amount := baseAmount

		// Last term absorbs any floating-point rounding difference
		if i == len(missingTerms)-1 {
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

func MarkPaidInstallments(paymentID int, amountPaid float64) error {
	db := config.DB

	// ✅ I-fetch ang downpayment amount para ma-exclude sa increment calculation
	// Kung wala kang downpayment_amount column, gamitin ang amount ng unang
	// payment na ginawa (amountPaid bago pa mag-installment).
	// Pinakasimple: i-store ang downpayment_amount sa student_payments table.
	var downpaymentAmount float64
	err := db.QueryRow(`
		SELECT downpayment_amount FROM student_payments WHERE id = ?
	`, paymentID).Scan(&downpaymentAmount)
	if err != nil {
		// Kung walang column, fallback sa 0 (pero dapat idagdag ang column)
		downpaymentAmount = 0
		fmt.Println("⚠️ Warning: could not fetch downpayment_amount:", err)
	}

	// Fetch all installments in term order
	rows, err := db.Query(`
		SELECT id, term, amount, status
		FROM student_installments
		WHERE payment_id = ?
		ORDER BY FIELD(term, 'prelim', 'midterm', 'finals')
	`, paymentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type installment struct {
		id     int
		term   string
		amount float64
		status string
	}

	var installments []installment
	for rows.Next() {
		var inst installment
		if err := rows.Scan(&inst.id, &inst.term, &inst.amount, &inst.status); err != nil {
			return err
		}
		installments = append(installments, inst)
	}

	if len(installments) == 0 {
		return nil
	}

	// Sum ng installments na paid na
	var alreadyPaidTotal float64
	for _, inst := range installments {
		if inst.status == "paid" {
			alreadyPaidTotal += inst.amount
		}
	}

	// ✅ FIXED: I-strip ang downpayment bago kalkulahin ang increment
	// increment = (total paid) - (downpayment) - (already paid installments)
	increment := amountPaid - downpaymentAmount - alreadyPaidTotal

	if increment <= 0 {
		fmt.Println("ℹ️ MarkPaidInstallments: no new increment to process")
		return nil
	}

	fmt.Printf("💰 MarkPaidInstallments: amountPaid=%.2f, downpayment=%.2f, alreadyPaid=%.2f, increment=%.2f\n",
		amountPaid, downpaymentAmount, alreadyPaidTotal, increment)

	const tolerance = 1.0

	// Exact match first
	for _, inst := range installments {
		if inst.status == "paid" {
			continue
		}
		diff := increment - inst.amount
		if diff >= -tolerance && diff <= tolerance {
			_, err := db.Exec(`
				UPDATE student_installments
				SET status = 'paid', paid_at = NOW()
				WHERE id = ?
			`, inst.id)
			if err != nil {
				return fmt.Errorf("failed to mark %s as paid: %w", inst.term, err)
			}
			fmt.Printf("✅ Marked %s (id=%d, amount=%.2f) as paid\n", inst.term, inst.id, inst.amount)
			return nil
		}
	}

	// Fallback
	for _, inst := range installments {
		if inst.status == "paid" {
			continue
		}
		if increment >= inst.amount-tolerance {
			_, err := db.Exec(`
				UPDATE student_installments
				SET status = 'paid', paid_at = NOW()
				WHERE id = ?
			`, inst.id)
			if err != nil {
				return fmt.Errorf("failed to mark %s as paid (fallback): %w", inst.term, err)
			}
			fmt.Printf("✅ Marked %s (id=%d) as paid via fallback\n", inst.term, inst.id)
			return nil
		}
	}

	fmt.Printf("ℹ️ MarkPaidInstallments: no installment matched increment ₱%.2f\n", increment)
	return nil
}
