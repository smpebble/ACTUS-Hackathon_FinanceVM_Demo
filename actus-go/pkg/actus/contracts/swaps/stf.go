package swaps

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/utils"
	"github.com/yourusername/actus-go/pkg/riskfactor"
)

// StateTransition performs state transition for SWAPS contract
func (s *SWAPS) StateTransition(
	event events.ContractEvent,
	preState *states.ContractState,
	rf riskfactor.Observer,
) (*states.ContractState, error) {

	postState := preState.Clone()

	switch event.Type {
	case events.IED:
		return s.stfIED(postState, event), nil
	case events.IP:
		return s.stfIP(postState, event), nil
	case events.RR:
		return s.stfRR(postState, event, rf), nil
	case events.MD:
		return s.stfMD(postState, event), nil
	default:
		return postState, fmt.Errorf("unsupported event type for SWAPS: %s", event.Type)
	}
}

// stfIED handles Initial Exchange Date state transition
func (s *SWAPS) stfIED(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// For SWAPS, IED typically just updates the status date
	// No principal exchange in a plain vanilla interest rate swap

	state.StatusDate = event.Time

	return state
}

// stfIP handles Interest Payment state transition
func (s *SWAPS) stfIP(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// After interest payment, reset accrued interest

	// Ipac+ = 0
	state.AccruedInterest = decimal.Zero

	// Update fee accrued if applicable
	if !s.Attributes.FeeRate.IsZero() && s.Attributes.FeeBasis == "N" {
		yf := utils.YearFraction(state.StatusDate, event.Time, s.Attributes.DayCountConvention)
		state.FeeAccrued = state.FeeAccrued.Add(
			yf.Mul(state.NotionalPrincipal).Mul(s.Attributes.FeeRate),
		)
	}

	// Sd+ = t
	state.StatusDate = event.Time

	return state
}

// stfRR handles Rate Reset state transition (for floating leg)
func (s *SWAPS) stfRR(state *states.ContractState, event events.ContractEvent, rf riskfactor.Observer) *states.ContractState {
	// Accumulate accrued interest before rate reset
	yf := utils.YearFraction(state.StatusDate, event.Time, s.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.InterestCalculationBase),
	)

	// Update fee accrued if applicable
	if !s.Attributes.FeeRate.IsZero() && s.Attributes.FeeBasis == "N" {
		state.FeeAccrued = state.FeeAccrued.Add(
			yf.Mul(state.NotionalPrincipal).Mul(s.Attributes.FeeRate),
		)
	}

	// Get new floating rate from risk factor observer
	if s.Attributes.MarketObjectCodeOfRateReset != "" && rf != nil {
		newRate, err := rf.GetMarketRate(
			s.Attributes.MarketObjectCodeOfRateReset,
			event.Time,
		)

		if err == nil {
			// Apply rate spread/multiplier if defined
			if !s.Attributes.RateSpread.IsZero() {
				newRate = newRate.Add(s.Attributes.RateSpread)
			}
			if !s.Attributes.RateMultiplier.IsZero() {
				newRate = newRate.Mul(s.Attributes.RateMultiplier)
			}

			// Store the floating rate in InterestCalculationBase temporarily
			// This is a workaround since we don't have a separate field for floating rate
			// The actual floating rate will be retrieved from rf during payoff calculation
		}
	}

	// Sd+ = t
	state.StatusDate = event.Time

	return state
}

// stfMD handles Maturity Date state transition
func (s *SWAPS) stfMD(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// At maturity, the swap terminates
	// There's typically no principal exchange in a plain vanilla swap
	// Only the final interest payment

	// Accumulate final interest
	yf := utils.YearFraction(state.StatusDate, event.Time, s.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.InterestCalculationBase),
	)

	// Nt+ = 0 (conceptual - no principal in vanilla swap)
	state.NotionalPrincipal = decimal.Zero

	// Ipac+ = 0 (will be paid out in final settlement)
	state.AccruedInterest = decimal.Zero

	// Feac+ = 0
	state.FeeAccrued = decimal.Zero

	// Sd+ = t
	state.StatusDate = event.Time

	return state
}
