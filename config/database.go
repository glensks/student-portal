package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {
	host := os.Getenv("MYSQLHOST")
	port := os.Getenv("MYSQLPORT")
	user := os.Getenv("MYSQLUSER")
	password := os.Getenv("MYSQLPASSWORD")
	dbname := os.Getenv("MYSQLDATABASE")

	// Debug: tingnan kung may laman ang env vars
	log.Printf("DEBUG - Host: '%s', Port: '%s', User: '%s', DB: '%s'", host, port, user, dbname)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("sql.Open error: ", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Ping error: ", err) // <-- makikita natin ang exact error dito
	}

	log.Println("✅ MySQL Connected")
}
