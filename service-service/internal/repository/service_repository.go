package repository

import (
	"errors"

	"github.com/queue-booking/service-service/internal/model"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("record not found")

type ListFilter struct {
	Category   string
	Keyword    string
	ProviderID uint
}

type ServiceRepository struct {
	db *gorm.DB
}

func NewServiceRepository(db *gorm.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

func (r *ServiceRepository) List(f ListFilter) ([]model.Service, error) {
	q := r.db.Order("id")
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ?", kw, kw)
	}
	if f.ProviderID != 0 {
		q = q.Where("provider_id = ?", f.ProviderID)
	}
	var list []model.Service
	return list, q.Find(&list).Error
}

func (r *ServiceRepository) FindByID(id uint) (*model.Service, error) {
	var s model.Service
	if err := r.db.First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepository) Create(s *model.Service) error { return r.db.Create(s).Error }

func (r *ServiceRepository) Update(s *model.Service) error {
	return r.db.Save(s).Error
}

func (r *ServiceRepository) Delete(id uint) error {
	return r.db.Delete(&model.Service{}, id).Error // availability rows go via ON DELETE CASCADE
}

func (r *ServiceRepository) CreateAvailability(a *model.Availability) error {
	return r.db.Create(a).Error
}

func (r *ServiceRepository) AvailabilityForDay(serviceID uint, dayOfWeek int) ([]model.Availability, error) {
	var list []model.Availability
	err := r.db.Where("service_id = ? AND day_of_week = ?", serviceID, dayOfWeek).
		Order("start_time").Find(&list).Error
	return list, err
}

// EnsureForeignKeys adds availability.service_id -> services.id (ON DELETE CASCADE).
// It is done in SQL rather than via a GORM association because GORM copies the
// `serial` primary-key type onto association foreign keys, breaking re-migration.
func EnsureForeignKeys(db *gorm.DB) error {
	return db.Exec(`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_availability_service') THEN
			ALTER TABLE availability ADD CONSTRAINT fk_availability_service
				FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;
		END IF;
	END $$`).Error
}
