package domain

import "time"

type User struct {
	ID             uint
	TelegramUserID int64
	Username       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}
