package usecase_test

import (
	"context"
	"errors"
	"testing"

	"go-challenge-agenda/services/api/internal/domain"
	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeAgenda is an in-memory usecase.AgendaPort that records what it was asked to do.
type fakeAgenda struct {
	patient         *domain.UserResponse
	err             error
	updated         *domain.UpdateUserRequest
	createdReserved bool
}

var _ usecase.AgendaPort = (*fakeAgenda)(nil)

func (f *fakeAgenda) ListDoctors(context.Context) ([]domain.DoctorResponse, error) { return nil, f.err }
func (f *fakeAgenda) GetDoctor(context.Context, string) (*domain.DoctorResponse, error) {
	return &domain.DoctorResponse{ID: "d1"}, f.err
}
func (f *fakeAgenda) GetReservation(context.Context, string) (*domain.ReservationResponse, error) {
	return &domain.ReservationResponse{ID: "r1"}, f.err
}
func (f *fakeAgenda) ListReservations(context.Context, string, string, string) ([]domain.ReservationResponse, error) {
	return nil, f.err
}
func (f *fakeAgenda) GetAvailability(context.Context, string, string, string) (*domain.AvailabilityResponse, error) {
	return &domain.AvailabilityResponse{}, f.err
}
func (f *fakeAgenda) CreateReservation(context.Context, *domain.CreateReservationRequest) (*domain.ReservationResponse, error) {
	f.createdReserved = true
	return &domain.ReservationResponse{ID: "r1"}, f.err
}
func (f *fakeAgenda) CancelReservation(context.Context, string) error { return f.err }
func (f *fakeAgenda) ListReservationsByUser(context.Context, string) ([]domain.ReservationResponse, error) {
	return []domain.ReservationResponse{{ID: "r1"}}, f.err
}
func (f *fakeAgenda) ListPatients(context.Context) ([]domain.UserResponse, error) {
	return nil, f.err
}
func (f *fakeAgenda) GetPatient(context.Context, string) (*domain.UserResponse, error) {
	return f.patient, f.err
}
func (f *fakeAgenda) CreatePatient(context.Context, *domain.CreateUserRequest) (*domain.UserResponse, error) {
	return f.patient, f.err
}
func (f *fakeAgenda) UpdatePatient(_ context.Context, _ string, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	f.updated = req
	return f.patient, f.err
}
func (f *fakeAgenda) DeletePatient(context.Context, string) error { return f.err }

func TestUserUsecase_Update_MergesUnsetFields(t *testing.T) {
	agenda := &fakeAgenda{patient: &domain.UserResponse{ID: "u1", Name: "Old", Phone: "555-1", Email: "old@example.com"}}
	uc := usecase.NewUserUsecase(agenda)

	_, err := uc.Update(context.Background(), "u1", &domain.UpdateUserRequest{Name: "New"})

	require.NoError(t, err)
	assert.Equal(t, &domain.UpdateUserRequest{Name: "New", Phone: "555-1", Email: "old@example.com"}, agenda.updated)
}

func TestReservationUsecase_Create_InvalidStartsAtSkipsAgenda(t *testing.T) {
	agenda := &fakeAgenda{}
	uc := usecase.NewReservationUsecase(agenda)

	_, err := uc.Create(context.Background(), &domain.CreateReservationRequest{StartsAt: "not-a-date"})

	require.Error(t, err)
	assert.False(t, agenda.createdReserved, "agenda must not be called for an invalid starts_at")
}

// The HTTP error middleware derives the response code from the gRPC status, so
// usecases must keep it reachable when they wrap the port's error.
func TestUsecases_PreserveGRPCStatusOnError(t *testing.T) {
	agenda := &fakeAgenda{err: status.Error(codes.NotFound, "patient not found")}

	_, err := usecase.NewUserUsecase(agenda).ListReservations(context.Background(), "missing")

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.True(t, errors.Is(err, agenda.err))
}

func TestUserUsecase_ListReservations(t *testing.T) {
	uc := usecase.NewUserUsecase(&fakeAgenda{})

	got, err := uc.ListReservations(context.Background(), "u1")

	require.NoError(t, err)
	assert.Equal(t, []domain.ReservationResponse{{ID: "r1"}}, got)
}
