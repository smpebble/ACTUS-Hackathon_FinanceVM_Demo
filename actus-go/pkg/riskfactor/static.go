package riskfactor

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// StaticObserver is a simple risk factor observer with static values
// Useful for testing and simple scenarios where market data doesn't change
type StaticObserver struct {
	marketRates map[string]decimal.Decimal
	fxRates     map[string]decimal.Decimal
	prepayments map[string]decimal.Decimal
	defaults    map[string]bool
}

// NewStaticObserver creates a new static observer with empty data
func NewStaticObserver() *StaticObserver {
	return &StaticObserver{
		marketRates: make(map[string]decimal.Decimal),
		fxRates:     make(map[string]decimal.Decimal),
		prepayments: make(map[string]decimal.Decimal),
		defaults:    make(map[string]bool),
	}
}

// SetMarketRate sets a market rate
func (so *StaticObserver) SetMarketRate(marketObjectCode string, rate decimal.Decimal) {
	so.marketRates[marketObjectCode] = rate
}

// GetMarketRate retrieves a market rate
func (so *StaticObserver) GetMarketRate(marketObjectCode string, t time.Time) (decimal.Decimal, error) {
	rate, exists := so.marketRates[marketObjectCode]
	if !exists {
		return decimal.Zero, fmt.Errorf("market rate not found for: %s", marketObjectCode)
	}
	return rate, nil
}

// SetFXRate sets a foreign exchange rate
func (so *StaticObserver) SetFXRate(currencyPair string, rate decimal.Decimal) {
	so.fxRates[currencyPair] = rate
}

// GetFXRate retrieves a foreign exchange rate
func (so *StaticObserver) GetFXRate(currencyPair string, t time.Time) (decimal.Decimal, error) {
	rate, exists := so.fxRates[currencyPair]
	if !exists {
		// Default to 1:1 if not found
		return decimal.NewFromInt(1), nil
	}
	return rate, nil
}

// SetPrepayment sets a prepayment amount for a contract
func (so *StaticObserver) SetPrepayment(contractID string, amount decimal.Decimal) {
	so.prepayments[contractID] = amount
}

// ObservePrepayment checks for prepayment events
func (so *StaticObserver) ObservePrepayment(contractID string, t time.Time) (decimal.Decimal, error) {
	amount, exists := so.prepayments[contractID]
	if !exists {
		return decimal.Zero, nil // No prepayment
	}
	return amount, nil
}

// SetDefault sets a default status for a contract
func (so *StaticObserver) SetDefault(contractID string, hasDefaulted bool) {
	so.defaults[contractID] = hasDefaulted
}

// ObserveDefault checks for credit default events
func (so *StaticObserver) ObserveDefault(contractID string, t time.Time) (bool, error) {
	defaulted, exists := so.defaults[contractID]
	if !exists {
		return false, nil // No default
	}
	return defaulted, nil
}
