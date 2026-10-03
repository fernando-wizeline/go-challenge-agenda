package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-challenge-agenda/services/api/internal/domain"
	apihttp "go-challenge-agenda/services/api/internal/http"
	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeAgenda is a usecase.AgendaPort that records calls and returns canned data.
type fakeAgenda struct {
	err error

	availabilityArgs []string // doctorID, date, type
	listArgs         []string // doctorID, from, to
}

var _ usecase.AgendaPort = (*fakeAgenda)(nil)

func (f *fakeAgenda) ListDoctors(context.Context) ([]domain.DoctorResponse, error) {
	return []domain.DoctorResponse{{ID: "d1"}}, f.err
}
func (f *fakeAgenda) GetDoctor(_ context.Context, id string) (*domain.DoctorResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.DoctorResponse{ID: id}, nil
}
func (f *fakeAgenda) GetAvailability(_ context.Context, doctorID, date, resType string) (*domain.AvailabilityResponse, error) {
	f.availabilityArgs = []string{doctorID, date, resType}
	return &domain.AvailabilityResponse{Slots: []domain.AvailableSlot{{StartsAt: "s", EndsAt: "e"}}}, f.err
}
func (f *fakeAgenda) CreateReservation(context.Context, *domain.CreateReservationRequest) (*domain.ReservationResponse, error) {
	return &domain.ReservationResponse{ID: "r1"}, f.err
}
func (f *fakeAgenda) GetReservation(_ context.Context, id string) (*domain.ReservationResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.ReservationResponse{ID: id}, nil
}
func (f *fakeAgenda) ListReservations(_ context.Context, doctorID, from, to string) ([]domain.ReservationResponse, error) {
	f.listArgs = []string{doctorID, from, to}
	return []domain.ReservationResponse{}, f.err
}
func (f *fakeAgenda) CancelReservation(context.Context, string) error { return f.err }
func (f *fakeAgenda) ListReservationsByUser(context.Context, string) ([]domain.ReservationResponse, error) {
	return nil, f.err
}
func (f *fakeAgenda) ListPatients(context.Context) ([]domain.UserResponse, error) { return nil, f.err }
func (f *fakeAgenda) GetPatient(context.Context, string) (*domain.UserResponse, error) {
	return &domain.UserResponse{}, f.err
}
func (f *fakeAgenda) CreatePatient(context.Context, *domain.CreateUserRequest) (*domain.UserResponse, error) {
	return &domain.UserResponse{}, f.err
}
func (f *fakeAgenda) UpdatePatient(context.Context, string, *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	return &domain.UserResponse{}, f.err
}
func (f *fakeAgenda) DeletePatient(context.Context, string) error { return f.err }

func do(t *testing.T, agenda usecase.AgendaPort, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	apihttp.NewRouter(agenda).ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestAvailability_Validation(t *testing.T) {
	for name, target := range map[string]string{
		"missing date": "/v1/doctors/d1/availability",
		"bad date":     "/v1/doctors/d1/availability?date=15-03-2024",
		"bad type":     "/v1/doctors/d1/availability?date=2024-03-15&type=checkup",
	} {
		t.Run(name, func(t *testing.T) {
			agenda := &fakeAgenda{}
			rec := do(t, agenda, http.MethodGet, target)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Nil(t, agenda.availabilityArgs, "agenda must not be called on invalid input")
		})
	}
}

func TestAvailability_PassesParamsAndDefaultsType(t *testing.T) {
	for target, wantType := range map[string]string{
		"/v1/doctors/d1/availability?date=2024-03-15":                  "first_visit",
		"/v1/doctors/d1/availability?date=2024-03-15&type=first_visit": "first_visit",
		"/v1/doctors/d1/availability?date=2024-03-15&type=follow_up":   "follow_up",
	} {
		agenda := &fakeAgenda{}
		rec := do(t, agenda, http.MethodGet, target)
		require.Equal(t, http.StatusOK, rec.Code, target)
		assert.Equal(t, []string{"d1", "2024-03-15", wantType}, agenda.availabilityArgs, target)
	}
}

func TestReservations_ListDelegatesAndReturnsEmptyArray(t *testing.T) {
	agenda := &fakeAgenda{}
	rec := do(t, agenda, http.MethodGet, "/v1/reservations?doctor_id=d1&from=2024-03-15T00:00:00Z&to=2024-03-16T00:00:00Z")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[]`, rec.Body.String())
	assert.Equal(t, []string{"d1", "2024-03-15T00:00:00Z", "2024-03-16T00:00:00Z"}, agenda.listArgs)
}

func TestReservations_ListRequiresFromAndTo(t *testing.T) {
	rec := do(t, &fakeAgenda{}, http.MethodGet, "/v1/reservations?doctor_id=d1")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandlers_MapGRPCStatusToHTTP(t *testing.T) {
	agenda := &fakeAgenda{err: status.Error(codes.NotFound, "not found")}

	assert.Equal(t, http.StatusNotFound, do(t, agenda, http.MethodGet, "/v1/doctors/nope").Code)
	assert.Equal(t, http.StatusNotFound, do(t, agenda, http.MethodGet, "/v1/reservations/nope").Code)

	agenda.err = status.Error(codes.AlreadyExists, "time slot not available")
	assert.Equal(t, http.StatusConflict, do(t, agenda, http.MethodGet, "/v1/doctors/d1/availability?date=2024-03-15").Code)
}
