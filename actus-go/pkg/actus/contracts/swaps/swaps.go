package swaps

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/scheduler"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/types"
)

// SWAPS Interest Rate Swap Contract
// Represents a contract where two parties exchange interest payment streams:
// - Fixed Leg: pays/receives fixed interest rate
// - Floating Leg: pays/receives floating interest rate (linked to market rate)
type SWAPS struct {
	Attributes *types.ContractAttributes
	scheduler  *scheduler.Scheduler
}

// NewSWAPS creates a new SWAPS contract
func NewSWAPS(attrs *types.ContractAttributes) (*SWAPS, error) {
	if attrs == nil {
		return nil, fmt.Errorf("contract attributes cannot be nil")
	}

	if err := validateSWAPSAttributes(attrs); err != nil {
		return nil, fmt.Errorf("invalid SWAPS attributes: %w", err)
	}

	return &SWAPS{
		Attributes: attrs,
		scheduler:  scheduler.NewScheduler(attrs),
	}, nil
}

// validateSWAPSAttributes validates SWAPS-specific attributes
func validateSWAPSAttributes(attrs *types.ContractAttributes) error {
	// Basic validation
	if attrs.ContractType != "SWAPS" {
		return fmt.Errorf("contract type must be SWAPS, got %s", attrs.ContractType)
	}

	// SWAPS must have an interest payment cycle
	if attrs.CycleOfInterestPayment == "" {
		return fmt.Errorf("CycleOfInterestPayment is required for SWAPS")
	}

	// SWAPS must have a notional principal
	if attrs.NotionalPrincipal.IsZero() {
		return fmt.Errorf("NotionalPrincipal must be non-zero for SWAPS")
	}

	// SWAPS must have interest rates defined
	if attrs.NominalInterestRate.IsZero() {
		return fmt.Errorf("NominalInterestRate is required for SWAPS")
	}

	// Validate contract role (must be RFL or PFL)
	if attrs.ContractRole != "RFL" && attrs.ContractRole != "PFL" {
		return fmt.Errorf("ContractRole must be RFL (Receive Fixed Leg) or PFL (Pay Fixed Leg), got %s", attrs.ContractRole)
	}

	return nil
}

// InitializeState initializes the SWAPS contract state
func (s *SWAPS) InitializeState() (*states.ContractState, error) {
	state := &states.ContractState{
		StatusDate:              s.Attributes.StatusDate,
		MaturityDate:            *s.Attributes.MaturityDate,
		NotionalPrincipal:       s.Attributes.NotionalPrincipal,
		NominalInterestRate:     s.Attributes.NominalInterestRate,
		AccruedInterest:         decimal.Zero,
		FeeAccrued:              decimal.Zero,
		NominalScalingIndex:     decimal.NewFromInt(1),
		InterestScalingIndex:    decimal.NewFromInt(1),
		ContractPerformance:     "PF", // Performant
		InterestCalculationBase: s.Attributes.NotionalPrincipal,
		NextPrincipalPayment:    decimal.Zero,
	}

	// For SWAPS:
	// - NominalInterestRate represents the FIXED leg rate
	// - Floating leg rate will be obtained from RiskFactorObserver during payoff calculation
	// - InterestCalculationBase is used to track the notional for both legs

	return state, nil
}

// getRoleSign returns the sign multiplier based on contract role
// RFL (Receive Fixed Leg): +1 (receive fixed, pay floating)
// PFL (Pay Fixed Leg): -1 (pay fixed, receive floating)
func (s *SWAPS) getRoleSign() decimal.Decimal {
	switch s.Attributes.ContractRole {
	case "RFL": // Receive Fixed Leg
		return decimal.NewFromInt(1)
	case "PFL": // Pay Fixed Leg
		return decimal.NewFromInt(-1)
	default:
		return decimal.NewFromInt(1)
	}
}
