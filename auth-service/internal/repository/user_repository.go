package repository

import (
	"errors"

	"github.com/queue-booking/auth-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrDuplicateEmail = errors.New("email already exists")
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *model.User) error {
	err := r.db.Create(u).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateEmail
	}
	return err
}

func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Update(u *model.User) error {
	return r.db.Save(u).Error
}

// List returns one page of users (optionally filtered by role) and the total match count.
func (r *UserRepository) List(role string, page, limit int) ([]model.User, int64, error) {
	q := r.db.Model(&model.User{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	err := q.Order("id").Offset((page - 1) * limit).Limit(limit).Find(&users).Error
	return users, total, err
}
