package main

import (
	"log"
	"os"
	"time"
	_ "time/tzdata" // appointment dates/times use the local TZ (Asia/Bangkok)

	"github.com/queue-booking/booking-service/internal/client"
	"github.com/queue-booking/booking-service/internal/handler"
	"github.com/queue-booking/booking-service/internal/model"
	"github.com/queue-booking/booking-service/internal/repository"
	"github.com/queue-booking/booking-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := mustEnv("DB_DSN")
	authURL := mustEnv("AUTH_SERVICE_URL")
	serviceURL := mustEnv("SERVICE_SERVICE_URL")
	internalKey := mustEnv("INTERNAL_API_KEY")
	port := envOr("PORT", "8083")

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
	if err := db.AutoMigrate(&model.Booking{}, &model.QueueStatus{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := repository.EnsureIndexes(db); err != nil {
		log.Fatalf("create indexes: %v", err)
	}

	bookings := service.NewBookingService(repository.NewBookingRepository(db), client.NewServiceClient(serviceURL))

	log.Printf("booking-service listening on :%s", port)
	if err := handler.NewRouter(bookings, client.NewAuthClient(authURL), internalKey).Run(":" + port); err != nil {
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
