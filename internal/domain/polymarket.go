package domain

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

var ErrEventNotFound = errors.New("event not found")

type MarketInfo struct {
	Slug          string
	ConditionID   string
	Outcomes      []string
	ClobTokenIDs  []string
	Question      string
	BestBid       *decimal.Decimal
	BestAsk       *decimal.Decimal
	LastTrade     *decimal.Decimal
	OutcomePrices []string
}

type EventMarkets struct {
	EventSlug string
	Markets   []MarketInfo
}

type GammaClient interface {
	GetEventBySlug(ctx context.Context, slug string) (*EventMarkets, error)
}

type PriceChange struct {
	AssetID string
	BestBid *decimal.Decimal
	BestAsk *decimal.Decimal
	Price   *decimal.Decimal
}

type PriceChangeMessage struct {
	EventType    string
	PriceChanges []PriceChange
}

type MarketWSClient interface {
	Subscribe(ctx context.Context, assetIDs []string) error
	Receive(ctx context.Context) (*PriceChangeMessage, error)
	Close() error
}

type MarketWSFactory interface {
	Connect(ctx context.Context) (MarketWSClient, error)
}
