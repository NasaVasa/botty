package usecase

import (
	"context"

	"github.com/NasaVasa/botty/internal/domain"
)

type UserUsecase struct {
	users domain.UserRepository
}

func NewUserUsecase(users domain.UserRepository) *UserUsecase {
	return &UserUsecase{users: users}
}

func (u *UserUsecase) StartOrGetUser(ctx context.Context, telegramUserID int64, username string) (*domain.User, error) {
	user, err := u.users.GetByTelegramID(ctx, telegramUserID)
	if err == nil {
		return user, nil
	}
	if err != domain.ErrNotFound {
		return nil, err
	}

	newUser := &domain.User{
		TelegramUserID: telegramUserID,
		Username:       username,
	}
	if err := u.users.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}
