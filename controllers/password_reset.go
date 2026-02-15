package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"student-portal/config"
	"student-portal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ========================== FORGOT PASSWORD ==========================
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid email address."})
		return
	}

	var studentDBID int
	var firstName string
	err := config.DB.QueryRow(`
		SELECT id, first_name FROM students WHERE email = ? LIMIT 1
	`, req.Email).Scan(&studentDBID, &firstName)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, you will receive a reset link shortly."})
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token."})
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Invalidate old tokens
	_, _ = config.DB.Exec(
		`UPDATE password_reset_tokens SET used = 1 WHERE student_id = ? AND used = 0`,
		studentDBID,
	)

	// ✅ Use MySQL NOW() + INTERVAL so timezone is handled entirely by MySQL
	_, err = config.DB.Exec(`
		INSERT INTO password_reset_tokens (student_id, token, expires_at)
		VALUES (?, ?, NOW() + INTERVAL 1 HOUR)
	`, studentDBID, token)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reset token."})
		return
	}

	resetURL := fmt.Sprintf("http://localhost:8080/reset-password?token=%s", token)
	subject := "Password Reset Request - Student Portal"
	body := buildResetEmailBody(firstName, resetURL)

	go func() {
		if err := utils.SendEmail(req.Email, subject, body); err != nil {
			fmt.Printf("[EMAIL ERROR] Failed to send reset email to %s: %v\n", req.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, you will receive a reset link shortly."})
}

// ========================== RESET PASSWORD ==========================
func ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and a new password (min 8 characters) are required."})
		return
	}

	var studentDBID int
	var used int

	// ✅ Let MySQL compare times — no Go timezone confusion at all
	err := config.DB.QueryRow(`
		SELECT student_id, used
		FROM password_reset_tokens
		WHERE token = ?
		  AND expires_at > NOW()
		  AND used = 0
		LIMIT 1
	`, req.Token).Scan(&studentDBID, &used)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This reset link is invalid or has expired. Please request a new one."})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password."})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error."})
		return
	}

	_, err = tx.Exec(`UPDATE students SET password = ? WHERE id = ?`, hashedPassword, studentDBID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password."})
		return
	}

	_, err = tx.Exec(`UPDATE password_reset_tokens SET used = 1 WHERE token = ?`, req.Token)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate token."})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. You can now log in with your new password."})
}

// ========================== VERIFY TOKEN ==========================
func VerifyResetToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false})
		return
	}

	var id int
	// ✅ MySQL handles the time comparison — no Go timezone issues
	err := config.DB.QueryRow(`
		SELECT id FROM password_reset_tokens
		WHERE token = ?
		  AND expires_at > NOW()
		  AND used = 0
		LIMIT 1
	`, token).Scan(&id)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// ========================== EMAIL TEMPLATE ==========================
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
      </p>
    </div>
    <div style="background:#f8f8f8;padding:16px 32px;text-align:center;border-top:1px solid #eee;">
      <p style="color:#aaa;font-size:12px;margin:0;">&copy; 2024 Student Information System. All rights reserved.</p>
    </div>
  </div>
</body>
</html>`, firstName, resetURL, resetURL, resetURL)
}
