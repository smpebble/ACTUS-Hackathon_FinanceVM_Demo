package pam

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/yourusername/actus-go/pkg/actus/events"
	"github.com/yourusername/actus-go/pkg/actus/scheduler"
	"github.com/yourusername/actus-go/pkg/actus/states"
	"github.com/yourusername/actus-go/pkg/actus/types"
	"github.com/yourusername/actus-go/pkg/riskfactor"
)

// PAM represents a Principal At Maturity contract
// This is a fixed-rate loan where the principal is repaid in full at maturity
// with periodic interest payments
type PAM struct {
	Attributes *types.ContractAttributes
	scheduler  *scheduler.Scheduler
}

// NewPAM creates a new PAM contract instance
// It validates the contract attributes and returns an error if validation fails
func NewPAM(attrs *types.ContractAttributes) (*PAM, error) {
	// Set defaults
	attrs.SetDefaults()

	// Validate attributes
	if err := attrs.Validate(); err != nil {
		return nil, fmt.Errorf("PAM validation failed: %w", err)
	}

	return &PAM{
		Attributes: attrs,
		scheduler:  scheduler.NewScheduler(attrs),
	}, nil
}

// GenerateSchedule generates the event schedule for the contract
func (p *PAM) GenerateSchedule() (events.EventSchedule, error) {
	return p.scheduler.Schedule()
}

// InitializeState creates the initial contract state
func (p *PAM) InitializeState() (*states.ContractState, error) {
	return states.InitializeState(p.Attributes)
}

// getRoleSign returns the sign multiplier based on contract role
// +1 for asset positions (RPA, LG, BUY, RFL)
// -1 for liability positions (RPL, ST, SEL, PFL)
func (p *PAM) getRoleSign() decimal.Decimal {
	switch p.Attributes.ContractRole {
	case "RPA", "LG", "BUY", "RFL":
		return decimal.NewFromInt(1)
	case "RPL", "ST", "SEL", "PFL":
		return decimal.NewFromInt(-1)
	default:
		return decimal.NewFromInt(1)
	}
}

// StateTransition performs state transition for a given event
// This implements the State Transition Function (STF) according to ACTUS spec
func (p *PAM) StateTransition(
	event events.ContractEvent,
	preState *states.ContractState,
	rf riskfactor.Observer,
) (*states.ContractState, error) {
	// Clone the state to avoid modifying the original
	postState := preState.Clone()

	// Dispatch to specific STF based on event type
	switch event.Type {
	case events.AD:
		return p.stfAD(postState, event), nil
	case events.IED:
		return p.stfIED(postState, event), nil
	case events.IP:
		return p.stfIP(postState, event), nil
	case events.PRD:
		return p.stfPRD(postState, event), nil
	case events.TD:
		return p.stfTD(postState, event), nil
	case events.MD:
		return p.stfMD(postState, event), nil
	case events.RR:
		return p.stfRR(postState, event, rf), nil
	case events.RRF:
		return p.stfRRF(postState, event, rf), nil
	case events.IPCI:
		return p.stfIPCI(postState, event), nil
	case events.FP:
		return p.stfFP(postState, event), nil
	case events.PP:
		return p.stfPP(postState, event, rf), nil
	case events.SC:
		return p.stfSC(postState, event, rf), nil
	default:
		return postState, fmt.Errorf("unsupported event type for PAM: %s", event.Type)
	}
}

// CalculatePayoff calculates the payoff for a given event
// This implements the Payoff Function (POF) according to ACTUS spec
func (p *PAM) CalculatePayoff(
	event events.ContractEvent,
	state *states.ContractState,
	rf riskfactor.Observer,
) (decimal.Decimal, error) {
	switch event.Type {
	case events.AD:
		return p.pofAD(), nil
	case events.IED:
		return p.pofIED(state), nil
	case events.IP:
		// Pass both schedule time and payment time to pofIP
		// pofIP will decide which one to use based on business day convention
		return p.pofIP(state, event.ScheduleTime, event.Time), nil
	case events.PRD:
		return p.pofPRD(state, event.ScheduleTime), nil
	case events.TD:
		return p.pofTD(state, event.ScheduleTime), nil
	case events.MD:
		return p.pofMD(state), nil
	case events.FP:
		return p.pofFP(state, event.ScheduleTime), nil
	case events.RR:
		// Rate Reset event has no cashflow (payoff = 0)
		return decimal.Zero, nil
	case events.RRF:
		// Rate Reset Fixed event has no cashflow (payoff = 0)
		return decimal.Zero, nil
	case events.IPCI:
		// Interest Capitalization event has no cashflow (payoff = 0)
		// Interest is added to principal instead of being paid out
		return decimal.Zero, nil
	case events.SC:
		return p.pofSC(), nil
	default:
		return decimal.Zero, nil
	}
}
