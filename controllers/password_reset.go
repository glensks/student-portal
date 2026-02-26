package controllers

import (
	"crypto/rand"
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

// fpAllow returns (allowed, retryAfterSeconds)
func fpAllow(ip string) (bool, int) {
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

	// Prune old attempts
	cutoff := now.Add(-fpWindow)
	fresh := e.attempts[:0]
	for _, t := range e.attempts {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	e.attempts = fresh

	// Record this attempt
	e.attempts = append(e.attempts, now)

	// Block if limit exceeded
	if len(e.attempts) > fpMax {
		e.blockedUntil = now.Add(fpBlockFor)
		return false, int(fpBlockFor.Seconds())
	}

	return true, 0
}

/*
============================================================

	POST /forgot-password
	============================================================
*/
func ForgotPassword(c *gin.Context) {
	ip := fpClientIP(c)

	// --- Rate limit check ---
	allowed, retryAfter := fpAllow(ip)
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":               "Too many requests. Please wait before trying again.",
			"retry_after_seconds": retryAfter,
		})
		return
	}

	// --- Parse body ---
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required."})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	// --- Format validation ---
	if !fpValidEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid email address."})
		return
	}

	// --- ALWAYS return 200 (prevents email enumeration) ---
	// Actual DB check + email send happens in background
	go fpSendIfExists(email)

	c.JSON(http.StatusOK, gin.H{
		"message": "If that email is registered, you will receive a reset link shortly.",
	})
}

// fpSendIfExists only sends an email when the address is actually in the DB.
// Running in a goroutine so the HTTP response is instant.
func fpSendIfExists(email string) {
	var studentDBID int
	var firstName string

	err := config.DB.QueryRow(`
		SELECT id, first_name FROM students WHERE LOWER(email) = ? LIMIT 1
	`, email).Scan(&studentDBID, &firstName)

	if err != nil {
		// Email not registered — silently do nothing (no log leak either)
		fmt.Printf("[ForgotPassword] email not found: %s — no email sent\n", email)
		return
	}

	// Invalidate existing unused tokens for this student
	_, _ = config.DB.Exec(`
		UPDATE password_reset_tokens SET used = 1
		WHERE student_id = ? AND used = 0
	`, studentDBID)

	// Generate secure token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Printf("[ForgotPassword] token gen error: %v\n", err)
		return
	}
	token := hex.EncodeToString(b)

	// Save token — expiry handled by MySQL
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
	// MySQL compares time — no Go timezone issues
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

// Basic email format check
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
