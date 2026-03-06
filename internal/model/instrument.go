package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InstrumentState represents the lifecycle state of a financial instrument.
type InstrumentState string

const (
	InstrumentStateDraft      InstrumentState = "DRAFT"
	InstrumentStateIssued     InstrumentState = "ISSUED"
	InstrumentStateActive     InstrumentState = "ACTIVE"
	InstrumentStateSuspended  InstrumentState = "SUSPENDED"
	InstrumentStateDefaulted  InstrumentState = "DEFAULTED"
	InstrumentStateMatured    InstrumentState = "MATURED"
	InstrumentStateTerminated InstrumentState = "TERMINATED"
)

// TokenStandard represents the token standard for on-chain deployment.
type TokenStandard string

const (
	TokenStandardERC20   TokenStandard = "ERC-20"
	TokenStandardERC1400 TokenStandard = "ERC-1400"
	TokenStandardERC3643 TokenStandard = "ERC-3643"
	TokenStandardCustom  TokenStandard = "CUSTOM"
)

// Instrument represents a tokenized financial instrument (adapted from FinanceVM).
type Instrument struct {
	ID                string          `json:"id"`
	ISIN              string          `json:"isin,omitempty"`
	FullName          string          `json:"fullName"`
	Description       string          `json:"description"`
	Issuer            string          `json:"issuer"`
	Currency          Currency        `json:"currency"`
	IssuedAmount      Money           `json:"issuedAmount"`
	OutstandingAmount Money           `json:"outstandingAmount"`
	NominalValue      Money           `json:"nominalValue"`
	IssueDate         time.Time       `json:"issueDate"`
	MaturityDate      *time.Time      `json:"maturityDate,omitempty"`
	State             InstrumentState `json:"state"`
	ACTUSContractType string          `json:"actusContractType"`
	TokenStandard     TokenStandard   `json:"tokenStandard"`
	InterestRate      decimal.Decimal `json:"interestRate,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// NewInstrument creates a new instrument in DRAFT state.
func NewInstrument(name, description, issuer string, currency Currency, actusType string, tokenStd TokenStandard) *Instrument {
	now := time.Now()
	return &Instrument{
		ID:                uuid.New().String(),
		FullName:          name,
		Description:       description,
		Issuer:            issuer,
		Currency:          currency,
		State:             InstrumentStateDraft,
		ACTUSContractType: actusType,
		TokenStandard:     tokenStd,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Issue transitions the instrument to ISSUED state.
func (i *Instrument) Issue(amount Money, issueDate time.Time) {
	i.IssuedAmount = amount
	i.OutstandingAmount = amount
	i.IssueDate = issueDate
	i.State = InstrumentStateIssued
	i.UpdatedAt = time.Now()
}

// Activate transitions the instrument to ACTIVE state.
func (i *Instrument) Activate() {
	i.State = InstrumentStateActive
	i.UpdatedAt = time.Now()
}

// Mature transitions the instrument to MATURED state.
func (i *Instrument) Mature() {
	i.State = InstrumentStateMatured
	i.UpdatedAt = time.Now()
}
