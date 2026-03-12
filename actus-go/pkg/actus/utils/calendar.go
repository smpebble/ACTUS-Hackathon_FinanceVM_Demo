package utils

import (
	"time"

	"github.com/yourusername/actus-go/pkg/actus/types"
)

// Calendar defines the interface for business day calendars
type Calendar interface {
	IsBusinessDay(date time.Time) bool
	AdjustDate(date time.Time, convention types.BusinessDayConvention) time.Time
}

// ===== NoHolidayCalendar =====

// NoHolidayCalendar treats all days as business days
type NoHolidayCalendar struct{}

// NewNoHolidayCalendar creates a new NoHoliday calendar
func NewNoHolidayCalendar() *NoHolidayCalendar {
	return &NoHolidayCalendar{}
}

// IsBusinessDay always returns true
func (c *NoHolidayCalendar) IsBusinessDay(date time.Time) bool {
	return true
}

// AdjustDate returns the date unchanged (no adjustment needed)
func (c *NoHolidayCalendar) AdjustDate(date time.Time, convention types.BusinessDayConvention) time.Time {
	return date
}

// ===== MondayToFridayCalendar =====

// MondayToFridayCalendar treats Monday-Friday as business days
type MondayToFridayCalendar struct{}

// NewMondayToFridayCalendar creates a new Monday-Friday calendar
func NewMondayToFridayCalendar() *MondayToFridayCalendar {
	return &MondayToFridayCalendar{}
}

// IsBusinessDay returns true for Monday-Friday
func (c *MondayToFridayCalendar) IsBusinessDay(date time.Time) bool {
	weekday := date.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

// AdjustDate adjusts non-business days according to the convention
func (c *MondayToFridayCalendar) AdjustDate(date time.Time, convention types.BusinessDayConvention) time.Time {
	// If already a business day, no adjustment needed
	if c.IsBusinessDay(date) {
		return date
	}

	switch convention {
	case types.BDC_NULL:
		// No adjustment
		return date

	case types.BDC_SCF, types.BDC_CSF:
		// Following: move forward to next business day
		return c.adjustFollowing(date)

	case types.BDC_SCMF, types.BDC_CSMF:
		// Modified Following: move forward, but if crosses month boundary, move backward
		adjusted := c.adjustFollowing(date)
		if adjusted.Month() != date.Month() {
			return c.adjustPreceding(date)
		}
		return adjusted

	case types.BDC_SCP, types.BDC_CSP:
		// Preceding: move backward to previous business day
		return c.adjustPreceding(date)

	case types.BDC_SCMP, types.BDC_CSMP:
		// Modified Preceding: move backward, but if crosses month boundary, move forward
		adjusted := c.adjustPreceding(date)
		if adjusted.Month() != date.Month() {
			return c.adjustFollowing(date)
		}
		return adjusted

	default:
		// Default: no adjustment
		return date
	}
}

// adjustFollowing moves the date forward to the next business day
func (c *MondayToFridayCalendar) adjustFollowing(date time.Time) time.Time {
	adjusted := date
	for !c.IsBusinessDay(adjusted) {
		adjusted = adjusted.AddDate(0, 0, 1)
	}
	return adjusted
}

// adjustPreceding moves the date backward to the previous business day
func (c *MondayToFridayCalendar) adjustPreceding(date time.Time) time.Time {
	adjusted := date
	for !c.IsBusinessDay(adjusted) {
		adjusted = adjusted.AddDate(0, 0, -1)
	}
	return adjusted
}

// ===== Calendar Factory =====

// GetCalendar returns a calendar instance based on the name
func GetCalendar(name string) Calendar {
	switch name {
	case "MF", "MondayToFriday":
		// MF = Monday-Friday calendar (ACTUS standard code)
		return NewMondayToFridayCalendar()
	case "NoHoliday", "NH", "":
		// NoHoliday calendar (no non-business days)
		return NewNoHolidayCalendar()
	default:
		// Default to NoHoliday for unknown calendar codes
		return NewNoHolidayCalendar()
	}
}
