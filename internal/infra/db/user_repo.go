package db

import (
	"context"
	"time"

	"github.com/NasaVasa/botty/internal/domain"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramUserID int64) (*domain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapUserToDomain(model), nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID uint) (*domain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).First(&model, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return mapUserToDomain(model), nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	model := mapUserToModel(*user)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	user.ID = model.ID
	user.CreatedAt = model.CreatedAt
	user.UpdatedAt = model.UpdatedAt
	if model.DeletedAt.Valid {
		deleted := model.DeletedAt.Time
		user.DeletedAt = &deleted
	}
	return nil
}

func mapUserToDomain(model userModel) *domain.User {
	var deleted *time.Time
	if model.DeletedAt.Valid {
		t := model.DeletedAt.Time
		deleted = &t
	}
	return &domain.User{
		ID:             model.ID,
		TelegramUserID: model.TelegramUserID,
		Username:       model.Username,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
		DeletedAt:      deleted,
	}
}

func mapUserToModel(user domain.User) userModel {
	model := userModel{
		ID:             user.ID,
		TelegramUserID: user.TelegramUserID,
		Username:       user.Username,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
	return model
}
