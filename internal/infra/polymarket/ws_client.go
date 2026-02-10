package polymarket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NasaVasa/botty/internal/domain"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSFactory struct {
	url         string
	dialer      *websocket.Dialer
	readTimeout time.Duration
	logger      *zap.Logger
}

func NewWSFactory(url string, readTimeout time.Duration, logger *zap.Logger) *WSFactory {
	return &WSFactory{
		url: url,
		dialer: &websocket.Dialer{
			Proxy: http.ProxyFromEnvironment,
		},
		readTimeout: readTimeout,
		logger:      logger,
	}
}

func (f *WSFactory) Connect(ctx context.Context) (domain.MarketWSClient, error) {
	f.logger.Info("ws connect start", zap.String("url", f.url))
	conn, _, err := f.dialer.DialContext(ctx, f.url, nil)
	if err != nil {
		f.logger.Error("ws connect failed", zap.String("url", f.url), zap.Error(err))
		return nil, err
	}
	f.logger.Info("ws connect success", zap.String("url", f.url))
	return &WSClient{conn: conn, readTimeout: f.readTimeout, logger: f.logger}, nil
}

type WSClient struct {
	conn        *websocket.Conn
	readTimeout time.Duration
	logger      *zap.Logger
}

func (c *WSClient) Subscribe(ctx context.Context, assetIDs []string) error {
	payload := map[string]any{
		"type":       "market",
		"assets_ids": assetIDs,
	}
	c.logger.Info("ws subscribe", zap.Int("asset_count", len(assetIDs)), zap.Strings("asset_ids", assetIDs))
	if err := c.conn.WriteJSON(payload); err != nil {
		c.logger.Error("ws subscribe failed", zap.Error(err))
		return err
	}
	return nil
}

func (c *WSClient) Receive(ctx context.Context) (*domain.PriceChangeMessage, error) {
	if c.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	message, err := c.decodeMessage(data)
	if err != nil {
		c.logger.Debug("ws message ignored", zap.Error(err))
		return nil, nil
	}

	return message, nil
}

func (c *WSClient) Close() error {
	c.logger.Info("ws close")
	return c.conn.Close()
}

func (c *WSClient) decodeMessage(data []byte) (*domain.PriceChangeMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	if trimmed[0] == '[' {
		var payloads []wsMessage
		if err := json.Unmarshal(trimmed, &payloads); err != nil {
			return nil, fmt.Errorf("decode ws message array: %w", err)
		}
		for _, payload := range payloads {
			if payload.EventType != "price_change" {
				continue
			}
			return mapPriceChange(payload), nil
		}
		return nil, nil
	}

	var payload wsMessage
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		return nil, fmt.Errorf("decode ws message: %w", err)
	}
	if payload.EventType != "price_change" {
		return nil, nil
	}
	return mapPriceChange(payload), nil
}

func mapPriceChange(payload wsMessage) *domain.PriceChangeMessage {
	message := &domain.PriceChangeMessage{
		EventType:    payload.EventType,
		PriceChanges: make([]domain.PriceChange, 0, len(payload.PriceChanges)),
	}

	for _, change := range payload.PriceChanges {
		msgChange := domain.PriceChange{AssetID: change.AssetID}
		if change.BestBid.Valid {
			value := change.BestBid.Decimal
			msgChange.BestBid = &value
		}
		if change.BestAsk.Valid {
			value := change.BestAsk.Decimal
			msgChange.BestAsk = &value
		}
		if change.Price.Valid {
			value := change.Price.Decimal
			msgChange.Price = &value
		}
		message.PriceChanges = append(message.PriceChanges, msgChange)
	}

	return message
}
