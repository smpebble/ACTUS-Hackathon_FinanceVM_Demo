package states

import (
	"time"

	"github.com/shopspring/decimal"
)

// ContractState represents the state variables of a contract at a specific point in time
// These follow ACTUS specification naming conventions
type ContractState struct {
	// ===== Time =====
	StatusDate   time.Time `json:"statusDate"`   // Sd - State date
	MaturityDate time.Time `json:"maturityDate"` // Md - Maturity date

	// ===== Principal =====
	NotionalPrincipal    decimal.Decimal `json:"notionalPrincipal"`    // Nt - Notional principal
	NextPrincipalPayment decimal.Decimal `json:"nextPrincipalPayment"` // Prnxt - Next principal payment (for LAM/ANN)

	// ===== Interest =====
	NominalInterestRate         decimal.Decimal `json:"nominalInterestRate"`         // Ipnr - Nominal interest rate
	AccruedInterest             decimal.Decimal `json:"accruedInterest"`             // Ipac - Accrued interest
	PreviousNominalInterestRate decimal.Decimal `json:"previousNominalInterestRate"` // Ipnr_prev - Rate before last RR (for ANN recalculation)
	LastRateResetDate           time.Time       `json:"lastRateResetDate"`           // Date of last rate reset event

	// ===== Fees =====
	FeeAccrued decimal.Decimal `json:"feeAccrued"` // Feac - Fee accrued

	// ===== Interest Calculation Base (for LAM) =====
	InterestCalculationBase decimal.Decimal `json:"interestCalculationBase"` // Ipcb - Interest calculation base

	// ===== Scaling Indices =====
	NominalScalingIndex  decimal.Decimal `json:"nominalScalingIndex"`  // Nsc - Nominal scaling index
	InterestScalingIndex decimal.Decimal `json:"interestScalingIndex"` // Isc - Interest scaling index

	// ===== Contract Performance =====
	ContractPerformance string `json:"contractPerformance"` // Prf - Performance status

	// ===== Options (for OPTNS) =====
	ExerciseDate   *time.Time      `json:"exerciseDate,omitempty"`   // Xd - Exercise date
	ExerciseAmount decimal.Decimal `json:"exerciseAmount,omitempty"` // Xa - Exercise amount
}

// Clone creates a deep copy of the contract state
// This is essential to avoid modifying the previous state during STF calculations
func (cs *ContractState) Clone() *ContractState {
	clone := *cs

	// Deep copy pointer fields
	if cs.ExerciseDate != nil {
		d := *cs.ExerciseDate
		clone.ExerciseDate = &d
	}

	return &clone
}

// IsZero checks if the state is in its zero/uninitialized state
func (cs *ContractState) IsZero() bool {
	return cs.NotionalPrincipal.IsZero() &&
		cs.AccruedInterest.IsZero() &&
		cs.StatusDate.IsZero()
}

// String returns a human-readable representation of the state
func (cs *ContractState) String() string {
	return "ContractState{" +
		"Sd=" + cs.StatusDate.Format("2006-01-02") + ", " +
		"Nt=" + cs.NotionalPrincipal.String() + ", " +
		"Ipnr=" + cs.NominalInterestRate.String() + ", " +
		"Ipac=" + cs.AccruedInterest.String() +
		"}"
}

// NewContractState creates a new zero-initialized contract state
func NewContractState() *ContractState {
	return &ContractState{
		NotionalPrincipal:           decimal.Zero,
		NominalInterestRate:         decimal.Zero,
		AccruedInterest:             decimal.Zero,
		PreviousNominalInterestRate: decimal.Zero,
		FeeAccrued:                  decimal.Zero,
		InterestCalculationBase:     decimal.Zero,
		NominalScalingIndex:         decimal.NewFromInt(1),
		InterestScalingIndex:        decimal.NewFromInt(1),
		ContractPerformance:         "PF", // Default: Performant
		NextPrincipalPayment:        decimal.Zero,
		ExerciseAmount:              decimal.Zero,
	}
}
