package riskfactor

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// TimePoint represents a single data point in a time series
type TimePoint struct {
	Time  time.Time
	Value decimal.Decimal
}

// TimeSeries represents a time series of values
type TimeSeries []TimePoint

// TimeSeriesObserver is a risk factor observer that supports time-varying market data
type TimeSeriesObserver struct {
	marketRates map[string]TimeSeries
	fxRates     map[string]TimeSeries
	prepayments map[string]decimal.Decimal
	defaults    map[string]bool
}

// NewTimeSeriesObserver creates a new time series observer
func NewTimeSeriesObserver() *TimeSeriesObserver {
	return &TimeSeriesObserver{
		marketRates: make(map[string]TimeSeries),
		fxRates:     make(map[string]TimeSeries),
		prepayments: make(map[string]decimal.Decimal),
		defaults:    make(map[string]bool),
	}
}

// SetMarketRateSeries sets a market rate time series
func (tso *TimeSeriesObserver) SetMarketRateSeries(marketObjectCode string, series TimeSeries) {
	tso.marketRates[marketObjectCode] = series
}

// AddMarketRatePoint adds a single point to a market rate time series
func (tso *TimeSeriesObserver) AddMarketRatePoint(marketObjectCode string, t time.Time, rate decimal.Decimal) {
	if _, exists := tso.marketRates[marketObjectCode]; !exists {
		tso.marketRates[marketObjectCode] = TimeSeries{}
	}
	tso.marketRates[marketObjectCode] = append(tso.marketRates[marketObjectCode], TimePoint{
		Time:  t,
		Value: rate,
	})
}

// GetMarketRate retrieves a market rate at a specific time
// Uses the value at or immediately before the requested time
// If no data exists before the requested time, uses the first available data point
func (tso *TimeSeriesObserver) GetMarketRate(marketObjectCode string, t time.Time) (decimal.Decimal, error) {
	series, exists := tso.marketRates[marketObjectCode]
	if !exists || len(series) == 0 {
		return decimal.Zero, fmt.Errorf("no market rate data for: %s", marketObjectCode)
	}

	// Find the most recent data point at or before time t
	var bestPoint *TimePoint
	for i := range series {
		point := &series[i]
		// Use data at or before requested time
		if point.Time.Before(t) || point.Time.Equal(t) {
			if bestPoint == nil || point.Time.After(bestPoint.Time) {
				bestPoint = point
			}
		}
	}

	// If no point found before/at t, use the first available point
	// This handles cases where we need to extrapolate backward
	if bestPoint == nil && len(series) > 0 {
		bestPoint = &series[0]
	}

	if bestPoint == nil {
		return decimal.Zero, fmt.Errorf("no suitable market rate data for %s at %s", marketObjectCode, t)
	}

	return bestPoint.Value, nil
}

// SetFXRateSeries sets an FX rate time series
func (tso *TimeSeriesObserver) SetFXRateSeries(currencyPair string, series TimeSeries) {
	tso.fxRates[currencyPair] = series
}

// AddFXRatePoint adds a single point to an FX rate time series
func (tso *TimeSeriesObserver) AddFXRatePoint(currencyPair string, t time.Time, rate decimal.Decimal) {
	if _, exists := tso.fxRates[currencyPair]; !exists {
		tso.fxRates[currencyPair] = TimeSeries{}
	}
	tso.fxRates[currencyPair] = append(tso.fxRates[currencyPair], TimePoint{
		Time:  t,
		Value: rate,
	})
}

// GetFXRate retrieves an FX rate at a specific time
func (tso *TimeSeriesObserver) GetFXRate(currencyPair string, t time.Time) (decimal.Decimal, error) {
	series, exists := tso.fxRates[currencyPair]
	if !exists || len(series) == 0 {
		// Default to 1:1 if not found
		return decimal.NewFromInt(1), nil
	}

	// Find the most recent data point at or before time t
	var bestPoint *TimePoint
	for i := range series {
		point := &series[i]
		if point.Time.Before(t) || point.Time.Equal(t) {
			if bestPoint == nil || point.Time.After(bestPoint.Time) {
				bestPoint = point
			}
		}
	}

	if bestPoint == nil && len(series) > 0 {
		bestPoint = &series[0]
	}

	if bestPoint == nil {
		return decimal.NewFromInt(1), nil
	}

	return bestPoint.Value, nil
}

// SetPrepayment sets a prepayment amount for a contract
func (tso *TimeSeriesObserver) SetPrepayment(contractID string, amount decimal.Decimal) {
	tso.prepayments[contractID] = amount
}

// ObservePrepayment checks for prepayment events
func (tso *TimeSeriesObserver) ObservePrepayment(contractID string, t time.Time) (decimal.Decimal, error) {
	amount, exists := tso.prepayments[contractID]
	if !exists {
		return decimal.Zero, nil // No prepayment
	}
	return amount, nil
}

// SetDefault sets a default status for a contract
func (tso *TimeSeriesObserver) SetDefault(contractID string, hasDefaulted bool) {
	tso.defaults[contractID] = hasDefaulted
}

// ObserveDefault checks for credit default events
func (tso *TimeSeriesObserver) ObserveDefault(contractID string, t time.Time) (bool, error) {
	defaulted, exists := tso.defaults[contractID]
	if !exists {
		return false, nil // No default
	}
	return defaulted, nil
}

// ObserveEvent is a placeholder for general event observation
func (tso *TimeSeriesObserver) ObserveEvent(contractID string, t time.Time) (interface{}, error) {
	return nil, nil
}
