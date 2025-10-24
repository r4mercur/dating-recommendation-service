package search

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var pgClient *sql.DB

func InitPostgresClient() {
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDatabase := os.Getenv("POSTGRES_DATABASE")

	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"localhost",
		5432,
		postgresUser,
		postgresPassword,
		postgresDatabase)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Error opening connection to postgres: %s", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging postgres: %s", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	pgClient = db
	log.Println("Connected to postgres")
}

func GetPostgresClient() *sql.DB {
	if pgClient == nil {
		InitPostgresClient()
	}
	return pgClient
}

func ClosePostgresClient() {
	if pgClient != nil {
		if err := pgClient.Close(); err != nil {
			log.Printf("Error closing postgres client: %s", err)
		}
	}
}
