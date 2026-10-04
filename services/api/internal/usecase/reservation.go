package usecase

import (
	"context"
	"fmt"
	"time"

	"go-challenge-agenda/services/api/internal/domain"
)

type ReservationUsecase struct {
	agenda AgendaPort
}

func NewReservationUsecase(agenda AgendaPort) *ReservationUsecase {
	return &ReservationUsecase{agenda: agenda}
}

func (u *ReservationUsecase) Create(ctx context.Context, req *domain.CreateReservationRequest) (*domain.ReservationResponse, error) {
	// Validate starts_at
	if _, err := time.Parse(time.RFC3339, req.StartsAt); err != nil {
		return nil, fmt.Errorf("invalid starts_at: %w", err)
	}

	res, err := u.agenda.CreateReservation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("agenda.CreateReservation: %w", err)
	}
	return res, nil
}

func (u *ReservationUsecase) Get(ctx context.Context, id string) (*domain.ReservationResponse, error) {
	res, err := u.agenda.GetReservation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agenda.GetReservation: %w", err)
	}
	return res, nil
}

// List returns a doctor's reservations in [from, to]; both must be RFC3339.
func (u *ReservationUsecase) List(ctx context.Context, doctorID, from, to string) ([]domain.ReservationResponse, error) {
	list, err := u.agenda.ListReservations(ctx, doctorID, from, to)
	if err != nil {
		return nil, fmt.Errorf("agenda.ListReservations: %w", err)
	}
	return list, nil
}

func (u *ReservationUsecase) Cancel(ctx context.Context, id string) error {
	return u.agenda.CancelReservation(ctx, id)
}
