// Package adapter provides adapters wrapping external project APIs.
package adapter

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/contracts/ann"
	"github.com/yourusername/actus-go/pkg/actus/contracts/pam"
	"github.com/yourusername/actus-go/pkg/actus/contracts/swaps"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/types"
	"github.com/yourusername/actus-go/pkg/riskfactor"

	"github.com/smpebble/actus-fvm/internal/model"
)

// ACTUSClient wraps the actus-go contract engine.
type ACTUSClient struct{}

// NewACTUSClient creates a new ACTUS client.
func NewACTUSClient() *ACTUSClient {
	return &ACTUSClient{}
}

// CreatePAMContract creates a PAM (Principal At Maturity) contract and generates its cashflow schedule.
func (c *ACTUSClient) CreatePAMContract(attrs *types.ContractAttributes) (*pam.PAM, events.EventSchedule, *states.ContractState, error) {
	contract, err := pam.NewPAM(attrs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create PAM contract: %w", err)
	}

	schedule, err := contract.GenerateSchedule()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate PAM schedule: %w", err)
	}
	schedule.Sort()

	state, err := contract.InitializeState()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize PAM state: %w", err)
	}

	return contract, schedule, state, nil
}

// CreateANNContract creates an ANN (Annuity) contract and generates its cashflow schedule.
func (c *ACTUSClient) CreateANNContract(attrs *types.ContractAttributes) (*ann.ANN, events.EventSchedule, *states.ContractState, error) {
	contract, err := ann.NewANN(attrs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create ANN contract: %w", err)
	}

	schedule, err := contract.GenerateSchedule()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate ANN schedule: %w", err)
	}
	schedule.Sort()

	state, err := contract.InitializeState()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize ANN state: %w", err)
	}

	return contract, schedule, state, nil
}

// CreateSWAPSContract creates a SWAPS (Interest Rate Swap) contract and generates its cashflow schedule.
func (c *ACTUSClient) CreateSWAPSContract(attrs *types.ContractAttributes) (*swaps.SWAPS, *states.ContractState, error) {
	contract, err := swaps.NewSWAPS(attrs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SWAPS contract: %w", err)
	}

	state, err := contract.InitializeState()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize SWAPS state: %w", err)
	}

	return contract, state, nil
}

// SimulateCashflows processes the event schedule through state transitions and payoff calculations.
func (c *ACTUSClient) SimulateCashflows(
	contract interface{ StateTransition(events.ContractEvent, *states.ContractState, riskfactor.Observer) (*states.ContractState, error) },
	payoffCalc interface{ CalculatePayoff(events.ContractEvent, *states.ContractState, riskfactor.Observer) (decimal.Decimal, error) },
	schedule events.EventSchedule,
	initialState *states.ContractState,
	rf riskfactor.Observer,
) ([]model.CashFlowEvent, error) {
	var cashflows []model.CashFlowEvent
	currentState := initialState

	for _, event := range schedule {
		payoff, err := payoffCalc.CalculatePayoff(event, currentState, rf)
		if err != nil {
			return nil, fmt.Errorf("payoff calculation failed for %s at %s: %w", event.Type, event.Time.Format("2006-01-02"), err)
		}
		event.Payoff = payoff

		newState, err := contract.StateTransition(event, currentState, rf)
		if err != nil {
			return nil, fmt.Errorf("state transition failed for %s at %s: %w", event.Type, event.Time.Format("2006-01-02"), err)
		}

		cashflows = append(cashflows, model.CashFlowEvent{
			EventType:    string(event.Type),
			Time:         event.Time,
			Payoff:       payoff,
			Currency:     event.Currency,
			NominalValue: newState.NotionalPrincipal,
			Description:  describeEvent(event.Type, payoff),
		})

		currentState = newState
	}

	return cashflows, nil
}

// StaticRiskFactorObserver provides static market rates for demo purposes.
type StaticRiskFactorObserver struct {
	rates map[string]decimal.Decimal
}

// NewStaticRiskFactorObserver creates an observer with predefined rates.
func NewStaticRiskFactorObserver(rates map[string]decimal.Decimal) *StaticRiskFactorObserver {
	return &StaticRiskFactorObserver{rates: rates}
}

func (o *StaticRiskFactorObserver) GetMarketRate(code string, t time.Time) (decimal.Decimal, error) {
	if rate, ok := o.rates[code]; ok {
		return rate, nil
	}
	return decimal.Zero, fmt.Errorf("no rate found for %s", code)
}

func (o *StaticRiskFactorObserver) GetFXRate(pair string, t time.Time) (decimal.Decimal, error) {
	return decimal.NewFromInt(1), nil
}

func (o *StaticRiskFactorObserver) ObservePrepayment(contractID string, t time.Time) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (o *StaticRiskFactorObserver) ObserveDefault(contractID string, t time.Time) (bool, error) {
	return false, nil
}

func describeEvent(eventType events.EventType, payoff decimal.Decimal) string {
	descriptions := map[events.EventType]string{
		"IED":  "Initial Exchange",
		"IP":   "Interest Payment",
		"IPCI": "Interest Capitalization",
		"PR":   "Principal Redemption",
		"PRF":  "Principal Redemption Fixing",
		"MD":   "Maturity",
		"RR":   "Rate Reset",
		"RRF":  "Rate Reset Fixed",
		"FP":   "Fee Payment",
		"PP":   "Principal Prepayment",
		"TD":   "Termination",
		"AD":   "Analysis Date",
		"IPFX": "Interest Payment Fixed Leg",
		"IPFL": "Interest Payment Floating Leg",
		"SC":   "Scaling Index Fixing",
	}
	if desc, ok := descriptions[eventType]; ok {
		return desc
	}
	return string(eventType)
}
