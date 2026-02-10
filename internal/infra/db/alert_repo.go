package db

import (
	"context"
	"time"

	"github.com/NasaVasa/botty/internal/domain"
	"gorm.io/gorm"
)

type AlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Create(ctx context.Context, alert *domain.Alert) error {
	model := mapAlertToModel(*alert)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	alert.ID = model.ID
	alert.CreatedAt = model.CreatedAt
	alert.UpdatedAt = model.UpdatedAt
	if model.DeletedAt.Valid {
		deleted := model.DeletedAt.Time
		alert.DeletedAt = &deleted
	}
	return nil
}

func (r *AlertRepository) ListByUser(ctx context.Context, userID uint) ([]domain.Alert, error) {
	var models []alertModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id").Find(&models).Error; err != nil {
		return nil, err
	}
	return mapAlertsToDomain(models), nil
}

func (r *AlertRepository) ListEnabledByUser(ctx context.Context, userID uint) ([]domain.Alert, error) {
	var models []alertModel
	if err := r.db.WithContext(ctx).Where("user_id = ? AND enabled = ?", userID, true).Order("id").Find(&models).Error; err != nil {
		return nil, err
	}
	return mapAlertsToDomain(models), nil
}

func (r *AlertRepository) SetEnabled(ctx context.Context, userID uint, alertID uint, enabled bool) error {
	result := r.db.WithContext(ctx).Model(&alertModel{}).Where("id = ? AND user_id = ?", alertID, userID).Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AlertRepository) Delete(ctx context.Context, userID uint, alertID uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", alertID, userID).Delete(&alertModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AlertRepository) ListUserIDsWithEnabledAlerts(ctx context.Context) ([]uint, error) {
	var userIDs []uint
	if err := r.db.WithContext(ctx).
		Model(&alertModel{}).
		Where("enabled = ?", true).
		Distinct().
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, err
	}
	return userIDs, nil
}

func mapAlertsToDomain(models []alertModel) []domain.Alert {
	alerts := make([]domain.Alert, 0, len(models))
	for _, model := range models {
		var deleted *time.Time
		if model.DeletedAt.Valid {
			t := model.DeletedAt.Time
			deleted = &t
		}
		alerts = append(alerts, domain.Alert{
			ID:          model.ID,
			UserID:      model.UserID,
			MarketSlug:  model.MarketSlug,
			ConditionID: model.ConditionID,
			Outcome:     model.Outcome,
			AssetID:     model.AssetID,
			Comparator:  model.Comparator,
			Threshold:   model.Threshold,
			Enabled:     model.Enabled,
			CreatedAt:   model.CreatedAt,
			UpdatedAt:   model.UpdatedAt,
			DeletedAt:   deleted,
		})
	}
	return alerts
}

func mapAlertToModel(alert domain.Alert) alertModel {
	return alertModel{
		ID:          alert.ID,
		UserID:      alert.UserID,
		MarketSlug:  alert.MarketSlug,
		ConditionID: alert.ConditionID,
		Outcome:     alert.Outcome,
		AssetID:     alert.AssetID,
		Comparator:  alert.Comparator,
		Threshold:   alert.Threshold,
		Enabled:     alert.Enabled,
		CreatedAt:   alert.CreatedAt,
		UpdatedAt:   alert.UpdatedAt,
	}
}
