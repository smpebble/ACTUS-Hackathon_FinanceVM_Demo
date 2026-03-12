package riskfactor

import (
	"time"

	"github.com/shopspring/decimal"
)

// Observer is the interface for observing risk factors (market data, events)
// This follows the ACTUS specification for risk factor observation
type Observer interface {
	// GetMarketRate retrieves the market rate for a given market object at a specific time
	// marketObjectCode: identifier for the rate (e.g., "LIBOR3M", "EURIBOR6M")
	// t: the observation time
	GetMarketRate(marketObjectCode string, t time.Time) (decimal.Decimal, error)

	// GetFXRate retrieves the foreign exchange rate for a currency pair
	// currencyPair: format "BASE/QUOTE" (e.g., "EUR/USD")
	// t: the observation time
	GetFXRate(currencyPair string, t time.Time) (decimal.Decimal, error)

	// ObservePrepayment checks for prepayment events
	// contractID: the contract identifier
	// t: the observation time
	// Returns the prepayment amount (zero if no prepayment)
	ObservePrepayment(contractID string, t time.Time) (decimal.Decimal, error)

	// ObserveDefault checks for credit default events
	// contractID: the contract identifier
	// t: the observation time
	// Returns true if default occurred
	ObserveDefault(contractID string, t time.Time) (bool, error)
}
