package ann

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/utils"
)

// Payoff Functions (POF) for ANN contract
// ANN has constant total payments (principal + interest)
// Interest portion decreases over time, principal portion increases

// pofPR implements the Principal Redemption payoff for ANN
// For ANN, the principal payment is the total payment minus the ACCRUED interest
// Formula: POF_PR_ANN = R(CNTRL) × Nsc- × (Prnxt- - Ipac-)
// Where Prnxt is the total constant payment amount (always positive)
// and Ipac is accrued interest magnitude (always positive in this calculation)
// IMPORTANT: POF is calculated BEFORE STF, so we need to calculate period interest here
func (a *ANN) pofPR(state *states.ContractState, eventTime time.Time) decimal.Decimal {
	// PRECISION OPTIMIZATION: Increase precision for PR payoff calculation
	oldPrecision := decimal.DivisionPrecision
	decimal.DivisionPrecision = 32
	defer func() { decimal.DivisionPrecision = oldPrecision }()

	roleSign := a.getRoleSign()

	// Calculate total accrued interest INCLUDING period interest since last status date
	// Use ABSOLUTE values for calculation - roleSign determines final direction
	// AccruedInterest may be negative for RPL contracts (because NotionalPrincipal is negative)
	accruedInterest := state.AccruedInterest.Abs()

	// Calculate period interest from last status date to PR
	// Use Abs() on NotionalPrincipal to ensure positive interest amount
	yf := utils.YearFraction(state.StatusDate, eventTime, a.Attributes.DayCountConvention)
	if !yf.IsZero() {
		periodInterest := yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal.Abs())
		accruedInterest = accruedInterest.Add(periodInterest)
	}

	// Principal payment = Total payment - Total Accrued Interest
	// Both are now positive magnitudes
	totalPayment := state.NextPrincipalPayment
	principalPayment := totalPayment.Sub(accruedInterest)

	// Ensure we don't return more than the remaining principal
	if principalPayment.GreaterThan(state.NotionalPrincipal.Abs()) {
		principalPayment = state.NotionalPrincipal.Abs()
	}

	// Payoff = roleSign × scaling index × principal payment magnitude
	// roleSign determines whether this is a payment (negative for RPL) or receipt (positive for RPA)
	payoff := roleSign.Mul(state.NominalScalingIndex).Mul(principalPayment)

	return payoff
}

// pofIP implements the Interest Payment payoff for ANN
// Formula: POF_IP_ANN = R(CNTRL) × Isc- × |Ipac-|
// Interest payment is the accrued interest at the time of IP event
// RoleSign determines direction: positive for RPA (receive interest), negative for RPL (pay interest)
// IMPORTANT: POF is calculated BEFORE STF, so we need to handle period interest here
func (a *ANN) pofIP(state *states.ContractState, eventTime time.Time) decimal.Decimal {
	roleSign := a.getRoleSign()

	// Calculate accrued interest INCLUDING any period since last status date
	// Use ABSOLUTE values - roleSign determines final direction
	accruedInterest := state.AccruedInterest.Abs()

	// If there's a period since last status date, accumulate interest
	yf := utils.YearFraction(state.StatusDate, eventTime, a.Attributes.DayCountConvention)
	if !yf.IsZero() {
		periodInterest := yf.Mul(state.NominalInterestRate).Mul(state.NotionalPrincipal.Abs())
		accruedInterest = accruedInterest.Add(periodInterest)
	}

	// Interest payment = roleSign × scaling index × accrued interest magnitude
	interestPayment := roleSign.Mul(state.InterestScalingIndex).Mul(accruedInterest)

	return interestPayment
}
