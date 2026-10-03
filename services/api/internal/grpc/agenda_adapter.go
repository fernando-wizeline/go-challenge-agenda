package grpc

import (
	"context"

	agendav1 "go-challenge-agenda/gen/agenda/v1"
	"go-challenge-agenda/services/api/internal/domain"
)

// AgendaAdapter implements usecase.AgendaPort on top of the generated gRPC
// client, translating between protobuf messages and API domain types.
// Errors are returned as-is so gRPC status codes reach the HTTP error middleware.
type AgendaAdapter struct {
	client agendav1.AgendaServiceClient
}

func NewAgendaAdapter(client agendav1.AgendaServiceClient) *AgendaAdapter {
	return &AgendaAdapter{client: client}
}

func (a *AgendaAdapter) ListDoctors(ctx context.Context) ([]domain.DoctorResponse, error) {
	resp, err := a.client.ListDoctors(ctx, &agendav1.ListDoctorsRequest{})
	if err != nil {
		return nil, err
	}
	doctors := make([]domain.DoctorResponse, len(resp.Doctors))
	for i, d := range resp.Doctors {
		doctors[i] = protoDoctorToDTO(d)
	}
	return doctors, nil
}

func (a *AgendaAdapter) GetDoctor(ctx context.Context, id string) (*domain.DoctorResponse, error) {
	resp, err := a.client.GetDoctor(ctx, &agendav1.GetDoctorRequest{Id: id})
	if err != nil {
		return nil, err
	}
	dto := protoDoctorToDTO(resp.Doctor)
	return &dto, nil
}

func (a *AgendaAdapter) GetAvailability(ctx context.Context, doctorID, date, reservationType string) (*domain.AvailabilityResponse, error) {
	resp, err := a.client.GetAvailability(ctx, &agendav1.GetAvailabilityRequest{
		DoctorId:        doctorID,
		Date:            date,
		ReservationType: reservationTypeToProto(reservationType),
	})
	if err != nil {
		return nil, err
	}
	// Non-nil slices so empty results serialize as [] rather than null.
	result := &domain.AvailabilityResponse{
		Slots:      []domain.AvailableSlot{},
		FreeRanges: []domain.TimeRange{},
	}
	for _, s := range resp.Slots {
		result.Slots = append(result.Slots, domain.AvailableSlot{StartsAt: s.GetStartsAt(), EndsAt: s.GetEndsAt()})
	}
	for _, r := range resp.FreeRanges {
		result.FreeRanges = append(result.FreeRanges, domain.TimeRange{From: r.GetFrom(), To: r.GetTo()})
	}
	return result, nil
}

func (a *AgendaAdapter) CreateReservation(ctx context.Context, req *domain.CreateReservationRequest) (*domain.ReservationResponse, error) {
	resp, err := a.client.CreateReservation(ctx, &agendav1.CreateReservationRequest{
		DoctorId:     req.DoctorID,
		StartsAt:     req.StartsAt,
		Type:         reservationTypeToProto(req.Type),
		PatientId:    req.PatientID,
		PatientName:  req.PatientName,
		PatientPhone: req.PatientPhone,
		PatientEmail: req.PatientEmail,
	})
	if err != nil {
		return nil, err
	}
	return protoReservationToDTO(resp.Reservation), nil
}

func (a *AgendaAdapter) GetReservation(ctx context.Context, id string) (*domain.ReservationResponse, error) {
	resp, err := a.client.GetReservation(ctx, &agendav1.GetReservationRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return protoReservationToDTO(resp.Reservation), nil
}

func (a *AgendaAdapter) ListReservations(ctx context.Context, doctorID, from, to string) ([]domain.ReservationResponse, error) {
	resp, err := a.client.ListReservations(ctx, &agendav1.ListReservationsRequest{
		DoctorId: doctorID,
		From:     from,
		To:       to,
	})
	if err != nil {
		return nil, err
	}
	return reservationsToDTO(resp.Reservations), nil
}

func (a *AgendaAdapter) CancelReservation(ctx context.Context, id string) error {
	_, err := a.client.CancelReservation(ctx, &agendav1.CancelReservationRequest{Id: id})
	return err
}

func (a *AgendaAdapter) ListReservationsByUser(ctx context.Context, patientID string) ([]domain.ReservationResponse, error) {
	resp, err := a.client.ListReservationsByUser(ctx, &agendav1.ListReservationsByUserRequest{PatientId: patientID})
	if err != nil {
		return nil, err
	}
	return reservationsToDTO(resp.Reservations), nil
}

func (a *AgendaAdapter) ListPatients(ctx context.Context) ([]domain.UserResponse, error) {
	resp, err := a.client.ListPatients(ctx, &agendav1.ListPatientsRequest{})
	if err != nil {
		return nil, err
	}
	users := make([]domain.UserResponse, len(resp.Patients))
	for i, p := range resp.Patients {
		users[i] = protoPatientToDTO(p)
	}
	return users, nil
}

func (a *AgendaAdapter) GetPatient(ctx context.Context, id string) (*domain.UserResponse, error) {
	resp, err := a.client.GetPatient(ctx, &agendav1.GetPatientRequest{Id: id})
	if err != nil {
		return nil, err
	}
	dto := protoPatientToDTO(resp.Patient)
	return &dto, nil
}

func (a *AgendaAdapter) CreatePatient(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error) {
	resp, err := a.client.CreatePatient(ctx, &agendav1.CreatePatientRequest{
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	})
	if err != nil {
		return nil, err
	}
	dto := protoPatientToDTO(resp.Patient)
	return &dto, nil
}

func (a *AgendaAdapter) UpdatePatient(ctx context.Context, id string, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	resp, err := a.client.UpdatePatient(ctx, &agendav1.UpdatePatientRequest{
		Id:    id,
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	})
	if err != nil {
		return nil, err
	}
	dto := protoPatientToDTO(resp.Patient)
	return &dto, nil
}

func (a *AgendaAdapter) DeletePatient(ctx context.Context, id string) error {
	_, err := a.client.DeletePatient(ctx, &agendav1.DeletePatientRequest{Id: id})
	return err
}

// reservationTypeToProto maps the API string to the proto enum; anything other
// than "first_visit" is treated as a follow-up.
func reservationTypeToProto(t string) agendav1.ReservationType {
	if t == "first_visit" {
		return agendav1.ReservationType_RESERVATION_TYPE_FIRST_VISIT
	}
	return agendav1.ReservationType_RESERVATION_TYPE_FOLLOW_UP
}

func protoReservationToDTO(r *agendav1.Reservation) *domain.ReservationResponse {
	if r == nil {
		return nil
	}
	typeStr := "follow_up"
	if r.Type == agendav1.ReservationType_RESERVATION_TYPE_FIRST_VISIT {
		typeStr = "first_visit"
	}
	statusStr := "confirmed"
	if r.Status == agendav1.ReservationStatus_RESERVATION_STATUS_CANCELLED {
		statusStr = "cancelled"
	}
	return &domain.ReservationResponse{
		ID:        r.Id,
		DoctorID:  r.DoctorId,
		PatientID: r.PatientId,
		StartsAt:  r.StartsAt,
		EndsAt:    r.EndsAt,
		Type:      typeStr,
		Status:    statusStr,
	}
}

// reservationsToDTO converts a proto list, skipping nil entries; the result is never nil.
func reservationsToDTO(rs []*agendav1.Reservation) []domain.ReservationResponse {
	out := make([]domain.ReservationResponse, 0, len(rs))
	for _, r := range rs {
		if dto := protoReservationToDTO(r); dto != nil {
			out = append(out, *dto)
		}
	}
	return out
}

func protoDoctorToDTO(d *agendav1.Doctor) domain.DoctorResponse {
	resp := domain.DoctorResponse{ID: d.GetId(), Name: d.GetName(), Specialty: d.GetSpecialty()}
	for _, wh := range d.GetWorkingHours() {
		resp.WorkingHours = append(resp.WorkingHours, domain.WorkingHoursResp{
			Weekday: int(wh.Weekday),
			From:    wh.From,
			To:      wh.To,
		})
	}
	return resp
}

func protoPatientToDTO(p *agendav1.Patient) domain.UserResponse {
	return domain.UserResponse{ID: p.GetId(), Name: p.GetName(), Phone: p.GetPhone(), Email: p.GetEmail()}
}
