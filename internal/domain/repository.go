package domain

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")

type UserRepository interface {
	GetByTelegramID(ctx context.Context, telegramUserID int64) (*User, error)
	GetByID(ctx context.Context, userID uint) (*User, error)
	Create(ctx context.Context, user *User) error
}

type AlertRepository interface {
	Create(ctx context.Context, alert *Alert) error
	ListByUser(ctx context.Context, userID uint) ([]Alert, error)
	ListEnabledByUser(ctx context.Context, userID uint) ([]Alert, error)
	SetEnabled(ctx context.Context, userID uint, alertID uint, enabled bool) error
	Delete(ctx context.Context, userID uint, alertID uint) error
	ListUserIDsWithEnabledAlerts(ctx context.Context) ([]uint, error)
}
