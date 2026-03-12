package ann

import (
	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/utils"
)

// State Transition Functions (STF) for ANN contract
// ANN differs from LAM in that the total payment (principal + interest) is constant
// The principal portion increases over time as interest decreases

// stfIED implements the Initial Exchange Date state transition for ANN
// This is where we calculate the annuity payment amount if not specified
func (a *ANN) stfIED(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	roleSign := a.getRoleSign()

	// Set notional principal
	state.NotionalPrincipal = roleSign.Mul(a.Attributes.NotionalPrincipal)

	// Set nominal interest rate
	if !a.Attributes.NominalInterestRate.IsZero() {
		state.NominalInterestRate = a.Attributes.NominalInterestRate
	}

	// Set accrued interest
	if !a.Attributes.AccruedInterest.IsZero() {
		state.AccruedInterest = a.Attributes.AccruedInterest
	} else {
		state.AccruedInterest = decimal.Zero
	}

	// Set fee accrued
	if !a.Attributes.FeeAccrued.IsZero() {
		state.FeeAccrued = a.Attributes.FeeAccrued
	} else {
		state.FeeAccrued = decimal.Zero
	}

	// Set scaling indices
	state.NominalScalingIndex = decimal.NewFromInt(1)
	state.InterestScalingIndex = decimal.NewFromInt(1)

	// ANN-specific: Calculate annuity payment if not specified
	// NextPrincipalPayment is always stored as a POSITIVE amount (magnitude)
	// The roleSign is applied in POF functions to determine direction
	if a.Attributes.NextPrincipalRedemptionPayment.IsZero() {
		state.NextPrincipalPayment = a.calculateAnnuityPayment(state)
	} else {
		state.NextPrincipalPayment = a.Attributes.NextPrincipalRedemptionPayment.Abs()
	}

	// Set interest calculation base
	if a.Attributes.InterestCalculationBase == "NTIED" && !a.Attributes.InterestCalculationBaseAmount.IsZero() {
		state.InterestCalculationBase = roleSign.Mul(a.Attributes.InterestCalculationBaseAmount)
	} else {
		// Default: use current notional principal
		state.InterestCalculationBase = state.NotionalPrincipal
	}

	// Update status date
	state.StatusDate = event.Time

	return state
}

// stfPRF implements the Principal Redemption Fixing state transition for ANN
// PRF updates accrued interest and potentially recalculates the annuity payment
// Recalculation only happens at PRF events that occur after Rate Reset (RR) events
// These PRF events have a higher EventOrder (PRFAfterRR = 12) than regular PRF events
func (a *ANN) stfPRF(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// Calculate year fraction from last status date to PRF date
	yf := utils.YearFraction(state.StatusDate, event.Time, a.Attributes.DayCountConvention)

	// Update accrued interest
	// Ipac+ = Ipac- + Y(Sd-, t) × Ipnr- × Nt-
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal),
	)

	// Only recalculate if this is a PRF event after Rate Reset
	// These PRF events have EventOrder = PRFAfterRR (12), not regular PRF (2)
	if event.EventOrder == events.PRFAfterRR {
		// Rate has changed (due to RR), recalculate the annuity payment
		state.NextPrincipalPayment = a.recalculateAnnuityPayment(state, event.Time)
	}

	// Update status date
	state.StatusDate = event.Time

	return state
}

// stfPR implements the Principal Redemption state transition for ANN
// For ANN, the principal payment amount varies based on the interest accrued
// Formula: Principal payment = Total payment - Interest payment
// Then: Nt+ = Nt- - R(CNTRL) × actual_principal_paid
func (a *ANN) stfPR(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// PRECISION OPTIMIZATION: Increase precision for PR state transition
	oldPrecision := decimal.DivisionPrecision
	decimal.DivisionPrecision = 32
	defer func() { decimal.DivisionPrecision = oldPrecision }()

	roleSign := a.getRoleSign()

	// Calculate year fraction from last status date to PR event
	yf := utils.YearFraction(state.StatusDate, event.Time, a.Attributes.DayCountConvention)

	// Calculate interest for the period since last status date
	// Use absolute value of NotionalPrincipal for magnitude, then apply sign
	// This works whether or not PRF occurred:
	// - If PRF occurred, this calculates interest from PRF to PR (small period)
	// - If no PRF, this calculates interest from last event to PR (full period)
	periodInterest := yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal)

	// Update accrued interest by adding period interest (signed)
	state.AccruedInterest = state.AccruedInterest.Add(periodInterest)

	// For ANN: Total payment is constant (NextPrincipalPayment)
	// Use ABSOLUTE values for the calculation to handle RPL contracts correctly
	// Total interest payment magnitude for annuity calculation
	interestPaymentMagnitude := state.AccruedInterest.Abs()

	// Actual principal payment magnitude = Total payment - Interest payment magnitude
	totalPayment := state.NextPrincipalPayment
	principalPayment := totalPayment.Sub(interestPaymentMagnitude)

	// Ensure we don't pay more principal than remaining
	if principalPayment.GreaterThan(state.NotionalPrincipal.Abs()) {
		principalPayment = state.NotionalPrincipal.Abs()
	}

	// Nt+ = Nt- - R(CNTRL) × principal_payment
	// For RPA: Nt is positive, so subtracting positive decreases it
	// For RPL: Nt is negative, so subtracting negative increases it (toward zero)
	state.NotionalPrincipal = state.NotionalPrincipal.Sub(
		roleSign.Mul(principalPayment),
	)

	// Update fee accrual if fee basis is notional
	if a.Attributes.FeeBasis == "N" && !a.Attributes.FeeRate.IsZero() {
		// Use the same year fraction calculated above
		state.FeeAccrued = state.FeeAccrued.Add(
			yf.Mul(state.NotionalPrincipal).Mul(a.Attributes.FeeRate),
		)
	}

	// Update interest calculation base
	// If IPCB attribute is "NT", it follows the notional principal
	if a.Attributes.InterestCalculationBase == "NT" || a.Attributes.InterestCalculationBase == "" {
		state.InterestCalculationBase = state.NotionalPrincipal
	}

	// Sd+ = t
	state.StatusDate = event.Time

	return state
}

// stfIP implements the Interest Payment state transition for ANN
// Similar to LAM, but interest is part of the constant total payment
func (a *ANN) stfIP(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// First, accumulate any interest from last event to this IP
	// This is necessary when IP occurs alone (not preceded by PR on the same day)
	yf := utils.YearFraction(state.StatusDate, event.Time, a.Attributes.DayCountConvention)
	if !yf.IsZero() {
		periodInterest := yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal)
		state.AccruedInterest = state.AccruedInterest.Add(periodInterest)
	}

	// After POF is calculated, accrued interest will be reset to zero
	// But POF is calculated BEFORE state transition, so state.AccruedInterest
	// should have the correct value for POF calculation

	// Ipac+ = 0 (interest is paid, accrual resets to zero)
	state.AccruedInterest = decimal.Zero

	// Update fee accrual if fee basis is notional
	if a.Attributes.FeeBasis == "N" && !a.Attributes.FeeRate.IsZero() {
		state.FeeAccrued = state.FeeAccrued.Add(
			yf.Mul(state.NotionalPrincipal).Mul(a.Attributes.FeeRate),
		)
	}

	// Sd+ = t
	state.StatusDate = event.Time

	return state
}

// stfIPCB implements the Interest Calculation Base Fixing event
// This updates the interest calculation base for ANN contracts
func (a *ANN) stfIPCB(state *states.ContractState, event events.ContractEvent) *states.ContractState {
	// Update accrued interest up to this point
	yf := utils.YearFraction(state.StatusDate, event.Time, a.Attributes.DayCountConvention)
	state.AccruedInterest = state.AccruedInterest.Add(
		yf.Mul(state.NominalInterestRate).Mul(state.InterestCalculationBase),
	)

	// Update interest calculation base
	// Ipcb+ = Nt- (set to current notional principal)
	state.InterestCalculationBase = state.NotionalPrincipal

	// Sd+ = t
	state.StatusDate = event.Time

	return state
}
