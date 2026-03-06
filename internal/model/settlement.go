package model

import (
	"time"

	"github.com/google/uuid"
)

// SettlementType represents the type of settlement.
type SettlementType string

const (
	SettlementTypeDVP SettlementType = "DVP" // Delivery vs Payment
	SettlementTypePVP SettlementType = "PVP" // Payment vs Payment
	SettlementTypeFOP SettlementType = "FOP" // Free of Payment
)

// SettlementState represents the state of a settlement.
type SettlementState string

const (
	SettlementStateInstructed SettlementState = "INSTRUCTED"
	SettlementStateMatched    SettlementState = "MATCHED"
	SettlementStatePending    SettlementState = "PENDING_SETTLEMENT"
	SettlementStateSettled    SettlementState = "SETTLED"
	SettlementStateFailed     SettlementState = "FAILED"
)

// Settlement represents a financial settlement (adapted from FinanceVM).
type Settlement struct {
	ID             string          `json:"id"`
	InstrumentID   string          `json:"instrumentId"`
	SettlementType SettlementType  `json:"settlementType"`
	Deliverer      string          `json:"deliverer"`
	Receiver       string          `json:"receiver"`
	CashAmount     Money           `json:"cashAmount"`
	SettlementDate time.Time       `json:"settlementDate"`
	State          SettlementState `json:"state"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// NewSettlement creates a new settlement instruction.
func NewSettlement(instrumentID string, sType SettlementType, deliverer, receiver string, amount Money, date time.Time) *Settlement {
	return &Settlement{
		ID:             uuid.New().String(),
		InstrumentID:   instrumentID,
		SettlementType: sType,
		Deliverer:      deliverer,
		Receiver:       receiver,
		CashAmount:     amount,
		SettlementDate: date,
		State:          SettlementStateInstructed,
		CreatedAt:      time.Now(),
	}
}

// Settle transitions the settlement to SETTLED state.
func (s *Settlement) Settle() {
	s.State = SettlementStateSettled
}
