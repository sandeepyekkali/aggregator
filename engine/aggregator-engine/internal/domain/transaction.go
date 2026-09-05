package domain

// Transaction represents a single historical event (buy, sell, dividend, fee) in a brokerage account.
type Transaction struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	Broker          string  `json:"broker"` // NEW: Satisfies FK constraint and isolates provider data
	AccountID       string  `json:"account_id"`
	Symbol          string  `json:"symbol"`
	Date            string  `json:"date"` // Format: YYYY-MM-DD
	Name            string  `json:"name"`
	Quantity        float64 `json:"quantity"`
	Price           float64 `json:"price"`
	Amount          float64 `json:"amount"`
	Type            string  `json:"type"`
	InstitutionName string  `json:"institution_name,omitempty"`
}
