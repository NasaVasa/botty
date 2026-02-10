package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NasaVasa/botty/internal/domain"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Notifier interface {
	Notify(telegramUserID int64, text string) error
}

type AlertingManager struct {
	users     domain.UserRepository
	alerts    domain.AlertRepository
	wsFactory domain.MarketWSFactory
	notifier  Notifier
	logger    *zap.Logger

	mu      sync.Mutex
	runners map[int64]*userRunner
}

type userRunner struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewAlertingManager(users domain.UserRepository, alerts domain.AlertRepository, wsFactory domain.MarketWSFactory, notifier Notifier, logger *zap.Logger) *AlertingManager {
	return &AlertingManager{
		users:     users,
		alerts:    alerts,
		wsFactory: wsFactory,
		notifier:  notifier,
		logger:    logger,
		runners:   make(map[int64]*userRunner),
	}
}

func (m *AlertingManager) StartAll(ctx context.Context) error {
	userIDs, err := m.alerts.ListUserIDsWithEnabledAlerts(ctx)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		user, err := m.users.GetByID(ctx, userID)
		if err != nil {
			if err == domain.ErrNotFound {
				continue
			}
			m.logger.Warn("failed to load user for alerting", zap.Uint("user_id", userID), zap.Error(err))
			continue
		}
		m.startUser(ctx, user)
	}
	return nil
}

func (m *AlertingManager) RestartUser(ctx context.Context, telegramUserID int64) {
	m.StopUser(telegramUserID)
	user, err := m.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		if err != domain.ErrNotFound {
			m.logger.Warn("failed to load user for alerting", zap.Int64("telegram_user_id", telegramUserID), zap.Error(err))
		}
		return
	}
	m.startUser(ctx, user)
}

func (m *AlertingManager) StopUser(telegramUserID int64) {
	m.mu.Lock()
	runner, ok := m.runners[telegramUserID]
	if ok {
		delete(m.runners, telegramUserID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	runner.cancel()
	select {
	case <-runner.done:
	case <-time.After(5 * time.Second):
		m.logger.Warn("timeout stopping alerting runner", zap.Int64("telegram_user_id", telegramUserID))
	}
}

func (m *AlertingManager) StopAll() {
	m.mu.Lock()
	ids := make([]int64, 0, len(m.runners))
	for id := range m.runners {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.StopUser(id)
	}
}

func (m *AlertingManager) startUser(ctx context.Context, user *domain.User) {
	alerts, err := m.alerts.ListEnabledByUser(ctx, user.ID)
	if err != nil {
		m.logger.Warn("failed to load alerts", zap.Int64("telegram_user_id", user.TelegramUserID), zap.Error(err))
		return
	}
	if len(alerts) == 0 {
		return
	}

	m.mu.Lock()
	if existing, ok := m.runners[user.TelegramUserID]; ok {
		m.mu.Unlock()
		m.logger.Debug("alerting runner already active", zap.Int64("telegram_user_id", user.TelegramUserID))
		existing.cancel()
		<-existing.done
	} else {
		m.mu.Unlock()
	}

	childCtx, cancel := context.WithCancel(ctx)
	runner := &userRunner{cancel: cancel, done: make(chan struct{})}

	m.mu.Lock()
	m.runners[user.TelegramUserID] = runner
	m.mu.Unlock()

	go func() {
		defer close(runner.done)
		m.runUser(childCtx, user, alerts)
	}()
}

type alertEval struct {
	AlertID    uint
	MarketSlug string
	Outcome    string
	Comparator string
	Threshold  decimal.Decimal
}

func (m *AlertingManager) runUser(ctx context.Context, user *domain.User, alerts []domain.Alert) {
	assetAlerts := make(map[string][]alertEval)
	assetIDs := make([]string, 0, len(alerts))

	for _, alert := range alerts {
		threshold, err := decimal.NewFromString(alert.Threshold)
		if err != nil {
			m.logger.Warn("invalid threshold on alert", zap.Uint("alert_id", alert.ID), zap.Error(err))
			continue
		}
		eval := alertEval{
			AlertID:    alert.ID,
			MarketSlug: alert.MarketSlug,
			Outcome:    alert.Outcome,
			Comparator: alert.Comparator,
			Threshold:  threshold,
		}
		assetAlerts[alert.AssetID] = append(assetAlerts[alert.AssetID], eval)
	}

	for assetID := range assetAlerts {
		assetIDs = append(assetIDs, assetID)
	}

	if len(assetIDs) == 0 {
		return
	}

	client, err := m.wsFactory.Connect(ctx)
	if err != nil {
		m.logger.Error("failed to connect websocket", zap.Int64("telegram_user_id", user.TelegramUserID), zap.Error(err))
		return
	}
	defer client.Close()

	go func() {
		<-ctx.Done()
		_ = client.Close()
	}()

	if err := client.Subscribe(ctx, assetIDs); err != nil {
		m.logger.Error("failed to subscribe websocket", zap.Int64("telegram_user_id", user.TelegramUserID), zap.Error(err))
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := client.Receive(ctx)
		if err != nil {
			m.logger.Error("websocket receive error", zap.Int64("telegram_user_id", user.TelegramUserID), zap.Error(err))
			return
		}

		if msg == nil || msg.EventType != "price_change" {
			continue
		}

		for _, change := range msg.PriceChanges {
			alertsForAsset, ok := assetAlerts[change.AssetID]
			if !ok {
				continue
			}
			for _, alert := range alertsForAsset {
				price := selectPrice(alert.Comparator, change)
				if price == nil {
					continue
				}
				if shouldNotify(alert.Comparator, *price, alert.Threshold) {
					text := fmt.Sprintf(
						"Alert #%d triggered: %s %s %s %s (price %s)",
						alert.AlertID,
						alert.MarketSlug,
						alert.Outcome,
						alert.Comparator,
						alert.Threshold.String(),
						price.String(),
					)
					if err := m.notifier.Notify(user.TelegramUserID, text); err != nil {
						m.logger.Warn("failed to send alert", zap.Int64("telegram_user_id", user.TelegramUserID), zap.Error(err))
					}
				}
			}
		}
	}
}

func selectPrice(comparator string, change domain.PriceChange) *decimal.Decimal {
	if comparator == "<=" {
		if change.BestAsk != nil {
			return change.BestAsk
		}
	} else {
		if change.BestBid != nil {
			return change.BestBid
		}
	}
	if change.Price != nil {
		return change.Price
	}
	return nil
}

func shouldNotify(comparator string, price decimal.Decimal, threshold decimal.Decimal) bool {
	cmp := price.Cmp(threshold)
	if comparator == "<=" {
		return cmp <= 0
	}
	return cmp >= 0
}
