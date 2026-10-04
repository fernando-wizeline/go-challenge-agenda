package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"

	"github.com/google/uuid"
)

type ReservationUsecase struct {
	reservations domain.ReservationRepository
	patients     domain.PatientRepository
	blockedSlots domain.BlockedSlotRepository

	doctorLocks sync.Map
}

func (u *ReservationUsecase) lockDoctor(doctorID string) func() {
	m, _ := u.doctorLocks.LoadOrStore(doctorID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func NewReservationUsecase(
	reservations domain.ReservationRepository,
	patients domain.PatientRepository,
	blockedSlots domain.BlockedSlotRepository,
) *ReservationUsecase {
	return &ReservationUsecase{
		reservations: reservations,
		patients:     patients,
		blockedSlots: blockedSlots,
	}
}

type CreateReservationInput struct {
	DoctorID     string
	StartsAt     time.Time
	Type         domain.ReservationType
	PatientID    string // set if patient already exists
	PatientName  string
	PatientPhone string
	PatientEmail string
}

func (u *ReservationUsecase) Create(ctx context.Context, in CreateReservationInput) (*domain.Reservation, error) {
	patient, err := u.resolvePatient(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("resolve patient: %w", err)
	}

	endsAt := in.StartsAt.Add(in.Type.SlotDuration())

	// Adding a lock taking into consideration race conditions.
	unlock := u.lockDoctor(in.DoctorID)
	defer unlock()

	conflict, err := u.hasConflict(ctx, in.DoctorID, in.StartsAt, endsAt)
	if err != nil {
		return nil, fmt.Errorf("check conflict: %w", err)
	}
	if conflict {
		return nil, domain.ErrSlotNotAvailable
	}

	blocked, err := u.isBlocked(ctx, in.DoctorID, in.StartsAt, endsAt)
	if err != nil {
		return nil, fmt.Errorf("check blocked slots: %w", err)
	}
	if blocked {
		return nil, domain.ErrSlotNotAvailable
	}

	res := &domain.Reservation{
		ID:        uuid.NewString(),
		DoctorID:  in.DoctorID,
		PatientID: patient.ID,
		StartsAt:  in.StartsAt,
		EndsAt:    endsAt,
		Type:      in.Type,
		Status:    domain.ReservationStatus(domain.ReservationStatusConfirmed),
	}

	if err := u.reservations.CreateReservation(ctx, res); err != nil {
		return nil, fmt.Errorf("create reservation: %w", err)
	}
	return res, nil
}

// ListReservationsByUser returns all reservations of a patient. It returns an
// error wrapping domain.ErrPatientNotFound if the patient does not exist.
func (u *ReservationUsecase) ListReservationsByUser(ctx context.Context, patientID string) ([]*domain.Reservation, error) {
	if _, err := u.patients.GetPatient(ctx, patientID); err != nil {
		return nil, fmt.Errorf("get patient: %w", err)
	}
	return u.reservations.ListReservationsByUser(ctx, patientID)
}

func (u *ReservationUsecase) Cancel(ctx context.Context, id string) error {
	return u.reservations.CancelReservation(ctx, id)
}

func (u *ReservationUsecase) Get(ctx context.Context, id string) (*domain.Reservation, error) {
	return u.reservations.GetReservation(ctx, id)
}

func (u *ReservationUsecase) List(ctx context.Context, doctorID string, from, to time.Time) ([]*domain.Reservation, error) {
	return u.reservations.ListReservations(ctx, doctorID, from, to)
}

func (u *ReservationUsecase) resolvePatient(ctx context.Context, in CreateReservationInput) (*domain.Patient, error) {
	if in.PatientID != "" {
		return u.patients.GetPatient(ctx, in.PatientID)
	}
	existing, err := u.patients.GetPatientByPhone(ctx, in.PatientPhone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	p := &domain.Patient{
		ID:    uuid.NewString(),
		Name:  in.PatientName,
		Phone: in.PatientPhone,
		Email: in.PatientEmail,
	}
	if err := u.patients.CreatePatient(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// hasConflict checks if [startsAt, endsAt) overlaps any confirmed reservation.
// Intervals are half-open, so back-to-back reservations do not conflict.
func (u *ReservationUsecase) hasConflict(ctx context.Context, doctorID string, startsAt, endsAt time.Time) (bool, error) {
	// Use a wide window to retrieve candidates
	existing, err := u.reservations.ListReservations(ctx, doctorID, startsAt.Add(-24*time.Hour), endsAt.Add(24*time.Hour))
	if err != nil {
		return false, err
	}
	for _, r := range existing {
		if int(r.Status) == int(domain.ReservationStatusCancelled) {
			continue
		}
		if startsAt.Before(r.EndsAt) && endsAt.After(r.StartsAt) {
			return true, nil
		}
	}
	return false, nil
}

// isBlocked checks if [startsAt, endsAt) overlaps any occurrence of the doctor's
// blocked slots. Intervals are half-open, so a reservation may start exactly
// when a block ends or end exactly when one starts.
func (u *ReservationUsecase) isBlocked(ctx context.Context, doctorID string, startsAt, endsAt time.Time) (bool, error) {
	slots, err := u.blockedSlots.ListBlockedSlots(ctx, doctorID, startsAt, endsAt)
	if err != nil {
		return false, err
	}
	for _, b := range slots {
		for _, occ := range b.Occurrences(startsAt, endsAt) {
			if startsAt.Before(occ.EndsAt) && endsAt.After(occ.StartsAt) {
				return true, nil
			}
		}
	}
	return false, nil
}
