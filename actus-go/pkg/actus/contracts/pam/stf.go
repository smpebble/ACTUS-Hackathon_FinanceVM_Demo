package pam

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/utils"
	"github.com/yourusername/actus-go/pkg/riskfactor"
)

// State Transition Functions (STF) for PAM contract
// These implement the ACTUS specification section 4.x for PAM contracts

// getCalculationTime returns the appropriate time to use for day count calculations
// based on the business day convention
func (p *PAM) getCalculationTime(event events.ContractEvent) time.Time {
	// SCMF/SCF/SCMP/SCP: Shift/Calculate - use payment time (after BD adjustment)
	// CSMF/CSF/CSMP/CSP/NULL: Calculate/Shift - use schedule time (before BD adjustment)
	switch p.Attributes.BusinessDayConvention {
	case "SCMF", "SCF", "SCMP", "SCP":
		return event.Time // Use adjusted date
	default:
		return event.ScheduleTime // Use schedule date
	}
}

// stfAD implements the Analysis Date state transition
// Updates accrued interest from previous state to analysis date
func (p *PAM) stfAD(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	calculationTime := p.getCalculationTime(event)

	// Calculate year fraction from last status date to analysis date
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)

	// Update accrued interest: Ipac+ = Ipac- + Y(Sd-, t) × Ipnr- × Nt-
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Update status date: Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfIED implements the Initial Exchange Date state transition
// This initializes the contract state at the beginning
func (p *PAM) stfIED(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	roleSign := p.getRoleSign()

	// Nt+ = R(CNTRL) × NT
	state.NotionalPrincipal = roleSign.Mul(p.Attributes.NotionalPrincipal)

	// Ipnr+ = IPNR
	if !p.Attributes.NominalInterestRate.IsZero() {
		state.NominalInterestRate = p.Attributes.NominalInterestRate
	}

	// Ipac+ = IPAC (or 0 if not specified)
	if !p.Attributes.AccruedInterest.IsZero() {
		state.AccruedInterest = p.Attributes.AccruedInterest
	} else {
		state.AccruedInterest = decimal.Zero
	}

	// Feac+ = FEAC (or 0 if not specified)
	if !p.Attributes.FeeAccrued.IsZero() {
		state.FeeAccrued = p.Attributes.FeeAccrued
	} else {
		state.FeeAccrued = decimal.Zero
	}

	// Nsc+ = 1, Isc+ = 1 (scaling indices)
	state.NominalScalingIndex = decimal.NewFromInt(1)
	state.InterestScalingIndex = decimal.NewFromInt(1)

	// Sd+ = t
	state.StatusDate = event.ScheduleTime

	return state
}

// stfIP implements the Interest Payment state transition
// Resets accrued interest after payment
func (p *PAM) stfIP(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	calculationTime := p.getCalculationTime(event)

	// Ipac+ = 0 (interest is paid, accrual resets)
	state.AccruedInterest = decimal.Zero

	// Update fee accrual if fee basis is notional
	if p.Attributes.FeeBasis == "N" && !p.Attributes.FeeRate.IsZero() {
		yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)
		state.FeeAccrued = state.FeeAccrued.Add(
			yf.Mul(state.NotionalPrincipal).Mul(p.Attributes.FeeRate),
		)
	}

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfIPCI implements the Interest Payment Capitalization state transition
// Capitalizes accrued interest into principal
func (p *PAM) stfIPCI(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// Calculate year fraction from last status date to capitalization date
	calculationTime := p.getCalculationTime(event)
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)

	// Calculate accrued interest up to this point
	// Ipac = Ipac- + Y(Sd-, t) × Ipnr- × Nt-
	accruedInterest := state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Capitalize interest into principal
	// Nt+ = Nt- + Ipac
	state.NotionalPrincipal = state.NotionalPrincipal.Add(accruedInterest)

	// Reset accrued interest after capitalization
	// Ipac+ = 0
	state.AccruedInterest = decimal.Zero

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfMD implements the Maturity Date state transition
// Finalizes the contract by setting all values to zero
func (p *PAM) stfMD(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	calculationTime := p.getCalculationTime(event)

	// Nt+ = 0 (principal is fully repaid)
	state.NotionalPrincipal = decimal.Zero

	// Ipac+ = 0 (all interest is paid)
	state.AccruedInterest = decimal.Zero

	// Feac+ = 0 (all fees are paid)
	state.FeeAccrued = decimal.Zero

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfRR implements the Rate Reset state transition
// Updates the nominal interest rate based on market observations
func (p *PAM) stfRR(
	state *states.ContractState,
	event events.ContractEvent,
	rf riskfactor.Observer,
) *states.ContractState {
	// Update accrued interest up to reset date
	calculationTime := p.getCalculationTime(event)
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Get new market rate
	if rf != nil && p.Attributes.MarketObjectCodeOfRateReset != "" {
		marketRate, err := rf.GetMarketRate(p.Attributes.MarketObjectCodeOfRateReset, event.Time)
		if err == nil {
			// IMPORTANT: Save the previous rate before updating
			// This is needed for ANN contracts to correctly calculate annuity payments
			// when the rate reset occurs between the last IP and next PR dates
			state.PreviousNominalInterestRate = state.NominalInterestRate
			state.LastRateResetDate = calculationTime

			// Ipnr+ = RRMLT × O_rf(RRMO, t) + RRSP
			// New rate = multiplier × market rate + spread
			rateMultiplier := p.Attributes.RateMultiplier
			if rateMultiplier.IsZero() {
				rateMultiplier = decimal.NewFromInt(1)
			}

			newRate := rateMultiplier.Mul(marketRate).Add(p.Attributes.RateSpread)
			state.NominalInterestRate = newRate
		}
	}

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfRRF implements the Rate Reset Fixed Leg state transition
// Updates the nominal interest rate to a fixed rate (NextResetRate)
// This is used for the first rate reset event when a fixed rate is specified
func (p *PAM) stfRRF(
	state *states.ContractState,
	event events.ContractEvent,
	rf riskfactor.Observer,
) *states.ContractState {
	// Update accrued interest up to reset date
	calculationTime := p.getCalculationTime(event)
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Set new rate to the fixed NextResetRate (no market observation needed)
	if !p.Attributes.NextResetRate.IsZero() {
		// IMPORTANT: Save the previous rate before updating
		// This is needed for ANN contracts to correctly calculate annuity payments
		state.PreviousNominalInterestRate = state.NominalInterestRate
		state.LastRateResetDate = calculationTime

		state.NominalInterestRate = p.Attributes.NextResetRate
	}

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfFP implements the Fee Payment state transition
// Resets fee accrual after payment
func (p *PAM) stfFP(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// Update accrued interest
	calculationTime := p.getCalculationTime(event)
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Feac+ = 0 (fee is paid, accrual resets)
	state.FeeAccrued = decimal.Zero

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfPP implements the Principal Prepayment state transition
// Reduces the notional principal by the prepayment amount
func (p *PAM) stfPP(
	state *states.ContractState,
	event events.ContractEvent,
	rf riskfactor.Observer,
) *states.ContractState {
	// Update accrued interest
	calculationTime := p.getCalculationTime(event)
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Observe prepayment amount
	if rf != nil {
		ppAmount, err := rf.ObservePrepayment(p.Attributes.ContractID, calculationTime)
		if err == nil && !ppAmount.IsZero() {
			// Nt+ = Nt- - PP amount
			state.NotionalPrincipal = state.NotionalPrincipal.Sub(ppAmount)
		}
	}

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}

// stfPRD implements the State Transition Function for Purchase (PRD) event
// STF_PRD_PAM: No state changes
// Purchase pays accrued interest in POF but does NOT update state
// This allows subsequent IP events to calculate from original StatusDate
func (p *PAM) stfPRD(
	state *states.ContractState,
	event events.ContractEvent,
) *states.ContractState {
	// PRD does not change contract state
	// Accrued interest is paid in POF but state variables remain unchanged
	// This ensures subsequent IP events calculate from the correct base date
	return state
}

// stfTD implements the State Transition Function for Termination (TD) event
// STF_TD_PAM: Sd+ = t, Nt+ = 0, Ipac+ = 0
// Termination ends the contract
func (p *PAM) stfTD(
	state *states.ContractState,
	event events.ContractEvent,
) *states.ContractState {
	calculationTime := p.getCalculationTime(event)

	// Sd+ = t
	state.StatusDate = calculationTime

	// Contract is terminated, so reset state variables
	state.NotionalPrincipal = decimal.Zero
	state.AccruedInterest = decimal.Zero
	state.FeeAccrued = decimal.Zero

	return state
}

// stfSC implements the State Transition Function for Scaling Index (SC) event
// STF_SC_PAM: Updates Nsc and/or Isc based on ScalingEffect
// SC is a state update event that adjusts scaling indices based on market data
func (p *PAM) stfSC(
	state *states.ContractState,
	event events.ContractEvent,
	rf riskfactor.Observer,
) *states.ContractState {
	calculationTime := p.getCalculationTime(event)

	// Update accrued interest before changing scaling indices
	// This is critical as SC events mark a point in time for index fixing
	yf := utils.YearFraction(state.StatusDate, calculationTime, p.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Get scaling index from risk factor observer
	if rf != nil && p.Attributes.MarketObjectCodeOfScalingIndex != "" {
		currentIndex, err := rf.GetMarketRate(p.Attributes.MarketObjectCodeOfScalingIndex, calculationTime)
		if err == nil && !currentIndex.IsZero() {
			// Calculate scaling multiplier as ratio: current index / initial index
			// Initial index comes from contract attributes (base value, typically 100)
			initialIndex := p.Attributes.ScalingIndexAtStatusDate
			if initialIndex.IsZero() {
				initialIndex = decimal.NewFromInt(1) // Default to 1 if not set
			}

			// Scaling multiplier = current index / initial index
			// E.g., if initial=100 and current=200, multiplier=2.0
			scalingMultiplier := currentIndex.Div(initialIndex)

			// Update scaling indices based on ScalingEffect
			// ScalingEffect determines which scaling indices to update
			switch p.Attributes.ScalingEffect {
			case "OOO", "ONO", "OOM", "ONM":
				// "O*O" = Update nominal (principal) scaling index
				state.NominalScalingIndex = scalingMultiplier
			case "IOO", "INO", "IOM", "INM":
				// "I*O" = Update interest scaling index
				state.InterestScalingIndex = scalingMultiplier
			}
		}
	}

	// Sd+ = t
	state.StatusDate = calculationTime

	return state
}
