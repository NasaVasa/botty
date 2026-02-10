package polymarket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type gammaEventResponse struct {
	Slug    string        `json:"slug"`
	Markets []gammaMarket `json:"markets"`
}

type gammaMarket struct {
	Slug           string          `json:"slug"`
	ConditionID    string          `json:"conditionId"`
	Outcomes       StringList      `json:"outcomes"`
	ClobTokenIDs   StringList      `json:"clobTokenIds"`
	Question       string          `json:"question"`
	OutcomePrices  StringList      `json:"outcomePrices"`
	BestBid        NullableDecimal `json:"bestBid"`
	BestAsk        NullableDecimal `json:"bestAsk"`
	LastTradePrice NullableDecimal `json:"lastTradePrice"`
}

type wsMessage struct {
	EventType    string          `json:"event_type"`
	PriceChanges []wsPriceChange `json:"price_changes"`
}

type wsPriceChange struct {
	AssetID string          `json:"asset_id"`
	BestBid NullableDecimal `json:"best_bid"`
	BestAsk NullableDecimal `json:"best_ask"`
	Price   NullableDecimal `json:"price"`
}

type NullableDecimal struct {
	Decimal decimal.Decimal
	Valid   bool
}

type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*s = nil
		return nil
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		*s = nil
		return nil
	}
	if trimmed[0] == '"' {
		var inner string
		if err := json.Unmarshal(data, &inner); err != nil {
			return err
		}
		inner = strings.TrimSpace(inner)
		if inner == "" {
			*s = nil
			return nil
		}
		var values []string
		if err := json.Unmarshal([]byte(inner), &values); err == nil {
			*s = values
			return nil
		}
		*s = []string{inner}
		return nil
	}

	if trimmed[0] == '[' {
		var values []string
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		*s = values
		return nil
	}

	return fmt.Errorf("unexpected string list format: %s", trimmed)
}

func (n *NullableDecimal) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		n.Valid = false
		return nil
	}
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		n.Valid = false
		return nil
	}
	if trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		trimmed = strings.Trim(trimmed, "\"")
	}
	dec, err := decimal.NewFromString(trimmed)
	if err != nil {
		n.Valid = false
		return err
	}
	n.Decimal = dec
	n.Valid = true
	return nil
}

func (n NullableDecimal) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Decimal.String())
}
