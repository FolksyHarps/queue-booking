package service

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/queue-booking/booking-service/internal/client"
	"github.com/queue-booking/booking-service/internal/model"
	"github.com/queue-booking/booking-service/internal/repository"
)

var (
	ErrNotFound        = errors.New("booking not found")
	ErrServiceNotFound = errors.New("service not found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalid         = errors.New("invalid input")
	ErrSlotUnavailable = errors.New("time slot is already booked")
	ErrBadState        = errors.New("booking cannot be changed in its current status")
)

type BookingService struct {
	repo     *repository.BookingRepository
	services *client.ServiceClient
}

func NewBookingService(repo *repository.BookingRepository, services *client.ServiceClient) *BookingService {
	return &BookingService{repo: repo, services: services}
}

func isAdmin(u *client.CurrentUser) bool { return u.Role == "admin" || u.Role == "system" }

// List returns the caller's own bookings (customer), bookings of services they
// own (provider), or everything (admin).
func (s *BookingService) List(u *client.CurrentUser, status string, page, limit int) ([]model.Booking, error) {
	f := repository.ListFilter{Status: status, Page: page, Limit: limit}
	switch {
	case isAdmin(u):
	case u.Role == "provider":
		ids, err := s.services.IDsByProvider(u.ID)
		if err != nil {
			return nil, err
		}
		f.ServiceIDs = ids
	default:
		f.CustomerID = u.ID
	}
	return s.repo.List(f)
}

func (s *BookingService) Get(u *client.CurrentUser, id uint) (*model.Booking, error) {
	b, err := s.find(id)
	if err != nil {
		return nil, err
	}
	if isAdmin(u) || b.CustomerID == u.ID {
		return b, nil
	}
	if ok, err := s.ownsService(u, b.ServiceID); err != nil || !ok {
		return nil, orForbidden(err)
	}
	return b, nil
}

func (s *BookingService) Create(u *client.CurrentUser, serviceID uint, date, t string) (*model.Booking, error) {
	d, c, err := s.checkSlot(serviceID, date, t, 0)
	if err != nil {
		return nil, err
	}
	b := &model.Booking{CustomerID: u.ID, ServiceID: serviceID, Date: d, Time: c, Status: model.StatusPending}
	if err := s.repo.Create(b); err != nil {
		return nil, slotErr(err)
	}
	return b, nil
}

// Cancel is allowed for the booking's customer, the service's provider, or an admin.
func (s *BookingService) Cancel(u *client.CurrentUser, id uint) (*model.Booking, error) {
	b, err := s.Get(u, id)
	if err != nil {
		return nil, err
	}
	if b.Status != model.StatusPending {
		return nil, ErrBadState
	}
	b.Status = model.StatusCancelled
	return b, s.repo.Save(b)
}

// Reschedule is allowed for the booking's customer or an admin.
func (s *BookingService) Reschedule(u *client.CurrentUser, id uint, date, t string) (*model.Booking, error) {
	b, err := s.find(id)
	if err != nil {
		return nil, err
	}
	if !isAdmin(u) && b.CustomerID != u.ID {
		return nil, ErrForbidden
	}
	if b.Status != model.StatusPending {
		return nil, ErrBadState
	}
	d, c, err := s.checkSlot(b.ServiceID, date, t, b.ID)
	if err != nil {
		return nil, err
	}
	b.Date, b.Time = d, c
	if err := s.repo.Save(b); err != nil {
		return nil, slotErr(err)
	}
	return b, nil
}

// UpdateStatus is for the service's provider (or admin). Setting in_progress
// on a booking for today also moves the service's queue to this booking.
func (s *BookingService) UpdateStatus(u *client.CurrentUser, id uint, status string) (*model.Booking, error) {
	b, err := s.find(id)
	if err != nil {
		return nil, err
	}
	if !isAdmin(u) {
		if ok, err := s.ownsService(u, b.ServiceID); err != nil || !ok {
			return nil, orForbidden(err)
		}
	}
	allowed := map[string][]string{
		model.StatusPending:    {model.StatusInProgress, model.StatusCompleted, model.StatusNoShow},
		model.StatusInProgress: {model.StatusCompleted, model.StatusNoShow},
	}
	if !slices.Contains(allowed[b.Status], status) {
		return nil, ErrBadState
	}
	b.Status = status
	if err := s.repo.Save(b); err != nil {
		return nil, err
	}
	if status == model.StatusInProgress && string(b.Date) == time.Now().Format("2006-01-02") {
		if err := s.repo.SetCurrentServing(b.ServiceID, b.ID); err != nil {
			return nil, err
		}
	}
	return b, nil
}

type Queue struct {
	ServiceID      uint   `json:"service_id"`
	CurrentServing *uint  `json:"current_serving"`
	Waiting        []uint `json:"waiting"`
}

// Queue shows today's queue of a service: the booking being served and the
// pending bookings after it.
func (s *BookingService) Queue(serviceID uint) (*Queue, error) {
	if _, err := s.services.Get(serviceID); err != nil {
		if errors.Is(err, client.ErrServiceNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	q := &Queue{ServiceID: serviceID}
	qs, err := s.repo.QueueStatus(serviceID)
	if err != nil {
		return nil, err
	}
	if qs != nil && qs.UpdatedAt.In(time.Local).Format("2006-01-02") == today {
		q.CurrentServing = &qs.CurrentServing
	}
	q.Waiting, err = s.repo.WaitingIDs(serviceID, model.Date(today))
	return q, err
}

// checkSlot validates date/time format, that the slot is in the future, offered
// by the service's availability, and not held by another booking.
func (s *BookingService) checkSlot(serviceID uint, date, t string, excludeID uint) (model.Date, model.Clock, error) {
	at, err := time.ParseInLocation("2006-01-02 15:04", date+" "+t, time.Local)
	if err != nil {
		return "", "", fmt.Errorf("%w: date must be YYYY-MM-DD and time HH:MM", ErrInvalid)
	}
	if !at.After(time.Now()) {
		return "", "", fmt.Errorf("%w: appointment must be in the future", ErrInvalid)
	}
	slots, err := s.services.Slots(serviceID, date)
	if errors.Is(err, client.ErrServiceNotFound) {
		return "", "", fmt.Errorf("%w: service not found", ErrInvalid)
	}
	if err != nil {
		return "", "", err
	}
	if !slices.Contains(slots, t) {
		return "", "", fmt.Errorf("%w: service is not available at %s %s", ErrInvalid, date, t)
	}
	d, c := model.Date(date), model.Clock(t)
	taken, err := s.repo.SlotTaken(serviceID, d, c, excludeID)
	if err != nil {
		return "", "", err
	}
	if taken {
		return "", "", ErrSlotUnavailable
	}
	return d, c, nil
}

func (s *BookingService) find(id uint) (*model.Booking, error) {
	b, err := s.repo.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return b, err
}

func (s *BookingService) ownsService(u *client.CurrentUser, serviceID uint) (bool, error) {
	if u.Role != "provider" {
		return false, nil
	}
	svc, err := s.services.Get(serviceID)
	if errors.Is(err, client.ErrServiceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return svc.ProviderID == u.ID, nil
}

func orForbidden(err error) error {
	if err != nil {
		return err
	}
	return ErrForbidden
}

// slotErr maps the DB unique-index violation (concurrent booking) to a conflict.
func slotErr(err error) error {
	if errors.Is(err, repository.ErrSlotTaken) {
		return ErrSlotUnavailable
	}
	return err
}
