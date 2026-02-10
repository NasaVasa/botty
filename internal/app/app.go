package app

import (
	"context"

	"github.com/NasaVasa/botty/internal/config"
	"github.com/NasaVasa/botty/internal/delivery/telegram"
	"github.com/NasaVasa/botty/internal/infra/db"
	"github.com/NasaVasa/botty/internal/infra/log"
	"github.com/NasaVasa/botty/internal/infra/polymarket"
	"github.com/NasaVasa/botty/internal/usecase"
	"go.uber.org/zap"
)

type App struct {
	bot       *telegram.Bot
	alerting  *usecase.AlertingManager
	logger    *zap.Logger
	cleanupFn func() error
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger, err := log.NewLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	dbConn, err := db.Open(cfg, logger)
	if err != nil {
		return nil, err
	}

	userRepo := db.NewUserRepository(dbConn)
	alertRepo := db.NewAlertRepository(dbConn)
	gammaClient := polymarket.NewGammaClient(cfg.PolymarketGammaBaseURL, cfg.PolymarketGammaTimeout, logger)
	wsFactory := polymarket.NewWSFactory(cfg.PolymarketWSURL, cfg.PolymarketWSReadTimeout, logger)

	userUC := usecase.NewUserUsecase(userRepo)
	alertUC := usecase.NewAlertUsecase(userRepo, alertRepo, gammaClient)
	eventUC := usecase.NewEventUsecase(gammaClient)

	api, err := telegram.NewAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, err
	}

	notifier := telegram.NewNotifier(api, logger)
	alerting := usecase.NewAlertingManager(userRepo, alertRepo, wsFactory, notifier, logger)
	handlers := telegram.NewHandlers(userUC, alertUC, eventUC, alerting, logger)
	bot := telegram.NewBot(api, handlers, cfg.TelegramPollTimeout)

	cleanup := func() error {
		sqlDB, err := dbConn.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}

	return &App{bot: bot, alerting: alerting, logger: logger, cleanupFn: cleanup}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("botty service starting")
	if err := a.alerting.StartAll(ctx); err != nil {
		a.logger.Warn("failed to start alerting for existing users", zap.Error(err))
	}

	a.logger.Info("botty service started")
	return a.bot.Start(ctx)
}

func (a *App) Shutdown() {
	a.logger.Info("botty service shutting down")
	a.alerting.StopAll()
	if a.cleanupFn != nil {
		if err := a.cleanupFn(); err != nil {
			a.logger.Warn("failed to close database", zap.Error(err))
		}
	}
	_ = a.logger.Sync()
}
