package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Currency represents an ISO 4217 currency code.
type Currency string

const (
	CurrencyTWD Currency = "TWD"
	CurrencyUSD Currency = "USD"
)

// Money represents a monetary amount with currency, using decimal for zero precision loss.
type Money struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency Currency        `json:"currency"`
}

// NewMoney creates a new Money value.
func NewMoney(amount decimal.Decimal, currency Currency) Money {
	return Money{Amount: amount, Currency: currency}
}

// NewMoneyFromFloat creates Money from a float64 (for convenience; prefer decimal string).
func NewMoneyFromFloat(amount float64, currency Currency) Money {
	return Money{Amount: decimal.NewFromFloat(amount), Currency: currency}
}

// String returns formatted money string.
func (m Money) String() string {
	return m.Amount.StringFixed(2) + " " + string(m.Currency)
}

// PrecisionComparison holds a comparison between decimal and float64 calculations.
type PrecisionComparison struct {
	Label        string          `json:"label"`
	DecimalValue decimal.Decimal `json:"decimalValue"`
	FloatValue   float64         `json:"floatValue"`
	Difference   decimal.Decimal `json:"difference"`
}

// RiskAnalysisResult represents the output of various risk analysis models.
type RiskAnalysisResult struct {
	MarketRisk       decimal.Decimal `json:"marketRisk"`
	CreditRisk       decimal.Decimal `json:"creditRisk"`
	LiquidityRisk    decimal.Decimal `json:"liquidityRisk"`
	CounterpartyRisk decimal.Decimal `json:"counterpartyRisk"`
	StressTestImpact decimal.Decimal `json:"stressTestImpact"`
}

// ScenarioType identifies the type of financial scenario.
type ScenarioType string

const (
	ScenarioStablecoin ScenarioType = "STABLECOIN"
	ScenarioBond       ScenarioType = "BOND"
	ScenarioLoan       ScenarioType = "LOAN"
	ScenarioDerivative ScenarioType = "DERIVATIVE"
)

// ScenarioResult holds the complete output of a demo scenario.
type ScenarioResult struct {
	Type             ScenarioType            `json:"type"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	Instrument       *Instrument             `json:"instrument"`
	CashFlowEvents   []CashFlowEvent         `json:"cashFlowEvents"`
	Settlements      []Settlement            `json:"settlements"`
	JournalEntries   []JournalEntry          `json:"journalEntries"`
	PrecisionTests   []PrecisionComparison   `json:"precisionTests"`
	SolidityFiles    []GeneratedFile         `json:"solidityFiles"`
	ISO20022Messages []ISO20022Message       `json:"iso20022Messages"`
	VLEIVerification *VLEIVerificationResult `json:"vleiVerification"`
	RiskAnalysis     *RiskAnalysisResult     `json:"riskAnalysis,omitempty"`
	StartTime        time.Time               `json:"startTime"`
	EndTime          time.Time               `json:"endTime"`
}

// CashFlowEvent represents a single ACTUS contract event with payoff.
type CashFlowEvent struct {
	EventType    string          `json:"eventType"`
	Time         time.Time       `json:"time"`
	Payoff       decimal.Decimal `json:"payoff"`
	Currency     string          `json:"currency"`
	NominalValue decimal.Decimal `json:"nominalValue"`
	Description  string          `json:"description"`
}

// GeneratedFile represents a generated Solidity contract file.
type GeneratedFile struct {
	Filename       string `json:"filename"`
	Content        string `json:"content"`
	FileType       string `json:"fileType"`
	CompilerOutput string `json:"compilerOutput,omitempty"`
}

// ISO20022Message represents a generated ISO 20022 XML message.
type ISO20022Message struct {
	MessageType       string   `json:"messageType"`
	Description       string   `json:"description"`
	XMLContent        string   `json:"xmlContent"`
	ValidationStatus  string   `json:"validationStatus"`
	ValidationDetails []string `json:"validationDetails,omitempty"`
}
