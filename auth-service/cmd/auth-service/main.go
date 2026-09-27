package main

import (
	"log"
	"os"
	"time"

	"github.com/queue-booking/auth-service/internal/handler"
	"github.com/queue-booking/auth-service/internal/model"
	"github.com/queue-booking/auth-service/internal/repository"
	"github.com/queue-booking/auth-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := mustEnv("DB_DSN")
	secret := mustEnv("JWT_SECRET")
	port := envOr("PORT", "8081")

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
	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, service.NewTokenManager(secret, time.Hour))
	userSvc := service.NewUserService(userRepo)

	if err := authSvc.EnsureAdmin(os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	log.Printf("auth-service listening on :%s", port)
	if err := handler.NewRouter(authSvc, userSvc).Run(":" + port); err != nil {
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
