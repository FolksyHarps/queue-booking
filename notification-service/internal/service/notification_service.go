package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/queue-booking/notification-service/internal/client"
	"github.com/queue-booking/notification-service/internal/model"
	"github.com/queue-booking/notification-service/internal/repository"
)

var (
	ErrNotFound = errors.New("notification not found")
	ErrInvalid  = errors.New("invalid input")
)

type NotificationService struct {
	repo     *repository.Repository
	bookings *client.BookingClient
}

func NewNotificationService(repo *repository.Repository, bookings *client.BookingClient) *NotificationService {
	return &NotificationService{repo: repo, bookings: bookings}
}

func (s *NotificationService) List(userID uint, isRead *bool) ([]model.Notification, error) {
	return s.repo.ListForUser(userID, isRead)
}

// Send records an appointment reminder for the booking's customer.
func (s *NotificationService) Send(bookingID uint, channel string) (*model.Notification, error) {
	b, err := s.bookings.Get(bookingID)
	if errors.Is(err, client.ErrBookingNotFound) {
		return nil, fmt.Errorf("%w: booking not found", ErrInvalid)
	}
	if err != nil {
		return nil, err
	}
	if b.Status != "pending" && b.Status != "in_progress" {
		return nil, fmt.Errorf("%w: booking is %s", ErrInvalid, b.Status)
	}
	n := &model.Notification{
		UserID:    b.CustomerID,
		BookingID: b.ID,
		Message:   fmt.Sprintf("แจ้งเตือนนัดหมายของคุณ วันที่ %s เวลา %s น.", b.Date, b.Time),
		Channel:   channel,
	}
	if err := s.repo.Create(n); err != nil {
		return nil, err
	}
	if channel == "email" {
		// No mail provider is configured; the email delivery is only logged.
		log.Printf("email reminder for user %d (booking %d): %s", n.UserID, n.BookingID, n.Message)
	}
	return n, nil
}

func (s *NotificationService) MarkRead(id, userID uint) (*model.Notification, error) {
	n, err := s.repo.MarkRead(id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return n, err
}
