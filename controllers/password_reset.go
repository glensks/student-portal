package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"student-portal/config"
	"student-portal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

/* ============================================================
   IP-BASED RATE LIMITER
   Max 3 requests per IP per 15 minutes.
   On exceed → blocked for 15 minutes.

   FIX: fpAllow is now called AFTER the email existence check,
   so a 404 (unregistered email / typo) does NOT consume an
   attempt. Only confirmed, valid submissions count.
   ============================================================ */

type fpEntry struct {
	attempts     []time.Time
	blockedUntil time.Time
}

var (
	fpLimiter  = make(map[string]*fpEntry)
	fpMu       sync.Mutex
	fpMax      = 3
	fpWindow   = 15 * time.Minute
	fpBlockFor = 15 * time.Minute
)

func fpClientIP(c *gin.Context) string {
	if fwd := c.GetHeader("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return c.ClientIP()
}

// fpAllow returns (allowed, retryAfterSeconds).
// It also performs a read-only check without recording when recordAttempt=false.
func fpAllow(ip string, recordAttempt bool) (bool, int) {
	fpMu.Lock()
	defer fpMu.Unlock()

	now := time.Now()
	e, ok := fpLimiter[ip]
	if !ok {
		e = &fpEntry{}
		fpLimiter[ip] = e
	}

	// Still in block period?
	if !e.blockedUntil.IsZero() && now.Before(e.blockedUntil) {
		secs := int(e.blockedUntil.Sub(now).Seconds()) + 1
		return false, secs
	}

	// Clear expired block
	if !e.blockedUntil.IsZero() {
		e.blockedUntil = time.Time{}
		e.attempts = nil
	}

	// Prune old attempts outside the window
	cutoff := now.Add(-fpWindow)
	fresh := e.attempts[:0]
	for _, t := range e.attempts {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	e.attempts = fresh

	// Read-only check — used at the top of the handler so a blocked IP
	// gets a 429 immediately without wasting a DB query.
	if !recordAttempt {
		if len(e.attempts) >= fpMax {
			e.blockedUntil = now.Add(fpBlockFor)
			return false, int(fpBlockFor.Seconds())
		}
		return true, 0
	}

	// Record this attempt (only called after email is confirmed to exist)
	e.attempts = append(e.attempts, now)

	if len(e.attempts) > fpMax {
		e.blockedUntil = now.Add(fpBlockFor)
		return false, int(fpBlockFor.Seconds())
	}

	return true, 0
}

/*
============================================================

	POST /forgot-password

	Order of operations (important for correct rate-limit UX):
	1. Validate email format           — no DB, no rate-limit cost
	2. Check IP is not already blocked — read-only, no cost
	3. Check email exists in DB        — 404 costs nothing
	4. Record rate-limit attempt       — only on confirmed email
	5. Send reset email async
	============================================================
*/
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 1. Basic format check — cheapest gate, no side effects
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !fpValidEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		return
	}

	ip := fpClientIP(c)

	// 2. Read-only block check — reject already-blocked IPs before DB hit
	if allowed, retryAfter := fpAllow(ip, false); !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":               "too many attempts, please try again later",
			"retry_after_seconds": retryAfter,
		})
		return
	}

	// 3. Check if email exists in DB — 404 does NOT count as an attempt
	var studentID int
	var firstName string
	err := config.DB.QueryRow(`
		SELECT id, first_name FROM students
		WHERE LOWER(email) = ? AND status = 'approved'
		LIMIT 1
	`, email).Scan(&studentID, &firstName)

	if err == sql.ErrNoRows {
		// Email not registered or not approved — visible error, no rate-limit cost
		c.JSON(http.StatusNotFound, gin.H{"error": "email_not_found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	// 4. Email confirmed — NOW record the rate-limit attempt
	if allowed, retryAfter := fpAllow(ip, true); !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":               "too many attempts, please try again later",
			"retry_after_seconds": retryAfter,
		})
		return
	}

	// 5. Generate token and send email asynchronously
	go fpSendIfExists(email)

	c.JSON(http.StatusOK, gin.H{"message": "reset link sent"})
}

/*
============================================================

	fpSendIfExists
	Runs in a goroutine — generates a secure token, saves it,
	and sends the reset email. Fails silently on error so the
	HTTP response is always instant.

	FIX: Added AND status = 'approved' — must match the main
	handler query exactly so a non-approved student can never
	receive a reset email even if the goroutine races ahead.
	============================================================
*/
func fpSendIfExists(email string) {
	var studentDBID int
	var firstName string

	// FIX: was missing AND status = 'approved' — caused emails to be
	// sent for pending/unapproved students and non-existent addresses.
	err := config.DB.QueryRow(`
		SELECT id, first_name FROM students
		WHERE LOWER(email) = ? AND status = 'approved'
		LIMIT 1
	`, email).Scan(&studentDBID, &firstName)

	if err != nil {
		fmt.Printf("[ForgotPassword] email not found or not approved in goroutine: %s\n", email)
		return
	}

	// Invalidate any existing unused tokens for this student
	_, _ = config.DB.Exec(`
		UPDATE password_reset_tokens SET used = 1
		WHERE student_id = ? AND used = 0
	`, studentDBID)

	// Generate a cryptographically secure 32-byte token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Printf("[ForgotPassword] token gen error: %v\n", err)
		return
	}
	token := hex.EncodeToString(b)

	// Persist token with 1-hour expiry
	_, err = config.DB.Exec(`
		INSERT INTO password_reset_tokens (student_id, token, expires_at)
		VALUES (?, ?, NOW() + INTERVAL 1 HOUR)
	`, studentDBID, token)
	if err != nil {
		fmt.Printf("[ForgotPassword] DB insert error: %v\n", err)
		return
	}

	resetURL := fmt.Sprintf("http://localhost:8080/reset-password?token=%s", token)
	body := buildResetEmailBody(firstName, resetURL)

	if err := utils.SendEmail(email, "Password Reset Request - Student Portal", body); err != nil {
		fmt.Printf("[ForgotPassword] email send error to %s: %v\n", email, err)
	} else {
		fmt.Printf("[ForgotPassword] reset email sent to %s (student_id=%d)\n", email, studentDBID)
	}
}

/*
============================================================

	GET /verify-reset-token?token=xxx
	============================================================
*/
func VerifyResetToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false})
		return
	}

	var id int
	err := config.DB.QueryRow(`
		SELECT id FROM password_reset_tokens
		WHERE token = ? AND expires_at > NOW() AND used = 0
		LIMIT 1
	`, token).Scan(&id)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true})
}

/*
============================================================

	POST /reset-password
	============================================================
*/
func ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token"        binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and a new password (min 8 characters) are required."})
		return
	}

	var studentDBID, used int
	err := config.DB.QueryRow(`
		SELECT student_id, used FROM password_reset_tokens
		WHERE token = ? AND expires_at > NOW() AND used = 0
		LIMIT 1
	`, req.Token).Scan(&studentDBID, &used)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This reset link is invalid or has expired. Please request a new one."})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password."})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error."})
		return
	}

	if _, err = tx.Exec(`UPDATE students SET password = ? WHERE id = ?`, hashed, studentDBID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password."})
		return
	}

	if _, err = tx.Exec(`UPDATE password_reset_tokens SET used = 1 WHERE token = ?`, req.Token); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate token."})
		return
	}

	tx.Commit()
	fmt.Printf("[ResetPassword] success for student_id=%d\n", studentDBID)
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. You can now log in with your new password."})
}

/* ============================================================
   HELPERS
   ============================================================ */

// fpValidEmail does a basic sanity check before hitting the DB.
func fpValidEmail(email string) bool {
	if len(email) < 5 || len(email) > 254 {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	return len(local) > 0 && strings.Contains(domain, ".") && len(domain) > 3
}

func buildResetEmailBody(firstName, resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;background:#f4f4f4;padding:20px;margin:0;">
  <div style="max-width:480px;margin:0 auto;background:white;border-radius:12px;overflow:hidden;box-shadow:0 4px 16px rgba(0,0,0,0.1);">
    <div style="background:linear-gradient(135deg,#1b4332,#40916c);padding:32px;text-align:center;">
      <div style="font-size:40px;margin-bottom:8px;">🎓</div>
      <h1 style="color:white;margin:0;font-size:22px;font-weight:700;">Student Portal</h1>
      <p style="color:rgba(255,255,255,0.8);margin:6px 0 0;font-size:14px;">Password Reset Request</p>
    </div>
    <div style="padding:36px 32px;">
      <p style="font-size:16px;color:#1a1a1a;margin:0 0 12px;">Hi <strong>%s</strong>,</p>
      <p style="color:#555;line-height:1.7;margin:0 0 24px;">
        We received a request to reset your Student Portal password.
        Click the button below to set a new password.
        This link will expire in <strong>1 hour</strong>.
      </p>
      <div style="text-align:center;margin:28px 0;">
        <a href="%s" style="background:linear-gradient(135deg,#40916c,#52b788);color:white;text-decoration:none;padding:14px 36px;border-radius:8px;font-weight:700;font-size:15px;display:inline-block;box-shadow:0 4px 12px rgba(64,145,108,0.35);">
          Reset My Password
        </a>
      </div>
      <p style="color:#888;font-size:13px;margin:0 0 6px;">Button not working? Copy and paste this link:</p>
      <p style="margin:0 0 24px;"><a href="%s" style="color:#40916c;font-size:13px;word-break:break-all;">%s</a></p>
      <hr style="border:none;border-top:1px solid #eee;margin:24px 0;">
      <p style="color:#999;font-size:12px;margin:0;line-height:1.6;">
        If you didn't request this, you can safely ignore this email.
        Your password will not be changed.
      </p>
    </div>
    <div style="background:#f8f8f8;padding:16px 32px;text-align:center;border-top:1px solid #eee;">
      <p style="color:#aaa;font-size:12px;margin:0;">&copy; 2024 Student Information System. All rights reserved.</p>
    </div>
  </div>
</body>
</html>`, firstName, resetURL, resetURL, resetURL)
}
