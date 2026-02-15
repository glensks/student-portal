package utils

import (
	"fmt"
	"net/smtp"
)

// SendEmail sends an email using Gmail SMTP
func SendEmail(to, subject, body string) error {
	from := "glensssg@gmail.com"
	password := "ueaejehhkllpvlgv" // app password

	// Gmail SMTP server configuration
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// Create message
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-version: 1.0;\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n"+
			"%s",
		from, to, subject, body,
	))

	// Authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Send email
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
}
