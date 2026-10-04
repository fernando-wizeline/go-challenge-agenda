package usecase_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/stretchr/testify/assert"
)

// concurrentPatientRepo is a thread-safe in-memory patient repo for race tests.
type concurrentPatientRepo struct {
	mu       sync.Mutex
	patients []*domain.Patient
}

func (r *concurrentPatientRepo) GetPatient(_ context.Context, id string) (*domain.Patient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.patients {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

func (r *concurrentPatientRepo) GetPatientByPhone(_ context.Context, phone string) (*domain.Patient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.patients {
		if p.Phone == phone {
			return p, nil
		}
	}
	return nil, nil
}

func (r *concurrentPatientRepo) CreatePatient(_ context.Context, p *domain.Patient) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.patients = append(r.patients, p)
	return nil
}

func (r *concurrentPatientRepo) ListPatients(_ context.Context) ([]*domain.Patient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.patients, nil
}

func (r *concurrentPatientRepo) UpdatePatient(_ context.Context, p *domain.Patient) error {
	return nil
}

func (r *concurrentPatientRepo) DeletePatient(_ context.Context, _ string) error { return nil }

// concurrentReservationRepo is a thread-safe in-memory reservation repo for race tests.
type concurrentReservationRepo struct {
	mu           sync.Mutex
	reservations []*domain.Reservation
}

func (r *concurrentReservationRepo) CreateReservation(_ context.Context, res *domain.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservations = append(r.reservations, res)
	return nil
}

func (r *concurrentReservationRepo) GetReservation(_ context.Context, id string) (*domain.Reservation, error) {
	return nil, nil
}

func (r *concurrentReservationRepo) ListReservations(_ context.Context, doctorID string, from, to time.Time) ([]*domain.Reservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.Reservation
	for _, res := range r.reservations {
		if res.StartsAt.Before(to) && res.EndsAt.After(from) {
			result = append(result, res)
		}
	}
	return result, nil
}

func (r *concurrentReservationRepo) ListReservationsByUser(_ context.Context, patientID string) ([]*domain.Reservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.Reservation
	for _, res := range r.reservations {
		if res.PatientID == patientID {
			result = append(result, res)
		}
	}
	return result, nil
}

func (r *concurrentReservationRepo) UpdateReservation(_ context.Context, _ *domain.Reservation) error {
	return nil
}

func (r *concurrentReservationRepo) CancelReservation(_ context.Context, _ string) error {
	return nil
}

// TestConcurrentReservationCreation spawns N goroutines all trying to book the same
// time slot simultaneously. Only one should succeed; all others should receive a
// conflict error. Run with: go test -race ./...
//
// NOTE: this test is likely to expose the incomplete conflict-check bug — multiple
// bookings may succeed when they shouldn't.
func TestConcurrentReservationCreation(t *testing.T) {
	const goroutines = 20
	slot := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)

	resRepo := &concurrentReservationRepo{}
	patRepo := &concurrentPatientRepo{}
	uc := usecase.NewReservationUsecase(resRepo, patRepo, &fakeEmptyBlockedSlotRepo{})

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		success int
		errors  int
	)

	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			_, err := uc.Create(context.Background(), usecase.CreateReservationInput{
				DoctorID:     "doc-001",
				StartsAt:     slot,
				Type:         domain.ReservationTypeFollowUp,
				PatientPhone: fmt.Sprintf("555-%04d", i),
				PatientName:  fmt.Sprintf("Patient %d", i),
				PatientEmail: fmt.Sprintf("p%d@example.com", i),
			})
			mu.Lock()
			if err != nil {
				errors++
			} else {
				success++
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	t.Logf("concurrent bookings: %d success, %d conflict errors", success, errors)
	assert.Equal(t, 1, success, "exactly one booking should succeed for the same slot")
	assert.Equal(t, goroutines-1, errors, "all other concurrent bookings should be rejected")
}

// TestConcurrentDistinctSlots verifies that concurrent bookings for different
// time slots all succeed without interfering with each other.
func TestConcurrentDistinctSlots(t *testing.T) {
	const goroutines = 10
	base := time.Date(2025, 6, 2, 9, 0, 0, 0, time.UTC)

	resRepo := &concurrentReservationRepo{}
	patRepo := &concurrentPatientRepo{}
	uc := usecase.NewReservationUsecase(resRepo, patRepo, &fakeEmptyBlockedSlotRepo{})

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		errors []error
	)

	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			slot := base.Add(time.Duration(i) * 30 * time.Minute)
			_, err := uc.Create(context.Background(), usecase.CreateReservationInput{
				DoctorID:     "doc-001",
				StartsAt:     slot,
				Type:         domain.ReservationTypeFollowUp,
				PatientPhone: fmt.Sprintf("555-%04d", i),
				PatientName:  fmt.Sprintf("Patient %d", i),
				PatientEmail: fmt.Sprintf("p%d@example.com", i),
			})
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	assert.Empty(t, errors, "all distinct-slot bookings should succeed concurrently")
}
