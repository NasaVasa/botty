package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NasaVasa/botty/internal/domain"
	"github.com/NasaVasa/botty/internal/usecase"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type Handlers struct {
	userUC   *usecase.UserUsecase
	alertUC  *usecase.AlertUsecase
	eventUC  *usecase.EventUsecase
	alerting *usecase.AlertingManager
	logger   *zap.Logger
}

func NewHandlers(userUC *usecase.UserUsecase, alertUC *usecase.AlertUsecase, eventUC *usecase.EventUsecase, alerting *usecase.AlertingManager, logger *zap.Logger) *Handlers {
	return &Handlers{userUC: userUC, alertUC: alertUC, eventUC: eventUC, alerting: alerting, logger: logger}
}

func (h *Handlers) HandleUpdate(ctx context.Context, api *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	if update.Message.From == nil {
		return
	}
	if update.Message.IsCommand() {
		h.handleCommand(ctx, api, update)
		return
	}
}

func (h *Handlers) handleCommand(ctx context.Context, api *tgbotapi.BotAPI, update tgbotapi.Update) {
	command := update.Message.Command()
	args := update.Message.CommandArguments()
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	username := update.Message.From.UserName

	h.logger.Info(
		"telegram command received",
		zap.Int64("chat_id", chatID),
		zap.Int64("telegram_user_id", userID),
		zap.String("username", username),
		zap.String("command", command),
		zap.String("args", args),
	)

	switch command {
	case "start":
		_, err := h.userUC.StartOrGetUser(ctx, userID, username)
		if err != nil {
			h.logger.Warn("start command failed", zap.Int64("telegram_user_id", userID), zap.Error(err))
			h.reply(api, chatID, "Failed to register. Please try again.")
			return
		}
		h.logger.Info("start command complete", zap.Int64("telegram_user_id", userID))
		h.reply(api, chatID, "Welcome to Botty.\n\n"+HelpText)
	case "help":
		h.logger.Info("help command complete", zap.Int64("telegram_user_id", userID))
		h.reply(api, chatID, HelpText)
	case "event":
		eventSlug, err := ParseEventSlug(args)
		if err != nil {
			h.reply(api, chatID, "Usage: /event <event_slug>")
			return
		}
		event, err := h.eventUC.GetEvent(ctx, eventSlug)
		if err != nil {
			h.reply(api, chatID, h.alertErrorMessage(err))
			return
		}
		h.reply(api, chatID, formatEventSummary(eventSlug, event))
	case "add_alert":
		eventSlug, marketSlug, outcome, comparator, threshold, err := ParseAddAlertArgs(args)
		if err != nil {
			h.logger.Warn("add_alert invalid args", zap.Int64("telegram_user_id", userID), zap.String("args", args))
			h.reply(api, chatID, "Usage: /add_alert <event_slug> <market_slug> <YES|NO> <=|>= <threshold>")
			return
		}
		alert, err := h.alertUC.AddAlert(ctx, userID, eventSlug, marketSlug, outcome, comparator, threshold)
		if err != nil {
			h.logger.Warn("add_alert failed", zap.Int64("telegram_user_id", userID), zap.Error(err))
			h.reply(api, chatID, h.alertErrorMessage(err))
			return
		}
		h.logger.Info("add_alert complete", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alert.ID))
		h.alerting.RestartUser(ctx, userID)
		h.reply(api, chatID, fmt.Sprintf("Alert created: #%d %s %s %s %s", alert.ID, alert.MarketSlug, alert.Outcome, alert.Comparator, alert.Threshold))
	case "alerts":
		alerts, err := h.alertUC.ListAlerts(ctx, userID)
		if err != nil {
			h.logger.Warn("alerts list failed", zap.Int64("telegram_user_id", userID), zap.Error(err))
			h.reply(api, chatID, h.alertErrorMessage(err))
			return
		}
		if len(alerts) == 0 {
			h.logger.Info("alerts list empty", zap.Int64("telegram_user_id", userID))
			h.reply(api, chatID, "No alerts yet. Use /add_alert to create one.")
			return
		}
		h.logger.Info("alerts list complete", zap.Int64("telegram_user_id", userID), zap.Int("count", len(alerts)))
		var builder strings.Builder
		builder.WriteString("Your alerts:\n")
		for _, alert := range alerts {
			status := "disabled"
			if alert.Enabled {
				status = "enabled"
			}
			builder.WriteString(fmt.Sprintf("#%d [%s] %s %s %s %s\n", alert.ID, status, alert.MarketSlug, alert.Outcome, alert.Comparator, alert.Threshold))
		}
		h.reply(api, chatID, builder.String())
	case "enable":
		alertID, err := ParseAlertID(args)
		if err != nil {
			h.logger.Warn("enable invalid args", zap.Int64("telegram_user_id", userID), zap.String("args", args))
			h.reply(api, chatID, "Usage: /enable <alert_id>")
			return
		}
		if err := h.alertUC.EnableAlert(ctx, userID, alertID); err != nil {
			h.logger.Warn("enable failed", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alertID), zap.Error(err))
			h.reply(api, chatID, h.alertErrorMessage(err))
			return
		}
		h.logger.Info("enable complete", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alertID))
		h.alerting.RestartUser(ctx, userID)
		h.reply(api, chatID, fmt.Sprintf("Alert #%d enabled.", alertID))
	case "disable":
		alertID, err := ParseAlertID(args)
		if err != nil {
			h.logger.Warn("disable invalid args", zap.Int64("telegram_user_id", userID), zap.String("args", args))
			h.reply(api, chatID, "Usage: /disable <alert_id>")
			return
		}
		if err := h.alertUC.DisableAlert(ctx, userID, alertID); err != nil {
			h.logger.Warn("disable failed", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alertID), zap.Error(err))
			h.reply(api, chatID, h.alertErrorMessage(err))
			return
		}
		h.logger.Info("disable complete", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alertID))
		h.alerting.RestartUser(ctx, userID)
		h.reply(api, chatID, fmt.Sprintf("Alert #%d disabled.", alertID))
	case "delete":
		alertID, err := ParseAlertID(args)
		if err != nil {
			h.logger.Warn("delete invalid args", zap.Int64("telegram_user_id", userID), zap.String("args", args))
			h.reply(api, chatID, "Usage: /delete <alert_id>")
			return
		}
		if err := h.alertUC.DeleteAlert(ctx, userID, alertID); err != nil {
			h.logger.Warn("delete failed", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alertID), zap.Error(err))
			h.reply(api, chatID, h.alertErrorMessage(err))
			return
		}
		h.logger.Info("delete complete", zap.Int64("telegram_user_id", userID), zap.Uint("alert_id", alertID))
		h.alerting.RestartUser(ctx, userID)
		h.reply(api, chatID, fmt.Sprintf("Alert #%d deleted.", alertID))
	default:
		h.logger.Warn("unknown command", zap.Int64("telegram_user_id", userID), zap.String("command", command))
		h.reply(api, chatID, "Unknown command.\n\n"+HelpText)
	}
}

func (h *Handlers) alertErrorMessage(err error) string {
	switch {
	case errors.Is(err, usecase.ErrUserNotRegistered):
		return "Please /start to register first."
	case errors.Is(err, usecase.ErrInvalidOutcome):
		return "Invalid outcome. Use YES or NO."
	case errors.Is(err, usecase.ErrInvalidComparator):
		return "Invalid comparator. Use <=, >=, <, or >."
	case errors.Is(err, usecase.ErrInvalidThreshold):
		return "Invalid threshold. Use a decimal like 0.23."
	case errors.Is(err, usecase.ErrAlertNotFound):
		return "Alert not found."
	case errors.Is(err, usecase.ErrEventNotFound):
		return "Event not found. Ensure the slug is correct."
	case errors.Is(err, usecase.ErrMarketNotInEvent):
		return "Market not found in that event. Use /event <event_slug> to list markets."
	}

	h.logger.Warn("unhandled error", zap.Error(err))
	return "Something went wrong. Please try again."
}

func formatEventSummary(requestedSlug string, event *domain.EventMarkets) string {
	const maxMessageLen = 3800

	eventSlug := event.EventSlug
	if eventSlug == "" {
		eventSlug = requestedSlug
	}

	header := fmt.Sprintf("Event: %s\nMarkets:\n", eventSlug)
	var builder strings.Builder
	builder.WriteString(header)
	remaining := 0
	for i, market := range event.Markets {
		block := formatMarketBlock(i+1, market)
		if builder.Len()+len(block) > maxMessageLen {
			remaining = len(event.Markets) - i
			break
		}
		builder.WriteString(block)
	}

	if remaining > 0 {
		builder.WriteString(fmt.Sprintf("...and %d more markets", remaining))
	}

	if builder.Len() == len(header) {
		builder.WriteString("(no markets)")
	}

	return builder.String()
}
func formatMarketBlock(index int, market domain.MarketInfo) string {
	priceSummary := formatPriceSummary(market)
	question := strings.TrimSpace(market.Question)
	if question != "" {
		question = strings.ReplaceAll(question, "\n", " ")
		if len(question) > 80 {
			question = question[:77] + "..."
		}
		return fmt.Sprintf("%d) %s\n%s\n%s\n\n", index, market.Slug, question, priceSummary)
	}
	return fmt.Sprintf("%d) %s\n%s\n\n", index, market.Slug, priceSummary)
}

func formatPriceSummary(market domain.MarketInfo) string {
	if len(market.OutcomePrices) >= 2 {
		return fmt.Sprintf("Price: YES %s$ NO %s$", market.OutcomePrices[0], market.OutcomePrices[1])
	}
	bid := "N/A"
	ask := "N/A"
	if market.BestBid != nil {
		bid = market.BestBid.String()
	}
	if market.BestAsk != nil {
		ask = market.BestAsk.String()
	}
	return fmt.Sprintf("Price: bid %s ask %s", bid, ask)
}

func (h *Handlers) reply(api *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := api.Send(msg); err != nil {
		h.logger.Warn("failed to send message", zap.Error(err))
	}
}
