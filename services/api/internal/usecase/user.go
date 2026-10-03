package usecase

import (
	"context"
	"fmt"

	"go-challenge-agenda/services/api/internal/domain"
)

type UserUsecase struct {
	agenda AgendaPort
}

func NewUserUsecase(agenda AgendaPort) *UserUsecase {
	return &UserUsecase{agenda: agenda}
}

func (u *UserUsecase) List(ctx context.Context) ([]domain.UserResponse, error) {
	users, err := u.agenda.ListPatients(ctx)
	if err != nil {
		return nil, fmt.Errorf("agenda.ListPatients: %w", err)
	}
	return users, nil
}

func (u *UserUsecase) Get(ctx context.Context, id string) (*domain.UserResponse, error) {
	user, err := u.agenda.GetPatient(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agenda.GetPatient: %w", err)
	}
	return user, nil
}

func (u *UserUsecase) Create(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error) {
	user, err := u.agenda.CreatePatient(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("agenda.CreatePatient: %w", err)
	}
	return user, nil
}

func (u *UserUsecase) Update(ctx context.Context, id string, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	// First fetch the existing patient to merge fields
	existing, err := u.agenda.GetPatient(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agenda.GetPatient: %w", err)
	}

	merged := domain.UpdateUserRequest{Name: existing.Name, Phone: existing.Phone, Email: existing.Email}
	if req.Name != "" {
		merged.Name = req.Name
	}
	if req.Phone != "" {
		merged.Phone = req.Phone
	}
	if req.Email != "" {
		merged.Email = req.Email
	}

	user, err := u.agenda.UpdatePatient(ctx, id, &merged)
	if err != nil {
		return nil, fmt.Errorf("agenda.UpdatePatient: %w", err)
	}
	return user, nil
}

func (u *UserUsecase) Delete(ctx context.Context, id string) error {
	return u.agenda.DeletePatient(ctx, id)
}

func (u *UserUsecase) ListReservations(ctx context.Context, id string) ([]domain.ReservationResponse, error) {
	reservations, err := u.agenda.ListReservationsByUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agenda.ListReservationsByUser: %w", err)
	}
	return reservations, nil
}
