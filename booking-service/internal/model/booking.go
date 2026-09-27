package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
	StatusNoShow     = "no_show"
)

// Booking maps to table `bookings` in booking_db.
// CustomerID / ServiceID reference other services' databases (no real FK).
type Booking struct {
	ID         uint      `gorm:"primaryKey;type:serial"`
	CustomerID uint      `gorm:"type:integer;not null;index"`
	ServiceID  uint      `gorm:"type:integer;not null;index"`
	Date       Date      `gorm:"type:date;not null"`
	Time       Clock     `gorm:"type:time without time zone;not null"`
	Status     string    `gorm:"type:varchar(20);not null;index"`
	CreatedAt  time.Time `gorm:"type:timestamp;not null"`
	UpdatedAt  time.Time `gorm:"type:timestamp;not null"`
}

// QueueStatus maps to table `queue_status`: which booking is being served now.
type QueueStatus struct {
	ID             uint      `gorm:"primaryKey;type:serial"`
	ServiceID      uint      `gorm:"type:integer;not null;uniqueIndex"`
	CurrentServing uint      `gorm:"type:integer;not null"`
	UpdatedAt      time.Time `gorm:"type:timestamp;not null"`
}

func (QueueStatus) TableName() string { return "queue_status" }

// Date is a Postgres DATE value represented as "YYYY-MM-DD".
type Date string

func (d *Date) Scan(src any) error {
	s, err := scanString(src, "2006-01-02")
	*d = Date(firstN(s, 10))
	return err
}

func (d Date) Value() (driver.Value, error) { return string(d), nil }

// Clock is a Postgres TIME value represented as "HH:MM".
type Clock string

func (c *Clock) Scan(src any) error {
	s, err := scanString(src, "15:04")
	*c = Clock(firstN(s, 5))
	return err
}

func (c Clock) Value() (driver.Value, error) { return string(c), nil }

func scanString(src any, layout string) (string, error) {
	switch v := src.(type) {
	case time.Time:
		return v.Format(layout), nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("cannot scan %T", src)
	}
}

func firstN(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
