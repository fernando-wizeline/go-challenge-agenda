package domain_test

import (
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func d(y int, m time.Month, day, h int) time.Time {
	return time.Date(y, m, day, h, 0, 0, 0, time.UTC)
}

func starts(occ []domain.BlockedSlot) []time.Time {
	var out []time.Time
	for _, o := range occ {
		out = append(out, o.StartsAt)
	}
	return out
}

func TestOccurrences_NonRecurring(t *testing.T) {
	b := &domain.BlockedSlot{StartsAt: d(2025, 6, 2, 10), EndsAt: d(2025, 6, 2, 11)}

	assert.Len(t, b.Occurrences(d(2025, 6, 2, 0), d(2025, 6, 3, 0)), 1)
	assert.Empty(t, b.Occurrences(d(2025, 6, 3, 0), d(2025, 6, 4, 0)))
	assert.Empty(t, b.Occurrences(d(2025, 6, 1, 0), d(2025, 6, 2, 9)))
	// inclusive boundaries
	assert.Len(t, b.Occurrences(d(2025, 6, 2, 11), d(2025, 6, 3, 0)), 1)
	assert.Len(t, b.Occurrences(d(2025, 6, 1, 0), d(2025, 6, 2, 10)), 1)
}

func TestOccurrences_Daily(t *testing.T) {
	b := &domain.BlockedSlot{
		ID: "x", Reason: "lunch",
		StartsAt: d(2025, 6, 2, 12), EndsAt: d(2025, 6, 2, 13),
		RecurrenceType: domain.RecurrenceDaily,
	}
	occ := b.Occurrences(d(2025, 6, 4, 0), d(2025, 6, 6, 23))
	assert.Equal(t, []time.Time{d(2025, 6, 4, 12), d(2025, 6, 5, 12), d(2025, 6, 6, 12)}, starts(occ))
	assert.Equal(t, d(2025, 6, 4, 13), occ[0].EndsAt)
	assert.Equal(t, "x", occ[0].ID)
	assert.Equal(t, "lunch", occ[0].Reason)
}

func TestOccurrences_WeeklyWithUntil(t *testing.T) {
	until := d(2025, 6, 16, 10) // equals the start of the 3rd occurrence: kept
	b := &domain.BlockedSlot{
		StartsAt: d(2025, 6, 2, 10), EndsAt: d(2025, 6, 2, 11),
		RecurrenceType: domain.RecurrenceWeekly, RecurrenceUntil: &until,
	}
	occ := b.Occurrences(d(2025, 6, 1, 0), d(2025, 12, 1, 0))
	assert.Equal(t, []time.Time{d(2025, 6, 2, 10), d(2025, 6, 9, 10), d(2025, 6, 16, 10)}, starts(occ))
}

func TestOccurrences_OverlapsWindowStart(t *testing.T) {
	b := &domain.BlockedSlot{
		StartsAt: d(2025, 6, 2, 22), EndsAt: d(2025, 6, 3, 2),
		RecurrenceType: domain.RecurrenceDaily,
	}
	occ := b.Occurrences(d(2025, 6, 5, 0), d(2025, 6, 5, 1))
	assert.Equal(t, []time.Time{d(2025, 6, 4, 22)}, starts(occ))
}

func TestOccurrences_WindowBeforeBase(t *testing.T) {
	b := &domain.BlockedSlot{
		StartsAt: d(2025, 6, 2, 10), EndsAt: d(2025, 6, 2, 11),
		RecurrenceType: domain.RecurrenceDaily,
	}
	assert.Empty(t, b.Occurrences(d(2025, 5, 1, 0), d(2025, 5, 31, 0)))
}

func TestOccurrences_FarFutureWindow(t *testing.T) {
	b := &domain.BlockedSlot{
		StartsAt: d(2025, 6, 2, 10), EndsAt: d(2025, 6, 2, 11),
		RecurrenceType: domain.RecurrenceWeekly,
	}
	occ := b.Occurrences(d(2030, 1, 1, 0), d(2030, 1, 14, 23))
	require.Len(t, occ, 2)
	for _, o := range occ {
		assert.Equal(t, time.Monday, o.StartsAt.Weekday())
	}
}

func TestOccurrences_MonthlyClamped(t *testing.T) {
	b := &domain.BlockedSlot{
		StartsAt: d(2025, 1, 31, 9), EndsAt: d(2025, 1, 31, 10),
		RecurrenceType: domain.RecurrenceMonthly,
	}
	occ := b.Occurrences(d(2025, 1, 1, 0), d(2025, 5, 1, 0))
	assert.Equal(t, []time.Time{
		d(2025, 1, 31, 9), d(2025, 2, 28, 9), d(2025, 3, 31, 9), d(2025, 4, 30, 9),
	}, starts(occ))

	// leap year
	leap := b.Occurrences(d(2028, 2, 1, 0), d(2028, 2, 29, 23))
	assert.Equal(t, []time.Time{d(2028, 2, 29, 9)}, starts(leap))
}
