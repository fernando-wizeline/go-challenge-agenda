package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type DoctorRepository interface {
	GetDoctor(ctx context.Context, id string) (*Doctor, error)
	ListDoctors(ctx context.Context) ([]*Doctor, error)
}

type PatientRepository interface {
	GetPatient(ctx context.Context, id string) (*Patient, error)
	GetPatientByPhone(ctx context.Context, phone string) (*Patient, error)
	CreatePatient(ctx context.Context, p *Patient) error
	ListPatients(ctx context.Context) ([]*Patient, error)
	UpdatePatient(ctx context.Context, p *Patient) error
	DeletePatient(ctx context.Context, id string) error
}

type ReservationRepository interface {
	CreateReservation(ctx context.Context, r *Reservation) error
	GetReservation(ctx context.Context, id string) (*Reservation, error)
	// ListReservations returns reservations for a doctor overlapping [from, to].
	ListReservations(ctx context.Context, doctorID string, from, to time.Time) ([]*Reservation, error)
	// ListReservationsByUser returns all reservations of a patient, ordered by start time.
	ListReservationsByUser(ctx context.Context, patientID string) ([]*Reservation, error)
	UpdateReservation(ctx context.Context, r *Reservation) error
	CancelReservation(ctx context.Context, id string) error
}

type BlockedSlotRepository interface {
	CreateBlockedSlot(ctx context.Context, b *BlockedSlot) error
	GetBlockedSlot(ctx context.Context, id string) (*BlockedSlot, error)
	ListBlockedSlots(ctx context.Context, doctorID string, from, to time.Time) ([]*BlockedSlot, error)
	DeleteBlockedSlot(ctx context.Context, id string) error
}

// ErrPatientNotFound is returned when a patient lookup finds no matching record.
var ErrPatientNotFound = errors.New("patient not found")

// ErrSlotNotAvailable is returned when a requested reservation overlaps an existing one.
var ErrSlotNotAvailable = errors.New("time slot not available")

// ErrSlotBlocked is returned when a requested reservation overlaps a blocked
// slot. It wraps ErrSlotNotAvailable, so errors.Is matches both.
var ErrSlotBlocked = fmt.Errorf("%w: blocked slot", ErrSlotNotAvailable)
