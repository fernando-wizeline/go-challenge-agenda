package usecase

import (
	"context"

	"go-challenge-agenda/services/api/internal/domain"
)

// AgendaPort is what the API usecases need from the agenda service. It speaks
// API domain types only, so usecases know nothing about gRPC or protobuf; the
// transport lives in an adapter (see internal/grpc.AgendaAdapter).
//
// Implementations should return transport errors unchanged (or wrapped with %w):
// the HTTP error middleware derives the response status from the gRPC status.
type AgendaPort interface {
	// Doctors.
	ListDoctors(ctx context.Context) ([]domain.DoctorResponse, error)
	GetDoctor(ctx context.Context, id string) (*domain.DoctorResponse, error)

	// Availability. reservationType is "first_visit" or "follow_up".
	GetAvailability(ctx context.Context, doctorID, date, reservationType string) (*domain.AvailabilityResponse, error)

	// Reservations.
	CreateReservation(ctx context.Context, req *domain.CreateReservationRequest) (*domain.ReservationResponse, error)
	GetReservation(ctx context.Context, id string) (*domain.ReservationResponse, error)
	// ListReservations returns a doctor's reservations in [from, to] (RFC3339 strings).
	ListReservations(ctx context.Context, doctorID, from, to string) ([]domain.ReservationResponse, error)
	CancelReservation(ctx context.Context, id string) error
	ListReservationsByUser(ctx context.Context, patientID string) ([]domain.ReservationResponse, error)

	// Users (patients).
	ListPatients(ctx context.Context) ([]domain.UserResponse, error)
	GetPatient(ctx context.Context, id string) (*domain.UserResponse, error)
	CreatePatient(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error)
	// UpdatePatient replaces the patient's fields; callers pass the full merged values.
	UpdatePatient(ctx context.Context, id string, req *domain.UpdateUserRequest) (*domain.UserResponse, error)
	DeletePatient(ctx context.Context, id string) error
}
