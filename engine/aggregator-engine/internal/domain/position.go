package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type AssetClass string
type OptionType string

const (
	AssetClassEquity AssetClass = "EQUITY"
	AssetClassOption AssetClass = "OPTION"
	OptionTypeCall   OptionType = "CALL"
	OptionTypePut    OptionType = "PUT"
)

type OptionDetails struct {
	UnderlyingSymbol string     `json:"underlying_symbol"`
	ExpirationDate   time.Time  `json:"expiration_date"`
	OptionType       OptionType `json:"option_type"`
	StrikePrice      float64    `json:"strike_price"`
}

// Position is the unified representation across any broker integration.
type Position struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Broker          BrokerProvider `json:"broker"`
	AccountID       string         `json:"account_id"`
	Symbol          string         `json:"symbol"`
	AssetClass      AssetClass     `json:"asset_class"`
	Quantity        float64        `json:"quantity"`
	CostBasis       float64        `json:"cost_basis"`
	MarketValue     float64        `json:"market_value"`
	UnrealizedPL    float64        `json:"unrealized_pl"`
	OptionData      *OptionDetails `json:"option_data,omitempty"`
	LastSyncedAt    time.Time      `json:"last_synced_at"`
	InstitutionName string         `json:"institution_name"`
}

// ParseOCC extracts standard 21-character OCC option symbols (e.g., "AMZN261218C00245000")
func ParseOCC(symbol string) (*OptionDetails, error) {
	trimmed := strings.TrimSpace(symbol)
	if len(trimmed) < 18 {
		return nil, fmt.Errorf("invalid OCC symbol length for %s", symbol)
	}

	n := len(trimmed)
	strikeStr := trimmed[n-8:]
	typeChar := trimmed[n-9 : n-8]
	dateStr := trimmed[n-15 : n-9]
	rootSymbol := strings.TrimSpace(trimmed[:n-15])

	expTime, err := time.Parse("060102", dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OCC date '%s': %w", dateStr, err)
	}

	var optType OptionType
	switch strings.ToUpper(typeChar) {
	case "C":
		optType = OptionTypeCall
	case "P":
		optType = OptionTypePut
	default:
		return nil, fmt.Errorf("invalid OCC type flag '%s'", typeChar)
	}

	strikeRaw, err := strconv.ParseFloat(strikeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OCC strike '%s': %w", strikeStr, err)
	}

	return &OptionDetails{
		UnderlyingSymbol: rootSymbol,
		ExpirationDate:   expTime,
		OptionType:       optType,
		StrikePrice:      strikeRaw / 1000.0,
	}, nil
}
