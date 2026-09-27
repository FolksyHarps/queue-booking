package repository

import (
	"errors"

	"github.com/queue-booking/notification-service/internal/model"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("record not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ListForUser returns a user's notifications, newest first; isRead nil = all.
func (r *Repository) ListForUser(userID uint, isRead *bool) ([]model.Notification, error) {
	q := r.db.Where("user_id = ?", userID).Order("created_at DESC, id DESC")
	if isRead != nil {
		q = q.Where("is_read = ?", *isRead)
	}
	var list []model.Notification
	return list, q.Find(&list).Error
}

func (r *Repository) Create(n *model.Notification) error { return r.db.Create(n).Error }

// MarkRead sets is_read on a notification owned by userID.
func (r *Repository) MarkRead(id, userID uint) (*model.Notification, error) {
	var n model.Notification
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	n.IsRead = true
	return &n, r.db.Model(&n).Update("is_read", true).Error
}

func (r *Repository) LogReport(l *model.ReportLog) error { return r.db.Create(l).Error }
