package model

import "time"

// Notification maps to table `notifications` in notification_db.
// UserID / BookingID reference other services' databases (no real FK).
type Notification struct {
	ID        uint      `gorm:"primaryKey;type:serial"`
	UserID    uint      `gorm:"type:integer;not null;index"`
	BookingID uint      `gorm:"type:integer;not null;index"`
	Message   string    `gorm:"type:text;not null"`
	Channel   string    `gorm:"type:varchar(20);not null"`
	IsRead    bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"type:timestamp;not null"`
}

// ReportLog maps to table `report_logs`: an audit trail of generated reports.
type ReportLog struct {
	ID          uint      `gorm:"primaryKey;type:serial"`
	ReportType  string    `gorm:"type:varchar(30);not null"`
	GeneratedBy uint      `gorm:"type:integer;not null"`
	FilePath    string    `gorm:"type:varchar(255)"`
	CreatedAt   time.Time `gorm:"type:timestamp;not null"`
}
