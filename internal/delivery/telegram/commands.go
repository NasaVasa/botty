package telegram

import (
	"errors"
	"strconv"
	"strings"
)

const HelpText = `Commands:
/start - register
/help - show this help
/event <event_slug>
/add_alert <event_slug> <market_slug> <YES|NO> <=|>=|<|> <threshold>
/alerts - list your alerts
/enable <alert_id>
/disable <alert_id>
/delete <alert_id>

Notes:
- Use < as alias for <=, and > as alias for >=.
- <= alerts compare against best_ask; >= alerts compare against best_bid (fallback to price).
Example:
/event us-strikes-iran-by
/add_alert us-strikes-iran-by us-strikes-iran-by-february-5-2026 YES <= 0.23
`

var ErrInvalidArguments = errors.New("invalid arguments")

func ParseAddAlertArgs(args string) (eventSlug, marketSlug, outcome, comparator, threshold string, err error) {
	parts := strings.Fields(args)
	if len(parts) != 5 {
		return "", "", "", "", "", ErrInvalidArguments
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), strings.TrimSpace(parts[3]), strings.TrimSpace(parts[4]), nil
}

func ParseEventSlug(args string) (string, error) {
	slug := strings.TrimSpace(args)
	if slug == "" {
		return "", ErrInvalidArguments
	}
	return slug, nil
}

func ParseAlertID(args string) (uint, error) {
	idStr := strings.TrimSpace(args)
	if idStr == "" {
		return 0, ErrInvalidArguments
	}
	value, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, ErrInvalidArguments
	}
	return uint(value), nil
}
