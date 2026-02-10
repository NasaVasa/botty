package telegram

import (
	"context"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	handlers    *Handlers
	pollTimeout int
}

func NewAPI(token string) (*tgbotapi.BotAPI, error) {
	return tgbotapi.NewBotAPI(token)
}

func NewBot(api *tgbotapi.BotAPI, handlers *Handlers, pollTimeout int) *Bot {
	return &Bot{api: api, handlers: handlers, pollTimeout: pollTimeout}
}

func (b *Bot) Start(ctx context.Context) error {
	config := tgbotapi.NewUpdate(0)
	config.Timeout = b.pollTimeout
	updates := b.api.GetUpdatesChan(config)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			b.handlers.HandleUpdate(ctx, b.api, update)
		}
	}
}

type Notifier struct {
	api    *tgbotapi.BotAPI
	logger *zap.Logger
}

func NewNotifier(api *tgbotapi.BotAPI, logger *zap.Logger) *Notifier {
	return &Notifier{api: api, logger: logger}
}

func (n *Notifier) Notify(telegramUserID int64, text string) error {
	n.logger.Info("telegram notify send", zap.Int64("telegram_user_id", telegramUserID), zap.String("text", text))
	msg := tgbotapi.NewMessage(telegramUserID, text)
	_, err := n.api.Send(msg)
	if err != nil {
		n.logger.Warn("failed to notify", zap.Error(err))
	}
	return err
}
