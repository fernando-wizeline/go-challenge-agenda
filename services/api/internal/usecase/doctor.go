package usecase

import (
	"context"
	"fmt"

	"go-challenge-agenda/services/api/internal/domain"
)

type DoctorUsecase struct {
	agenda AgendaPort
}

func NewDoctorUsecase(agenda AgendaPort) *DoctorUsecase {
	return &DoctorUsecase{agenda: agenda}
}

func (u *DoctorUsecase) List(ctx context.Context) ([]domain.DoctorResponse, error) {
	doctors, err := u.agenda.ListDoctors(ctx)
	if err != nil {
		return nil, fmt.Errorf("agenda.ListDoctors: %w", err)
	}
	return doctors, nil
}

func (u *DoctorUsecase) Get(ctx context.Context, id string) (*domain.DoctorResponse, error) {
	doctor, err := u.agenda.GetDoctor(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agenda.GetDoctor: %w", err)
	}
	return doctor, nil
}
