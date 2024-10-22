package main

import (
	"database/sql"
	"fmt"
	"fullstack-youtube-clone/internal/converter"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
)

func connectPostgres() (*sql.DB, error) {
	user := getEnvOrDefault("POSTGRES_USER", "postgres")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
	host := getEnvOrDefault("POSTGRES_HOST", "localhost")
	dbName := getEnvOrDefault("POSTGRES_DB", "postgres")
	sslmode := getEnvOrDefault("POSTGRES_SSL_MODE", "disable")

	connStr := fmt.Sprintf("user=%s password=%s host=%s dbname=%s sslmode=%s", user, password, host, dbName, sslmode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("Error connecting to postgres", slog.String("connStr", connStr))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		slog.Error("Error pinging postgres", slog.String("connStr", connStr))
		return nil, err
	}

	slog.Info("Connected to postgres", slog.String("db_name", dbName))
	return db, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	db, err := connectPostgres()
	if err != nil {
		panic(err)
	}

	vc := converter.NewVideoConverter(db)

	vc.Handle([]byte(`{"video_id": 1, "path": "/media/uploads/1"}`))
}
