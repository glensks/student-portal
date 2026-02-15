package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(p string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(p), 14)
	return string(h)
}

func CheckPassword(h, p string) bool {
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(p)) == nil
}
