package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
)

// SendEmail sends an HTML email using Gmail SMTP port 465 (SSL/TLS)
// Railway blocks port 587, so we use 465 with direct TLS instead
func SendEmail(to, subject, body string) error {
	from := "glensssg@gmail.com"
	password := "ueaejehhkllpvlgv" // Gmail App Password (no spaces)

	smtpHost := "smtp.gmail.com"
	smtpPort := "465"

	if to == "" {
		log.Println("❌ SendEmail: recipient email is empty")
		return fmt.Errorf("recipient email is empty")
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

	// Port 465 = immediate TLS (no STARTTLS)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpHost,
	}

	conn, err := tls.Dial("tcp", net.JoinHostPort(smtpHost, smtpPort), tlsConfig)
	if err != nil {
		log.Printf("❌ TLS dial failed: %v", err)
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Printf("❌ SMTP client failed: %v", err)
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", from, password, smtpHost)
	if err = client.Auth(auth); err != nil {
		log.Printf("❌ Auth failed: %v", err)
		return fmt.Errorf("SMTP auth failed: %w", err)
	}

	if err = client.Mail(from); err != nil {
		log.Printf("❌ MAIL FROM failed: %v", err)
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	if err = client.Rcpt(to); err != nil {
		log.Printf("❌ RCPT TO failed: %v", err)
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		log.Printf("❌ DATA failed: %v", err)
		return fmt.Errorf("DATA failed: %w", err)
	}

	if _, err = w.Write(msg); err != nil {
		log.Printf("❌ Write failed: %v", err)
		return fmt.Errorf("write failed: %w", err)
	}

	if err = w.Close(); err != nil {
		log.Printf("❌ Close writer failed: %v", err)
		return fmt.Errorf("close writer failed: %w", err)
	}

	client.Quit()
	log.Printf("✅ Email sent successfully to: %s", to)
	return nil
}
