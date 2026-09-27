package service

import (
	"errors"

	"github.com/queue-booking/auth-service/internal/model"
	"github.com/queue-booking/auth-service/internal/repository"
)

type UserService struct {
	users *repository.UserRepository
}

func NewUserService(users *repository.UserRepository) *UserService {
	return &UserService{users: users}
}

// UpdateProfile applies only the fields that were provided (nil = unchanged).
func (s *UserService) UpdateProfile(u *model.User, name, phone *string) (*model.User, error) {
	if name != nil {
		u.Name = *name
	}
	if phone != nil {
		u.Phone = *phone
	}
	if err := s.users.Update(u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) List(role string, page, limit int) ([]model.User, int64, error) {
	return s.users.List(role, page, limit)
}

func (s *UserService) SetStatus(id uint, status string) (*model.User, error) {
	u, err := s.users.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Status = status
	if err := s.users.Update(u); err != nil {
		return nil, err
	}
	return u, nil
}
