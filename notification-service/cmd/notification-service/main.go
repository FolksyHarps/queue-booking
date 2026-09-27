package main

import (
	"log"
	"os"
	"time"

	"github.com/queue-booking/notification-service/internal/client"
	"github.com/queue-booking/notification-service/internal/handler"
	"github.com/queue-booking/notification-service/internal/model"
	"github.com/queue-booking/notification-service/internal/repository"
	"github.com/queue-booking/notification-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := mustEnv("DB_DSN")
	authURL := mustEnv("AUTH_SERVICE_URL")
	bookingURL := mustEnv("BOOKING_SERVICE_URL")
	internalKey := mustEnv("INTERNAL_API_KEY")
	reportDir := envOr("REPORT_DIR", "/data/reports")
	port := envOr("PORT", "8084")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		// TIMESTAMP columns have no time zone: always store UTC so reads are unambiguous.
		NowFunc: func() time.Time { return time.Now().UTC() },
		Logger: logger.New(log.Default(), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}, &model.ReportLog{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := repository.New(db)
	bookings := client.NewBookingClient(bookingURL, internalKey)
	notifications := service.NewNotificationService(repo, bookings)
	reports := service.NewReportService(repo, bookings, reportDir)

	log.Printf("notification-service listening on :%s", port)
	if err := handler.NewRouter(notifications, reports, client.NewAuthClient(authURL), internalKey).Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env %s", key)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
