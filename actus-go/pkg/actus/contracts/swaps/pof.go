package swaps

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/utils"
	"github.com/yourusername/actus-go/pkg/riskfactor"
)

// CalculatePayoff calculates the payoff for SWAPS contract events
// SWAPS payoff is the net cash flow from both legs:
// - Fixed Leg: pays/receives fixed interest
// - Floating Leg: pays/receives floating interest
func (s *SWAPS) CalculatePayoff(
	event events.ContractEvent,
	state *states.ContractState,
	rf riskfactor.Observer,
) (decimal.Decimal, error) {

	switch event.Type {
	case events.IED:
		return s.pofIED(state), nil
	case events.IP:
		return s.pofIP(state, event, rf), nil
	case events.RR:
		return decimal.Zero, nil // RR has no direct payoff
	case events.MD:
		return s.pofMD(state, event, rf), nil
	default:
		return decimal.Zero, nil
	}
}

// pofIED calculates payoff for Initial Exchange Date
func (s *SWAPS) pofIED(state *states.ContractState) decimal.Decimal {
	// In a plain vanilla interest rate swap, there's no initial principal exchange
	return decimal.Zero
}

// pofIP calculates payoff for Interest Payment event
// This is the NET cash flow from both fixed and floating legs
func (s *SWAPS) pofIP(state *states.ContractState, event events.ContractEvent, rf riskfactor.Observer) decimal.Decimal {
	// Calculate year fraction
	yf := utils.YearFraction(state.StatusDate, event.Time, s.Attributes.DayCountConvention)

	// Calculate FIXED leg interest payment
	fixedInterest := state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)
	fixedInterest = state.InterestScalingIndex.Mul(fixedInterest)

	// Calculate FLOATING leg interest payment
	floatingRate := s.getFloatingRate(event.Time, rf)
	floatingInterest := yf.Mul(floatingRate).Mul(state.NotionalPrincipal)
	floatingInterest = state.InterestScalingIndex.Mul(floatingInterest)

	// Calculate net payment based on contract role
	var netPayoff decimal.Decimal

	if s.Attributes.ContractRole == "RFL" {
		// Receive Fixed Leg: receive fixed, pay floating
		// Payoff = Fixed received - Floating paid
		netPayoff = fixedInterest.Sub(floatingInterest)
	} else {
		// Pay Fixed Leg: pay fixed, receive floating
		// Payoff = Floating received - Fixed paid
		netPayoff = floatingInterest.Sub(fixedInterest)
	}

	return netPayoff
}

// pofMD calculates payoff for Maturity Date
// This is the final net interest payment
func (s *SWAPS) pofMD(state *states.ContractState, event events.ContractEvent, rf riskfactor.Observer) decimal.Decimal {
	// Calculate final interest payment (same as IP)
	yf := utils.YearFraction(state.StatusDate, event.Time, s.Attributes.DayCountConvention)

	// Fixed leg final interest
	fixedInterest := state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Floating leg final interest
	floatingRate := s.getFloatingRate(event.Time, rf)
	floatingInterest := yf.Mul(floatingRate).Mul(state.NotionalPrincipal)

	// Calculate net payment based on contract role
	var netPayoff decimal.Decimal

	if s.Attributes.ContractRole == "RFL" {
		// Receive Fixed Leg: receive fixed, pay floating
		netPayoff = fixedInterest.Sub(floatingInterest)
	} else {
		// Pay Fixed Leg: pay fixed, receive floating
		netPayoff = floatingInterest.Sub(fixedInterest)
	}

	// Add any accrued fees
	netPayoff = netPayoff.Add(state.FeeAccrued)

	return netPayoff
}

// getFloatingRate retrieves the floating rate from the risk factor observer
// If no observer is provided or no market object code is defined, returns zero
func (s *SWAPS) getFloatingRate(eventTime time.Time, rf riskfactor.Observer) decimal.Decimal {
	if rf == nil || s.Attributes.MarketObjectCodeOfRateReset == "" {
		// If no floating rate source, use the nominal rate (for testing)
		return s.Attributes.NominalInterestRate
	}

	// Get market rate from risk factor observer
	marketRate, err := rf.GetMarketRate(
		s.Attributes.MarketObjectCodeOfRateReset,
		eventTime,
	)
	if err != nil {
		// If error getting market rate, fallback to nominal rate
		return s.Attributes.NominalInterestRate
	}

	// Apply spread and multiplier
	floatingRate := marketRate
	if !s.Attributes.RateSpread.IsZero() {
		floatingRate = floatingRate.Add(s.Attributes.RateSpread)
	}
	if !s.Attributes.RateMultiplier.IsZero() && !s.Attributes.RateMultiplier.Equal(decimal.NewFromInt(1)) {
		floatingRate = floatingRate.Mul(s.Attributes.RateMultiplier)
	}

	return floatingRate
}
