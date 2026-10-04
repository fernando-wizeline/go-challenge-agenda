package usecase_test

import (
	"context"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/repository/sqlite"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListBlockedSlots_ExpandsRecurrence(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	uc := usecase.NewBlockedSlotUsecase(sqlite.NewBlockedSlotRepository(db))
	ctx := context.Background()

	at := func(m time.Month, day, h int) time.Time { return time.Date(2025, m, day, h, 0, 0, 0, time.UTC) }
	until := at(3, 31, 0)

	// weekly series created in January, ending in March
	_, err = uc.Create(ctx, &domain.BlockedSlot{DoctorID: "doc-001", StartsAt: at(1, 6, 10), EndsAt: at(1, 6, 11),
		RecurrenceType: domain.RecurrenceWeekly, RecurrenceUntil: &until})
	require.NoError(t, err)
	// open-ended daily series
	_, err = uc.Create(ctx, &domain.BlockedSlot{DoctorID: "doc-001", StartsAt: at(1, 1, 12), EndsAt: at(1, 1, 13),
		RecurrenceType: domain.RecurrenceDaily})
	require.NoError(t, err)
	// other doctor, must not leak
	_, err = uc.Create(ctx, &domain.BlockedSlot{DoctorID: "doc-002", StartsAt: at(1, 1, 12), EndsAt: at(1, 1, 13),
		RecurrenceType: domain.RecurrenceDaily})
	require.NoError(t, err)

	// March window: weekly Mondays 3,10,17,24,31(<=until? 31 00:00 < 10:00 so no) + 3 daily days
	got, err := uc.List(ctx, "doc-001", at(3, 3, 0), at(3, 5, 23))
	require.NoError(t, err)
	assert.Len(t, got, 1+3) // Mon 3 Mar weekly + daily on 3,4,5

	// After the weekly series ended only the daily one remains
	got, err = uc.List(ctx, "doc-001", at(4, 7, 0), at(4, 8, 23))
	require.NoError(t, err)
	assert.Len(t, got, 2)
}
