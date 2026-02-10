package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NasaVasa/botty/internal/domain"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type GammaClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

func NewGammaClient(baseURL string, timeout time.Duration, logger *zap.Logger) *GammaClient {
	return &GammaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		logger:  logger,
	}
}

func (c *GammaClient) GetEventBySlug(ctx context.Context, slug string) (*domain.EventMarkets, error) {
	endpoint := fmt.Sprintf("%s/events/slug/%s", c.baseURL, url.PathEscape(slug))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	c.logger.Info("gamma request start", zap.String("slug", slug), zap.String("url", endpoint))
	response, err := c.client.Do(request)
	if err != nil {
		c.logger.Error("gamma request failed", zap.String("slug", slug), zap.String("url", endpoint), zap.Error(err))
		return nil, err
	}
	defer response.Body.Close()

	c.logger.Info(
		"gamma request complete",
		zap.String("slug", slug),
		zap.String("url", endpoint),
		zap.Int("status", response.StatusCode),
		zap.Duration("duration", time.Since(start)),
	)

	if response.StatusCode == http.StatusNotFound {
		return nil, domain.ErrEventNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("gamma error: status %d", response.StatusCode)
	}

	var payload gammaEventResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}

	event := &domain.EventMarkets{EventSlug: payload.Slug, Markets: make([]domain.MarketInfo, 0, len(payload.Markets))}
	for _, market := range payload.Markets {
		var bestBid *decimal.Decimal
		if market.BestBid.Valid {
			value := market.BestBid.Decimal
			bestBid = &value
		}
		var bestAsk *decimal.Decimal
		if market.BestAsk.Valid {
			value := market.BestAsk.Decimal
			bestAsk = &value
		}
		var lastTrade *decimal.Decimal
		if market.LastTradePrice.Valid {
			value := market.LastTradePrice.Decimal
			lastTrade = &value
		}

		event.Markets = append(event.Markets, domain.MarketInfo{
			Slug:          market.Slug,
			ConditionID:   market.ConditionID,
			Outcomes:      []string(market.Outcomes),
			ClobTokenIDs:  []string(market.ClobTokenIDs),
			Question:      market.Question,
			OutcomePrices: []string(market.OutcomePrices),
			BestBid:       bestBid,
			BestAsk:       bestAsk,
			LastTrade:     lastTrade,
		})
	}

	return event, nil
}
