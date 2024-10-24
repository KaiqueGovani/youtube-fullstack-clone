package main

import (
	"context"
	"database/sql"
	"fmt"
	"fullstack-youtube-clone/internal/converter"
	"fullstack-youtube-clone/internal/rabbitmq"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/lmittmann/tint"
	"github.com/streadway/amqp"
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
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	db, err := connectPostgres()
	if err != nil {
		panic(err)
	}

	rabbitMQURL := getEnvOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	rabbitClient, err := rabbitmq.NewRabbitClient(rabbitMQURL)
	if err != nil {
		panic(err)
	}
	defer rabbitClient.Close()

	convertionExch := getEnvOrDefault("CONVERSION_EXCHANGE", "conversion_exchange")
	queueName := getEnvOrDefault("CONVERSION_QUEUE", "video_conversion_queue")
	conversionKey := getEnvOrDefault("CONVERSION_KEY", "conversion")
	confirmationKey := getEnvOrDefault("CONFIRMATION_KEY", "finish-conversion")
	confirmationQueue := getEnvOrDefault("CONFIRMATION_QUEUE", "finish_confirmation_queue")

	vc := converter.NewVideoConverter(rabbitClient, db)

	// vc.Handle([]byte(`{"video_id": 1, "path": "/media/uploads/1"}`))
	msgs, err := rabbitClient.ConsumeMessages(convertionExch, conversionKey, queueName)
	if err != nil {
		slog.Error("Error consuming messages", slog.String("error", err.Error()))
	}

	go func() {
		for d := range msgs {
			go func(d amqp.Delivery) {
				vc.Handle(d, convertionExch, confirmationKey, confirmationQueue)
			}(d)
		}
	}()

	<-sigChan
	cancel()
	slog.Info("Shutting down...")
}
