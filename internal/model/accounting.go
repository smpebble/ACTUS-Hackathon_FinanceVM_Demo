package model

import (
	"time"

	"github.com/google/uuid"
)

// EntryType represents a debit or credit journal entry.
type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

// JournalEntryLine represents a single line in a journal entry.
type JournalEntryLine struct {
	EntryType   EntryType `json:"entryType"`
	AccountName string    `json:"accountName"`
	Amount      Money     `json:"amount"`
	Description string    `json:"description"`
}

// JournalEntry represents a double-entry accounting journal entry.
type JournalEntry struct {
	ID          string             `json:"id"`
	EntryDate   time.Time          `json:"entryDate"`
	Description string             `json:"description"`
	Lines       []JournalEntryLine `json:"lines"`
	CreatedAt   time.Time          `json:"createdAt"`
}

// NewJournalEntry creates a new journal entry.
func NewJournalEntry(date time.Time, description string, lines []JournalEntryLine) *JournalEntry {
	return &JournalEntry{
		ID:          uuid.New().String(),
		EntryDate:   date,
		Description: description,
		Lines:       lines,
		CreatedAt:   time.Now(),
	}
}
