package usecase_test

import (
	"context"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/usecase"
)

// fakeStaticDoctorRepo is a minimal repo returning a fixed doctor for benchmarks.
type fakeStaticDoctorRepo struct{ doctor *domain.Doctor }

func (f *fakeStaticDoctorRepo) GetDoctor(_ context.Context, _ string) (*domain.Doctor, error) {
	return f.doctor, nil
}
func (f *fakeStaticDoctorRepo) ListDoctors(_ context.Context) ([]*domain.Doctor, error) {
	return []*domain.Doctor{f.doctor}, nil
}

// fakeEmptyReservationRepo returns no reservations.
type fakeEmptyReservationRepo struct{}

func (f *fakeEmptyReservationRepo) CreateReservation(_ context.Context, _ *domain.Reservation) error {
	return nil
}
func (f *fakeEmptyReservationRepo) GetReservation(_ context.Context, _ string) (*domain.Reservation, error) {
	return nil, nil
}
func (f *fakeEmptyReservationRepo) ListReservations(_ context.Context, _ string, _, _ time.Time) ([]*domain.Reservation, error) {
	return nil, nil
}
func (f *fakeEmptyReservationRepo) ListReservationsByUser(_ context.Context, _ string) ([]*domain.Reservation, error) {
	return nil, nil
}
func (f *fakeEmptyReservationRepo) UpdateReservation(_ context.Context, _ *domain.Reservation) error {
	return nil
}
func (f *fakeEmptyReservationRepo) CancelReservation(_ context.Context, _ string) error { return nil }

// fakeEmptyBlockedSlotRepo returns no blocked slots.
type fakeEmptyBlockedSlotRepo struct{}

func (f *fakeEmptyBlockedSlotRepo) CreateBlockedSlot(_ context.Context, _ *domain.BlockedSlot) error {
	return nil
}
func (f *fakeEmptyBlockedSlotRepo) GetBlockedSlot(_ context.Context, _ string) (*domain.BlockedSlot, error) {
	return nil, nil
}
func (f *fakeEmptyBlockedSlotRepo) ListBlockedSlots(_ context.Context, _ string, _, _ time.Time) ([]*domain.BlockedSlot, error) {
	return nil, nil
}
func (f *fakeEmptyBlockedSlotRepo) DeleteBlockedSlot(_ context.Context, _ string) error { return nil }

func BenchmarkGetAvailability_EmptyDay(b *testing.B) {
	uc := usecase.NewAvailabilityUsecase(
		&fakeStaticDoctorRepo{doctor: mondayDoctor()},
		&fakeEmptyReservationRepo{},
		&fakeEmptyBlockedSlotRepo{},
	)
	date := nextMonday()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uc.GetAvailability(ctx, "doc-001", date, domain.ReservationTypeFollowUp)
	}
}

func BenchmarkGetAvailability_FullDay(b *testing.B) {
	date := nextMonday()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)

	// Pre-fill with 16 back-to-back 30-min reservations (full 8hr day)
	var reservations []*domain.Reservation
	for i := range 16 {
		start := dayStart.Add(time.Duration(i) * 30 * time.Minute)
		reservations = append(reservations, &domain.Reservation{
			StartsAt: start,
			EndsAt:   start.Add(30 * time.Minute),
			Status:   domain.ReservationStatus(domain.ReservationStatusConfirmed),
		})
	}

	fullRepo := &fakeListReservationRepo{reservations: reservations}
	uc := usecase.NewAvailabilityUsecase(
		&fakeStaticDoctorRepo{doctor: mondayDoctor()},
		fullRepo,
		&fakeEmptyBlockedSlotRepo{},
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uc.GetAvailability(ctx, "doc-001", date, domain.ReservationTypeFollowUp)
	}
}

type fakeListReservationRepo struct{ reservations []*domain.Reservation }

func (f *fakeListReservationRepo) CreateReservation(_ context.Context, _ *domain.Reservation) error {
	return nil
}
func (f *fakeListReservationRepo) GetReservation(_ context.Context, _ string) (*domain.Reservation, error) {
	return nil, nil
}
func (f *fakeListReservationRepo) ListReservations(_ context.Context, _ string, _, _ time.Time) ([]*domain.Reservation, error) {
	return f.reservations, nil
}
func (f *fakeListReservationRepo) ListReservationsByUser(_ context.Context, _ string) ([]*domain.Reservation, error) {
	return nil, nil
}
func (f *fakeListReservationRepo) UpdateReservation(_ context.Context, _ *domain.Reservation) error {
	return nil
}
func (f *fakeListReservationRepo) CancelReservation(_ context.Context, _ string) error { return nil }
