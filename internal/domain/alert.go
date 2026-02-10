package domain

import "time"

type Alert struct {
	ID          uint
	UserID      uint
	MarketSlug  string
	ConditionID string
	Outcome     string
	AssetID     string
	Comparator  string
	Threshold   string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
