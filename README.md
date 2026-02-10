# Botty

Botty — Telegram-бот MVP для ценовых алертов на Polymarket по событию и рынку.

## Архитектура
- **Clean Architecture** слои:
- `internal/domain`: сущности и интерфейсы (без Telegram/GORM/WS).
- `internal/usecase`: прикладная логика (users, alerts, alerting, events).
- `internal/delivery/telegram`: парсинг команд Telegram и ответы.
- `internal/infra`: PostgreSQL (GORM), клиенты Polymarket, логирование, конфиг.
- `internal/app`: композиция зависимостей и жизненный цикл.

Поток работы (кратко):
- `/event <event_slug>` вызывает Gamma и выводит рынки события.
- `/add_alert <event_slug> <market_slug> ...` вызывает Gamma, находит token id, сохраняет алерт и перезапускает alerting для пользователя.
- По одному WebSocket на пользователя подписывается на token id активных алертов.
- Обрабатывается только `event_type == "price_change"`; при выполнении условия отправляется сообщение в Telegram.

Хранилище:
- Только Users и Alerts (soft-delete через GORM).
- Alerts содержат `market_slug`, `condition_id`, `asset_id` и правило — достаточно для работы WS без повторных запросов в Gamma.

## Переменные окружения
Обязательные:
- `TELEGRAM_BOT_TOKEN`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

Опциональные (значения по умолчанию в скобках):
- `DB_SSLMODE` (`disable`)
- `DB_MAX_IDLE_CONNS` (`10`)
- `DB_MAX_OPEN_CONNS` (`25`)
- `DB_CONN_MAX_LIFETIME` (`30m`)
- `POLYMARKET_WS_URL` (`wss://ws-subscriptions-clob.polymarket.com/ws/market`)
- `POLYMARKET_GAMMA_BASE_URL` (`https://gamma-api.polymarket.com`)
- `POLYMARKET_GAMMA_TIMEOUT` (`10s`)
- `POLYMARKET_WS_READ_TIMEOUT` (`0s`)
- `TELEGRAM_POLL_TIMEOUT` (`60`)
- `LOG_LEVEL` (`info`)

## Установка
### Docker Compose
1. Укажите `TELEGRAM_BOT_TOKEN` в `.env`.
2. Запустите:

```bash
docker compose up --build
```

### Локальный запуск
1. Установите Go 1.25+ и PostgreSQL 16.
2. Экспортируйте обязательные переменные окружения (или создайте `.env`).
3. Запустите:

```bash
go run ./cmd/botty
```

## Команды
```
/start
/help
/event <event_slug>
/add_alert <event_slug> <market_slug> <YES|NO> <=|>= <threshold>
/alerts
/enable <alert_id>
/disable <alert_id>
/delete <alert_id>
```

Пример:
```
/event us-strikes-iran-by
/add_alert us-strikes-iran-by us-strikes-iran-by-june-30-2026-699-664-723-485-753-218-567-164-387-443-377-384-159-973-494-631-694-956-361-443-224-518-537-678-486-386-275-153-976-862-149 YES >= 0.5
```

Примечания:
- Gamma `GET /events/slug/{slug}` принимает **event slug**. Используйте `/event`, чтобы получить список рынков и выбрать `market_slug`.

## Логика сравнения цены
- Для `<=` сравнение идет с `best_ask`.
- Для `>=` сравнение идет с `best_bid`.

## Внешние API
Polymarket Gamma (HTTP):
- `GET https://gamma-api.polymarket.com/events/slug/{event_slug}`

Polymarket CLOB WS:
- `wss://ws-subscriptions-clob.polymarket.com/ws/market`
- Сообщение подписки:

```json
{"type":"market","assets_ids":["<tokenId>","<tokenId>"]}
```

Telegram Bot API (через `tgbotapi`):
- Long polling `getUpdates`.
- Отправка сообщений `sendMessage`.
