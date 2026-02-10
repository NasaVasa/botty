package db

import (
	"time"

	"gorm.io/gorm"
)

type userModel struct {
	ID             uint   `gorm:"primaryKey"`
	TelegramUserID int64  `gorm:"uniqueIndex;not null"`
	Username       string `gorm:""`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type alertModel struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"index:idx_alerts_user_enabled_deleted,priority:1;not null"`
	MarketSlug  string `gorm:"not null"`
	ConditionID string `gorm:"not null"`
	Outcome     string `gorm:"not null"`
	AssetID     string `gorm:"not null"`
	Comparator  string `gorm:"not null"`
	Threshold   string `gorm:"not null"`
	Enabled     bool   `gorm:"index:idx_alerts_user_enabled_deleted,priority:2"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index:idx_alerts_user_enabled_deleted,priority:3"`
}
