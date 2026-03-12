package states

import (
	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/types"
)

// InitializeState creates the initial contract state from attributes
// This is the state at t0 (status date or IED, whichever comes first)
func InitializeState(attrs *types.ContractAttributes) (*ContractState, error) {
	state := NewContractState()

	// Set dates
	state.StatusDate = attrs.StatusDate
	if attrs.MaturityDate != nil {
		state.MaturityDate = *attrs.MaturityDate
	}

	// If StatusDate is before IED, state variables remain zero
	if attrs.StatusDate.Before(attrs.InitialExchangeDate) {
		return state, nil
	}

	// Get role sign (1 for asset/long, -1 for liability/short)
	roleSign := getRoleSign(attrs.ContractRole)

	// Initialize principal
	state.NotionalPrincipal = roleSign.Mul(attrs.NotionalPrincipal)

	// Initialize interest rate
	state.NominalInterestRate = attrs.NominalInterestRate

	// Initialize accrued interest
	if !attrs.AccruedInterest.IsZero() {
		state.AccruedInterest = attrs.AccruedInterest
	} else {
		// If not specified, accrued interest starts at zero at IED
		state.AccruedInterest = decimal.Zero
	}

	// Initialize fee
	state.FeeAccrued = attrs.FeeAccrued

	// Initialize scaling indices (default to 1)
	state.NominalScalingIndex = decimal.NewFromInt(1)
	state.InterestScalingIndex = decimal.NewFromInt(1)

	// Initialize contract performance
	state.ContractPerformance = attrs.ContractPerformance
	if state.ContractPerformance == "" {
		state.ContractPerformance = "PF" // Default: Performant
	}

	// Contract type specific initialization
	switch attrs.ContractType {
	case "LAM", "ANN":
		state.NextPrincipalPayment = attrs.NextPrincipalRedemptionPayment
		// Interest calculation base for LAM
		if attrs.InterestCalculationBase == "NT" || attrs.InterestCalculationBase == "" {
			state.InterestCalculationBase = state.NotionalPrincipal
		} else if !attrs.InterestCalculationBaseAmount.IsZero() {
			state.InterestCalculationBase = roleSign.Mul(attrs.InterestCalculationBaseAmount)
		} else {
			state.InterestCalculationBase = state.NotionalPrincipal
		}
	}

	return state, nil
}

// getRoleSign returns the sign multiplier based on contract role
// +1 for asset/long positions, -1 for liability/short positions
func getRoleSign(role string) decimal.Decimal {
	switch role {
	case "RPA", // Receive Payoff from Asset
		"LG",  // Long (options)
		"BUY", // Buy
		"RFL", // Receive First Leg
		"RF":  // Receive Fixed (SWPPV)
		return decimal.NewFromInt(1)
	case "RPL", // Receive Payoff from Liability
		"ST",  // Short (options)
		"SEL", // Sell
		"PFL", // Pay First Leg
		"PF":  // Pay Fixed (SWPPV)
		return decimal.NewFromInt(-1)
	default:
		// Default to asset position
		return decimal.NewFromInt(1)
	}
}
