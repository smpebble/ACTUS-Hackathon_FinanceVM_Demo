package adapter

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/smpebble/actus-fvm/internal/model"
)

// FVMClient simulates the FinanceVM instrument lifecycle (in-memory mock).
type FVMClient struct {
	instruments map[string]*model.Instrument
}

// NewFVMClient creates a new FVM client.
func NewFVMClient() *FVMClient {
	return &FVMClient{
		instruments: make(map[string]*model.Instrument),
	}
}

// CreateInstrument creates and registers a new financial instrument.
func (c *FVMClient) CreateInstrument(
	name, description, issuer string,
	currency model.Currency,
	actusType string,
	tokenStd model.TokenStandard,
) *model.Instrument {
	inst := model.NewInstrument(name, description, issuer, currency, actusType, tokenStd)
	c.instruments[inst.ID] = inst
	return inst
}

// IssueInstrument transitions an instrument to ISSUED state.
func (c *FVMClient) IssueInstrument(instrumentID string, amount decimal.Decimal, issueDate time.Time) error {
	inst, ok := c.instruments[instrumentID]
	if !ok {
		return fmt.Errorf("instrument not found: %s", instrumentID)
	}
	inst.Issue(model.NewMoney(amount, inst.Currency), issueDate)
	return nil
}

// ActivateInstrument transitions an instrument to ACTIVE state.
func (c *FVMClient) ActivateInstrument(instrumentID string) error {
	inst, ok := c.instruments[instrumentID]
	if !ok {
		return fmt.Errorf("instrument not found: %s", instrumentID)
	}
	inst.Activate()
	return nil
}

// GetInstrument retrieves an instrument by ID.
func (c *FVMClient) GetInstrument(instrumentID string) (*model.Instrument, error) {
	inst, ok := c.instruments[instrumentID]
	if !ok {
		return nil, fmt.Errorf("instrument not found: %s", instrumentID)
	}
	return inst, nil
}

// CreateSettlement creates a settlement for a cashflow event.
func (c *FVMClient) CreateSettlement(
	instrumentID string,
	sType model.SettlementType,
	deliverer, receiver string,
	amount model.Money,
	date time.Time,
) *model.Settlement {
	settlement := model.NewSettlement(instrumentID, sType, deliverer, receiver, amount, date)
	return settlement
}

// SettlePayment processes and settles a payment.
func (c *FVMClient) SettlePayment(settlement *model.Settlement) {
	settlement.Settle()
}

// CreateJournalEntry creates a double-entry accounting journal entry.
func (c *FVMClient) CreateJournalEntry(
	date time.Time,
	description string,
	debitAccount string,
	creditAccount string,
	amount model.Money,
) *model.JournalEntry {
	return model.NewJournalEntry(date, description, []model.JournalEntryLine{
		{EntryType: model.EntryTypeDebit, AccountName: debitAccount, Amount: amount, Description: description},
		{EntryType: model.EntryTypeCredit, AccountName: creditAccount, Amount: amount, Description: description},
	})
}
