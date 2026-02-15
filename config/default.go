package config

import (
	"fmt"
	"student-portal/utils"
)

func CreateDefaultUsers() {
	users := []struct {
		username string
		password string
		role     string
	}{
		{"admin", "admin123", "admin"},
		{"student1", "student123", "student"},
		{"teacher1", "teacher123", "teacher"},
		{"registrar1", "registrar123", "registrar"},
		{"cashier1", "cashier123", "cashier"},
		{"records1", "records123", "records"},
		{"parent1", "parent123", "parent"},
	}

	for _, u := range users {
		var count int
		err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", u.username).Scan(&count)
		if err != nil {
			panic(err)
		}

		if count == 0 {
			hash := utils.HashPassword(u.password)
			_, err := DB.Exec("INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
				u.username, hash, u.role)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Default account created: %s / %s\n", u.username, u.password)
		}
	}
}
