package domain

import "time"

type BrokerProvider string

const (
	BrokerSchwab  BrokerProvider = "SCHWAB"
	BrokerIBKR    BrokerProvider = "IBKR"
	BrokerTradier BrokerProvider = "TRADIER"
)

// BrokerAccount ties a specific broker instance to a user (tenant).
type BrokerAccount struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`    // Platform tenant ID
	Broker    BrokerProvider `json:"broker"`     // Discriminator
	AccountID string         `json:"account_id"` // Broker's native account number
	IsActive  bool           `json:"is_active"`
	SyncedAt  time.Time      `json:"synced_at"`
}

// AccountBalance represents aggregated liquidity for a specific broker account.
type AccountBalance struct {
	UserID            string         `json:"user_id"`
	Broker            BrokerProvider `json:"broker"`
	AccountID         string         `json:"account_id"`
	TotalEquity       float64        `json:"total_equity"`
	TotalCash         float64        `json:"total_cash"`
	OptionBuyingPower float64        `json:"option_buying_power"`
	StockBuyingPower  float64        `json:"stock_buying_power"`
	UpdatedAt         time.Time      `json:"updated_at"`
}
