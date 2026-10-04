package usecase

import (
	"context"
	"fmt"

	"go-challenge-agenda/services/api/internal/domain"
)

type AvailabilityUsecase struct {
	agenda AgendaPort
}

func NewAvailabilityUsecase(agenda AgendaPort) *AvailabilityUsecase {
	return &AvailabilityUsecase{agenda: agenda}
}

func (u *AvailabilityUsecase) GetAvailability(ctx context.Context, doctorID, date, resType string) (*domain.AvailabilityResponse, error) {
	resp, err := u.agenda.GetAvailability(ctx, doctorID, date, resType)
	if err != nil {
		return nil, fmt.Errorf("agenda.GetAvailability: %w", err)
	}
	return resp, nil
}
