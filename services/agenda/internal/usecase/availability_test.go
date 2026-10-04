package usecase_test

import (
	"context"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/domain/mocks"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mondayDoctor() *domain.Doctor {
	return &domain.Doctor{
		ID:        "doc-001",
		Name:      "Dr. Test",
		Specialty: "General",
		WorkingHours: []domain.WorkingHours{
			{Weekday: domain.Monday, From: "09:00", To: "17:00"},
		},
	}
}

func nextMonday() time.Time {
	t := time.Now().UTC()
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, 1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func TestGetAvailability_HappyPath(t *testing.T) {
	date := nextMonday()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.UTC)

	doctorRepo := mocks.NewDoctorRepository(t)
	reservationRepo := mocks.NewReservationRepository(t)
	blockedSlotRepo := mocks.NewBlockedSlotRepository(t)

	doctorRepo.EXPECT().GetDoctor(context.Background(), "doc-001").Return(mondayDoctor(), nil)
	reservationRepo.EXPECT().ListReservations(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)
	blockedSlotRepo.EXPECT().ListBlockedSlots(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)

	uc := usecase.NewAvailabilityUsecase(doctorRepo, reservationRepo, blockedSlotRepo)

	result, err := uc.GetAvailability(context.Background(), "doc-001", date, domain.ReservationTypeFollowUp)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Slots)
	assert.NotEmpty(t, result.FreeRanges)
}

// TestGetAvailability_WithBlockedSlots verifies that blocked slots remove time from availability.
func TestGetAvailability_WithBlockedSlots(t *testing.T) {
	date := nextMonday()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.UTC)
	blockedStart := time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, time.UTC)
	blockedEnd := time.Date(date.Year(), date.Month(), date.Day(), 11, 0, 0, 0, time.UTC)

	doctorRepo := mocks.NewDoctorRepository(t)
	reservationRepo := mocks.NewReservationRepository(t)
	blockedSlotRepo := mocks.NewBlockedSlotRepository(t)

	doctorRepo.EXPECT().GetDoctor(context.Background(), "doc-001").Return(mondayDoctor(), nil)
	reservationRepo.EXPECT().ListReservations(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)
	blockedSlotRepo.EXPECT().ListBlockedSlots(context.Background(), "doc-001", dayStart, dayEnd).
		Return([]*domain.BlockedSlot{{StartsAt: blockedStart, EndsAt: blockedEnd}}, nil)

	uc := usecase.NewAvailabilityUsecase(doctorRepo, reservationRepo, blockedSlotRepo)

	result, err := uc.GetAvailability(context.Background(), "doc-001", date, domain.ReservationTypeFollowUp)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Slots)

	for _, slot := range result.Slots {
		if !slot.StartsAt.Before(blockedStart) && slot.StartsAt.Before(blockedEnd) {
			t.Errorf("slot %v falls within blocked period [%v, %v]", slot.StartsAt, blockedStart, blockedEnd)
		}
	}
}

// A weekly block created weeks earlier must still block today's availability.
func TestGetAvailability_WithRecurringBlockedSlot(t *testing.T) {
	date := nextMonday()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.UTC)
	weekly := &domain.BlockedSlot{
		StartsAt:       dayStart.AddDate(0, 0, -21).Add(3 * time.Hour), // 12:00, three Mondays ago
		EndsAt:         dayStart.AddDate(0, 0, -21).Add(4 * time.Hour), // 13:00
		RecurrenceType: domain.RecurrenceWeekly,
	}

	doctorRepo := mocks.NewDoctorRepository(t)
	reservationRepo := mocks.NewReservationRepository(t)
	blockedSlotRepo := mocks.NewBlockedSlotRepository(t)
	doctorRepo.EXPECT().GetDoctor(context.Background(), "doc-001").Return(mondayDoctor(), nil)
	reservationRepo.EXPECT().ListReservations(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)
	blockedSlotRepo.EXPECT().ListBlockedSlots(context.Background(), "doc-001", dayStart, dayEnd).
		Return([]*domain.BlockedSlot{weekly}, nil)

	uc := usecase.NewAvailabilityUsecase(doctorRepo, reservationRepo, blockedSlotRepo)
	result, err := uc.GetAvailability(context.Background(), "doc-001", date, domain.ReservationTypeFollowUp)
	require.NoError(t, err)

	noon := dayStart.Add(3 * time.Hour)
	for _, slot := range result.Slots {
		if !slot.StartsAt.Before(noon) && slot.StartsAt.Before(noon.Add(time.Hour)) {
			t.Errorf("slot %v falls within recurring block", slot.StartsAt)
		}
	}
	assert.Len(t, result.FreeRanges, 2)
}

func TestGetAvailability_BlockedSlotsError(t *testing.T) {
	date := nextMonday()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.UTC)

	doctorRepo := mocks.NewDoctorRepository(t)
	reservationRepo := mocks.NewReservationRepository(t)
	blockedSlotRepo := mocks.NewBlockedSlotRepository(t)
	doctorRepo.EXPECT().GetDoctor(context.Background(), "doc-001").Return(mondayDoctor(), nil)
	reservationRepo.EXPECT().ListReservations(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)
	blockedSlotRepo.EXPECT().ListBlockedSlots(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, assert.AnError)

	uc := usecase.NewAvailabilityUsecase(doctorRepo, reservationRepo, blockedSlotRepo)
	_, err := uc.GetAvailability(context.Background(), "doc-001", date, domain.ReservationTypeFollowUp)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGetAvailability_SlotLengthByType(t *testing.T) {
	date := nextMonday()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		resType   domain.ReservationType
		wantSlots int
		wantLen   time.Duration
	}{
		{"first visit", domain.ReservationTypeFirstVisit, 8, time.Hour},
		{"follow up", domain.ReservationTypeFollowUp, 16, 30 * time.Minute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doctorRepo := mocks.NewDoctorRepository(t)
			reservationRepo := mocks.NewReservationRepository(t)
			blockedSlotRepo := mocks.NewBlockedSlotRepository(t)
			doctorRepo.EXPECT().GetDoctor(context.Background(), "doc-001").Return(mondayDoctor(), nil)
			reservationRepo.EXPECT().ListReservations(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)
			blockedSlotRepo.EXPECT().ListBlockedSlots(context.Background(), "doc-001", dayStart, dayEnd).Return(nil, nil)

			uc := usecase.NewAvailabilityUsecase(doctorRepo, reservationRepo, blockedSlotRepo)
			result, err := uc.GetAvailability(context.Background(), "doc-001", date, tc.resType)
			require.NoError(t, err)

			require.Len(t, result.Slots, tc.wantSlots)
			for _, s := range result.Slots {
				assert.Equal(t, tc.wantLen, s.EndsAt.Sub(s.StartsAt))
			}
		})
	}
}
