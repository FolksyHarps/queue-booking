package repository

import (
	"errors"

	"github.com/queue-booking/booking-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrSlotTaken = errors.New("slot already booked")
)

// EnsureIndexes creates constraints AutoMigrate cannot express: at most one
// active (non-cancelled) booking per service/date/time slot.
func EnsureIndexes(db *gorm.DB) error {
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_bookings_active_slot
		ON bookings (service_id, date, time) WHERE status <> 'cancelled'`).Error
}

type ListFilter struct {
	CustomerID uint
	ServiceIDs []uint // nil = no filter; empty = match nothing
	Status     string
	Page       int
	Limit      int
}

type BookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) List(f ListFilter) ([]model.Booking, error) {
	q := r.db.Order("id")
	if f.CustomerID != 0 {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.ServiceIDs != nil {
		if len(f.ServiceIDs) == 0 {
			return []model.Booking{}, nil
		}
		q = q.Where("service_id IN ?", f.ServiceIDs)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var list []model.Booking
	err := q.Offset((f.Page - 1) * f.Limit).Limit(f.Limit).Find(&list).Error
	return list, err
}

func (r *BookingRepository) FindByID(id uint) (*model.Booking, error) {
	var b model.Booking
	if err := r.db.First(&b, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *BookingRepository) Create(b *model.Booking) error {
	return translate(r.db.Create(b).Error)
}

func (r *BookingRepository) Save(b *model.Booking) error {
	return translate(r.db.Save(b).Error)
}

// SlotTaken reports whether another active booking holds the slot.
func (r *BookingRepository) SlotTaken(serviceID uint, date model.Date, t model.Clock, excludeID uint) (bool, error) {
	var n int64
	err := r.db.Model(&model.Booking{}).
		Where("service_id = ? AND date = ? AND time = ? AND status <> ? AND id <> ?",
			serviceID, date, t, model.StatusCancelled, excludeID).
		Count(&n).Error
	return n > 0, err
}

// WaitingIDs lists pending bookings of a service on a date in queue order.
func (r *BookingRepository) WaitingIDs(serviceID uint, date model.Date) ([]uint, error) {
	ids := []uint{}
	err := r.db.Model(&model.Booking{}).
		Where("service_id = ? AND date = ? AND status = ?", serviceID, date, model.StatusPending).
		Order("time, id").Pluck("id", &ids).Error
	return ids, err
}

func (r *BookingRepository) SetCurrentServing(serviceID, bookingID uint) error {
	qs := model.QueueStatus{ServiceID: serviceID, CurrentServing: bookingID}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "service_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"current_serving", "updated_at"}),
	}).Create(&qs).Error
}

func (r *BookingRepository) QueueStatus(serviceID uint) (*model.QueueStatus, error) {
	var qs model.QueueStatus
	err := r.db.Where("service_id = ?", serviceID).First(&qs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &qs, err
}

func translate(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrSlotTaken
	}
	return err
}
