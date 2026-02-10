package config

import (
	"context"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	TelegramBotToken  string        `env:"TELEGRAM_BOT_TOKEN,required"`
	DBHost            string        `env:"DB_HOST,required"`
	DBPort            int           `env:"DB_PORT,default=5432"`
	DBUser            string        `env:"DB_USER,required"`
	DBPassword        string        `env:"DB_PASSWORD,required"`
	DBName            string        `env:"DB_NAME,required"`
	DBSSLMode         string        `env:"DB_SSLMODE,default=disable"`
	DBMaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS,default=10"`
	DBMaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS,default=25"`
	DBConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME,default=30m"`

	PolymarketWSURL         string        `env:"POLYMARKET_WS_URL,default=wss://ws-subscriptions-clob.polymarket.com/ws/market"`
	PolymarketGammaBaseURL  string        `env:"POLYMARKET_GAMMA_BASE_URL,default=https://gamma-api.polymarket.com"`
	PolymarketGammaTimeout  time.Duration `env:"POLYMARKET_GAMMA_TIMEOUT,default=10s"`
	PolymarketWSReadTimeout time.Duration `env:"POLYMARKET_WS_READ_TIMEOUT,default=0s"`

	TelegramPollTimeout int    `env:"TELEGRAM_POLL_TIMEOUT,default=60"`
	LogLevel            string `env:"LOG_LEVEL,default=info"`
}

func Load(ctx context.Context) (Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
