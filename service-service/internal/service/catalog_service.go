package service

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/queue-booking/service-service/internal/client"
	"github.com/queue-booking/service-service/internal/model"
	"github.com/queue-booking/service-service/internal/repository"
)

var (
	ErrNotFound  = errors.New("service not found")
	ErrForbidden = errors.New("only the service owner or an admin can do this")
	ErrInvalid   = errors.New("invalid input")
)

type CatalogService struct {
	repo *repository.ServiceRepository
}

func NewCatalogService(repo *repository.ServiceRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) List(f repository.ListFilter) ([]model.Service, error) {
	return s.repo.List(f)
}

func (s *CatalogService) Get(id uint) (*model.Service, error) {
	svc, err := s.repo.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return svc, err
}

func (s *CatalogService) Create(owner *client.CurrentUser, svc *model.Service) error {
	svc.ProviderID = owner.ID
	return s.repo.Create(svc)
}

type UpdateInput struct {
	Name        *string
	DurationMin *int
	Price       *float64
}

func (s *CatalogService) Update(user *client.CurrentUser, id uint, in UpdateInput) (*model.Service, error) {
	svc, err := s.getOwned(user, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		svc.Name = *in.Name
	}
	if in.DurationMin != nil {
		svc.DurationMin = *in.DurationMin
	}
	if in.Price != nil {
		svc.Price = *in.Price
	}
	return svc, s.repo.Update(svc)
}

func (s *CatalogService) Delete(user *client.CurrentUser, id uint) error {
	if _, err := s.getOwned(user, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *CatalogService) AddAvailability(user *client.CurrentUser, id uint, day int, start, end string) (*model.Availability, error) {
	if _, err := s.getOwned(user, id); err != nil {
		return nil, err
	}
	st, err1 := time.Parse("15:04", start)
	et, err2 := time.Parse("15:04", end)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("%w: start_time/end_time must be HH:MM", ErrInvalid)
	}
	if !st.Before(et) {
		return nil, fmt.Errorf("%w: start_time must be before end_time", ErrInvalid)
	}
	a := &model.Availability{ServiceID: id, DayOfWeek: day, StartTime: model.Clock(start), EndTime: model.Clock(end)}
	return a, s.repo.CreateAvailability(a)
}

// Slots lists bookable start times on the given date: each availability window
// for that weekday is split into consecutive blocks of the service's duration.
func (s *CatalogService) Slots(id uint, date time.Time) ([]string, error) {
	svc, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	windows, err := s.repo.AvailabilityForDay(id, int(date.Weekday()))
	if err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, w := range windows {
		for m := w.StartTime.Minutes(); m+svc.DurationMin <= w.EndTime.Minutes(); m += svc.DurationMin {
			seen[m] = true
		}
	}
	mins := make([]int, 0, len(seen))
	for m := range seen {
		mins = append(mins, m)
	}
	sort.Ints(mins)
	slots := make([]string, 0, len(mins))
	for _, m := range mins {
		slots = append(slots, fmt.Sprintf("%02d:%02d", m/60, m%60))
	}
	return slots, nil
}

func (s *CatalogService) getOwned(user *client.CurrentUser, id uint) (*model.Service, error) {
	svc, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if user.Role != "admin" && svc.ProviderID != user.ID {
		return nil, ErrForbidden
	}
	return svc, nil
}
