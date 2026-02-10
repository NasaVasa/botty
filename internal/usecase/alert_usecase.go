package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NasaVasa/botty/internal/domain"
	"github.com/shopspring/decimal"
)

var (
	ErrUserNotRegistered = errors.New("user not registered")
	ErrInvalidOutcome    = errors.New("invalid outcome")
	ErrInvalidComparator = errors.New("invalid comparator")
	ErrInvalidThreshold  = errors.New("invalid threshold")
	ErrAlertNotFound     = errors.New("alert not found")
	ErrEventNotFound     = errors.New("event not found")
	ErrMarketNotInEvent  = errors.New("market not in event")
)

type AlertUsecase struct {
	users  domain.UserRepository
	alerts domain.AlertRepository
	gamma  domain.GammaClient
}

func NewAlertUsecase(users domain.UserRepository, alerts domain.AlertRepository, gamma domain.GammaClient) *AlertUsecase {
	return &AlertUsecase{users: users, alerts: alerts, gamma: gamma}
}

func (u *AlertUsecase) AddAlert(ctx context.Context, telegramUserID int64, eventSlug, marketSlug, outcome, comparator, threshold string) (*domain.Alert, error) {
	user, err := u.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, ErrUserNotRegistered
		}
		return nil, err
	}

	normalizedComparator, err := normalizeComparator(comparator)
	if err != nil {
		return nil, ErrInvalidComparator
	}

	decThreshold, err := decimal.NewFromString(strings.TrimSpace(threshold))
	if err != nil {
		return nil, ErrInvalidThreshold
	}

	event, err := u.gamma.GetEventBySlug(ctx, eventSlug)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	selected, ok := findMarketBySlug(marketSlug, event)
	if !ok {
		return nil, ErrMarketNotInEvent
	}

	assetID, normalizedOutcome, err := mapOutcomeToAssetID(selected, outcome)
	if err != nil {
		return nil, ErrInvalidOutcome
	}

	alert := &domain.Alert{
		UserID:      user.ID,
		MarketSlug:  selected.Slug,
		ConditionID: selected.ConditionID,
		Outcome:     normalizedOutcome,
		AssetID:     assetID,
		Comparator:  normalizedComparator,
		Threshold:   decThreshold.String(),
		Enabled:     true,
	}

	if err := u.alerts.Create(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

func (u *AlertUsecase) ListAlerts(ctx context.Context, telegramUserID int64) ([]domain.Alert, error) {
	user, err := u.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, ErrUserNotRegistered
		}
		return nil, err
	}

	return u.alerts.ListByUser(ctx, user.ID)
}

func (u *AlertUsecase) EnableAlert(ctx context.Context, telegramUserID int64, alertID uint) error {
	return u.setEnabled(ctx, telegramUserID, alertID, true)
}

func (u *AlertUsecase) DisableAlert(ctx context.Context, telegramUserID int64, alertID uint) error {
	return u.setEnabled(ctx, telegramUserID, alertID, false)
}

func (u *AlertUsecase) DeleteAlert(ctx context.Context, telegramUserID int64, alertID uint) error {
	user, err := u.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return ErrUserNotRegistered
		}
		return err
	}

	if err := u.alerts.Delete(ctx, user.ID, alertID); err != nil {
		if err == domain.ErrNotFound {
			return ErrAlertNotFound
		}
		return err
	}

	return nil
}

func (u *AlertUsecase) setEnabled(ctx context.Context, telegramUserID int64, alertID uint, enabled bool) error {
	user, err := u.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return ErrUserNotRegistered
		}
		return err
	}

	if err := u.alerts.SetEnabled(ctx, user.ID, alertID, enabled); err != nil {
		if err == domain.ErrNotFound {
			return ErrAlertNotFound
		}
		return err
	}

	return nil
}

func normalizeComparator(input string) (string, error) {
	switch strings.TrimSpace(input) {
	case "<=", "<":
		return "<=", nil
	case ">=", ">":
		return ">=", nil
	default:
		return "", ErrInvalidComparator
	}
}

func findMarketBySlug(marketSlug string, event *domain.EventMarkets) (domain.MarketInfo, bool) {
	for _, market := range event.Markets {
		if market.Slug == marketSlug {
			return market, true
		}
	}
	return domain.MarketInfo{}, false
}

func mapOutcomeToAssetID(market domain.MarketInfo, outcome string) (string, string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(outcome))
	if normalized != "YES" && normalized != "NO" {
		return "", "", fmt.Errorf("invalid outcome")
	}
	if len(market.ClobTokenIDs) < 2 {
		return "", "", fmt.Errorf("missing token ids")
	}

	if len(market.Outcomes) >= 2 {
		first := strings.ToUpper(market.Outcomes[0])
		second := strings.ToUpper(market.Outcomes[1])
		if first == "YES" && second == "NO" {
			if normalized == "YES" {
				return market.ClobTokenIDs[0], normalized, nil
			}
			return market.ClobTokenIDs[1], normalized, nil
		}
	}

	if normalized == "YES" {
		return market.ClobTokenIDs[0], normalized, nil
	}
	return market.ClobTokenIDs[1], normalized, nil
}
