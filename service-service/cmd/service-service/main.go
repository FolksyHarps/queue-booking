package main

import (
	"log"
	"os"
	"time"

	"github.com/queue-booking/service-service/internal/client"
	"github.com/queue-booking/service-service/internal/handler"
	"github.com/queue-booking/service-service/internal/model"
	"github.com/queue-booking/service-service/internal/repository"
	"github.com/queue-booking/service-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := mustEnv("DB_DSN")
	authURL := mustEnv("AUTH_SERVICE_URL")
	port := envOr("PORT", "8082")

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
	if err := db.AutoMigrate(&model.Service{}, &model.Availability{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := repository.EnsureForeignKeys(db); err != nil {
		log.Fatalf("create foreign keys: %v", err)
	}

	catalog := service.NewCatalogService(repository.NewServiceRepository(db))

	log.Printf("service-service listening on :%s", port)
	if err := handler.NewRouter(catalog, client.NewAuthClient(authURL)).Run(":" + port); err != nil {
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
