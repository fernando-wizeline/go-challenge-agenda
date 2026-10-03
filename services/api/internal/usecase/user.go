package usecase

import (
	"context"
	"fmt"

	agendav1 "go-challenge-agenda/gen/agenda/v1"
	"go-challenge-agenda/services/api/internal/domain"
)

type UserUsecase struct {
	agendaClient agendav1.AgendaServiceClient
}

func NewUserUsecase(client agendav1.AgendaServiceClient) *UserUsecase {
	return &UserUsecase{agendaClient: client}
}

func (u *UserUsecase) List(ctx context.Context) ([]domain.UserResponse, error) {
	resp, err := u.agendaClient.ListPatients(ctx, &agendav1.ListPatientsRequest{})
	if err != nil {
		return nil, fmt.Errorf("agenda.ListPatients: %w", err)
	}
	users := make([]domain.UserResponse, len(resp.Patients))
	for i, p := range resp.Patients {
		users[i] = protoPatientToDTO(p)
	}
	return users, nil
}

func (u *UserUsecase) Get(ctx context.Context, id string) (*domain.UserResponse, error) {
	resp, err := u.agendaClient.GetPatient(ctx, &agendav1.GetPatientRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("agenda.GetPatient: %w", err)
	}
	dto := protoPatientToDTO(resp.Patient)
	return &dto, nil
}

func (u *UserUsecase) Create(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error) {
	resp, err := u.agendaClient.CreatePatient(ctx, &agendav1.CreatePatientRequest{
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("agenda.CreatePatient: %w", err)
	}
	dto := protoPatientToDTO(resp.Patient)
	return &dto, nil
}

func (u *UserUsecase) Update(ctx context.Context, id string, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	// First fetch the existing patient to merge fields
	existing, err := u.agendaClient.GetPatient(ctx, &agendav1.GetPatientRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("agenda.GetPatient: %w", err)
	}

	name := existing.Patient.Name
	phone := existing.Patient.Phone
	email := existing.Patient.Email
	if req.Name != "" {
		name = req.Name
	}
	if req.Phone != "" {
		phone = req.Phone
	}
	if req.Email != "" {
		email = req.Email
	}

	resp, err := u.agendaClient.UpdatePatient(ctx, &agendav1.UpdatePatientRequest{
		Id: id, Name: name, Phone: phone, Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("agenda.UpdatePatient: %w", err)
	}
	dto := protoPatientToDTO(resp.Patient)
	return &dto, nil
}

func (u *UserUsecase) Delete(ctx context.Context, id string) error {
	_, err := u.agendaClient.DeletePatient(ctx, &agendav1.DeletePatientRequest{Id: id})
	return err
}

func (u *UserUsecase) ListReservations(ctx context.Context, id string) ([]domain.ReservationResponse, error) {
	resp, err := u.agendaClient.ListReservationsByUser(ctx, &agendav1.ListReservationsByUserRequest{PatientId: id})
	if err != nil {
		return nil, fmt.Errorf("agenda.ListReservationsByUser: %w", err)
	}
	reservations := make([]domain.ReservationResponse, len(resp.Reservations))
	for i, r := range resp.Reservations {
		reservations[i] = *protoReservationToDTO(r)
	}
	return reservations, nil
}

func protoPatientToDTO(p *agendav1.Patient) domain.UserResponse {
	return domain.UserResponse{ID: p.Id, Name: p.Name, Phone: p.Phone, Email: p.Email}
}
