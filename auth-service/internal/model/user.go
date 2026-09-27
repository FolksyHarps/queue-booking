package model

import "time"

const (
	RoleCustomer = "customer"
	RoleProvider = "provider"
	RoleAdmin    = "admin"

	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// User maps to table `users` in auth_db.
type User struct {
	ID           uint      `gorm:"primaryKey;type:serial"`
	Name         string    `gorm:"type:varchar(100);not null"`
	Email        string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	Role         string    `gorm:"type:varchar(20);not null"`
	Phone        string    `gorm:"type:varchar(20)"`
	Status       string    `gorm:"type:varchar(20);not null;default:active"`
	CreatedAt    time.Time `gorm:"type:timestamp;not null"`
}
