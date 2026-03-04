package utils

import (
	"fmt"
	"log"
	"net/smtp"
)

// SendEmail sends an HTML email using Gmail SMTP
func SendEmail(to, subject, body string) error {
	from := "glensssg@gmail.com"
	password := "ueaejehhkllpvlgv" // Gmail App Password

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// Validate inputs
	if to == "" {
		log.Println("❌ SendEmail: recipient email is empty")
		return fmt.Errorf("recipient email is empty")
	}
	if subject == "" {
		log.Println("❌ SendEmail: subject is empty")
		return fmt.Errorf("subject is empty")
	}

	log.Printf("📧 Sending email to: %s | Subject: %s", to, subject)

	msg := []byte(fmt.Sprintf(
		"From: University of Manila <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"+
			"%s",
		from, to, subject, body,
	))

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("❌ SendEmail failed to %s: %v", to, err)
		return err
	}

	log.Printf("✅ Email sent successfully to: %s", to)
	return nil
}
