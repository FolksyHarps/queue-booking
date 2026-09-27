package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Service maps to table `services` in service_db.
// ProviderID references a user id in auth_db (no real FK across databases).
type Service struct {
	ID          uint      `gorm:"primaryKey;type:serial"`
	ProviderID  uint      `gorm:"type:integer;not null;index"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description string    `gorm:"type:text"`
	DurationMin int       `gorm:"type:integer;not null"`
	Price       float64   `gorm:"type:decimal(10,2);not null"`
	Category    string    `gorm:"type:varchar(50);index"`
	CreatedAt   time.Time `gorm:"type:timestamp;not null"`
}

// Availability maps to table `availability` (FK -> services.id).
type Availability struct {
	ID        uint  `gorm:"primaryKey;type:serial"`
	ServiceID uint  `gorm:"type:integer;not null;index"`
	DayOfWeek int   `gorm:"type:integer;not null"` // 0-6 = Sunday-Saturday
	StartTime Clock `gorm:"type:time without time zone;not null"`
	EndTime   Clock `gorm:"type:time without time zone;not null"`
}

func (Availability) TableName() string { return "availability" }

// Clock is a Postgres TIME value represented as "HH:MM".
type Clock string

func (c *Clock) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		*c = Clock(v.Format("15:04"))
	case string:
		*c = Clock(firstN(v, 5))
	case []byte:
		*c = Clock(firstN(string(v), 5))
	default:
		return fmt.Errorf("cannot scan %T into Clock", src)
	}
	return nil
}

func (c Clock) Value() (driver.Value, error) { return string(c), nil }

// Minutes returns minutes since midnight.
func (c Clock) Minutes() int {
	t, _ := time.Parse("15:04", string(c))
	return t.Hour()*60 + t.Minute()
}

func firstN(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
