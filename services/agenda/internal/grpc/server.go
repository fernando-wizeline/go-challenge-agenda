package grpc

import (
	"context"
	"errors"
	"time"

	agendav1 "go-challenge-agenda/gen/agenda/v1"
	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AvailabilityService and ReservationService are the usecase methods the server
// needs. Keeping them as interfaces lets cross-cutting concerns such as logging
// wrap the usecases without touching them.
type AvailabilityService interface {
	GetAvailability(ctx context.Context, doctorID string, date time.Time, resType domain.ReservationType) (*usecase.AvailabilityResult, error)
}

type ReservationService interface {
	Create(ctx context.Context, in usecase.CreateReservationInput) (*domain.Reservation, error)
	Get(ctx context.Context, id string) (*domain.Reservation, error)
	List(ctx context.Context, doctorID string, from, to time.Time) ([]*domain.Reservation, error)
	ListReservationsByUser(ctx context.Context, patientID string) ([]*domain.Reservation, error)
	Cancel(ctx context.Context, id string) error
}

type Server struct {
	agendav1.UnimplementedAgendaServiceServer
	doctors      domain.DoctorRepository
	availability AvailabilityService
	reservations ReservationService
	blockedSlots *usecase.BlockedSlotUsecase
	patients     domain.PatientRepository
}

func NewServer(
	doctors domain.DoctorRepository,
	availability AvailabilityService,
	reservations ReservationService,
	blockedSlots *usecase.BlockedSlotUsecase,
	patients domain.PatientRepository,
) *Server {
	return &Server{
		doctors:      doctors,
		availability: availability,
		reservations: reservations,
		blockedSlots: blockedSlots,
		patients:     patients,
	}
}

func (s *Server) GetDoctor(ctx context.Context, req *agendav1.GetDoctorRequest) (*agendav1.GetDoctorResponse, error) {
	doc, err := s.doctors.GetDoctor(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "doctor not found: %v", err)
	}
	return &agendav1.GetDoctorResponse{Doctor: domainDoctorToProto(doc)}, nil
}

func (s *Server) ListDoctors(ctx context.Context, _ *agendav1.ListDoctorsRequest) (*agendav1.ListDoctorsResponse, error) {
	docs, err := s.doctors.ListDoctors(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list doctors: %v", err)
	}
	pb := make([]*agendav1.Doctor, len(docs))
	for i, d := range docs {
		pb[i] = domainDoctorToProto(d)
	}
	return &agendav1.ListDoctorsResponse{Doctors: pb}, nil
}

func (s *Server) GetAvailability(ctx context.Context, req *agendav1.GetAvailabilityRequest) (*agendav1.GetAvailabilityResponse, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid date: %v", err)
	}

	resType := domain.ReservationType(req.ReservationType)
	result, err := s.availability.GetAvailability(ctx, req.DoctorId, date, resType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get availability: %v", err)
	}

	resp := &agendav1.GetAvailabilityResponse{}
	for _, sl := range result.Slots {
		resp.Slots = append(resp.Slots, &agendav1.AvailableSlot{
			StartsAt: sl.StartsAt.UTC().Format(time.RFC3339),
			EndsAt:   sl.EndsAt.UTC().Format(time.RFC3339),
		})
	}
	for _, fr := range result.FreeRanges {
		resp.FreeRanges = append(resp.FreeRanges, &agendav1.TimeRange{
			From: fr[0].UTC().Format(time.RFC3339),
			To:   fr[1].UTC().Format(time.RFC3339),
		})
	}
	return resp, nil
}

func (s *Server) CreateReservation(ctx context.Context, req *agendav1.CreateReservationRequest) (*agendav1.CreateReservationResponse, error) {
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid starts_at: %v", err)
	}

	res, err := s.reservations.Create(ctx, usecase.CreateReservationInput{
		DoctorID:     req.DoctorId,
		StartsAt:     startsAt,
		Type:         domain.ReservationType(req.Type),
		PatientID:    req.PatientId,
		PatientName:  req.PatientName,
		PatientPhone: req.PatientPhone,
		PatientEmail: req.PatientEmail,
	})
	if errors.Is(err, domain.ErrSlotNotAvailable) {
		return nil, status.Errorf(codes.AlreadyExists, "%v", err)
	}
	if errors.Is(err, domain.ErrPatientNotFound) {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create reservation: %v", err)
	}
	return &agendav1.CreateReservationResponse{Reservation: domainReservationToProto(res)}, nil
}

func (s *Server) GetReservation(ctx context.Context, req *agendav1.GetReservationRequest) (*agendav1.GetReservationResponse, error) {
	res, err := s.reservations.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return &agendav1.GetReservationResponse{Reservation: domainReservationToProto(res)}, nil
}

func (s *Server) ListReservations(ctx context.Context, req *agendav1.ListReservationsRequest) (*agendav1.ListReservationsResponse, error) {
	from, err := time.Parse(time.RFC3339, req.From)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid from: %v", err)
	}
	to, err := time.Parse(time.RFC3339, req.To)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid to: %v", err)
	}
	list, err := s.reservations.List(ctx, req.DoctorId, from, to)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	pb := make([]*agendav1.Reservation, len(list))
	for i, r := range list {
		pb[i] = domainReservationToProto(r)
	}
	return &agendav1.ListReservationsResponse{Reservations: pb}, nil
}

func (s *Server) ListReservationsByUser(ctx context.Context, req *agendav1.ListReservationsByUserRequest) (*agendav1.ListReservationsResponse, error) {
	if req.PatientId == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id is required")
	}
	list, err := s.reservations.ListReservationsByUser(ctx, req.PatientId)
	if errors.Is(err, domain.ErrPatientNotFound) {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	pb := make([]*agendav1.Reservation, len(list))
	for i, r := range list {
		pb[i] = domainReservationToProto(r)
	}
	return &agendav1.ListReservationsResponse{Reservations: pb}, nil
}

func (s *Server) CancelReservation(ctx context.Context, req *agendav1.CancelReservationRequest) (*agendav1.CancelReservationResponse, error) {
	if err := s.reservations.Cancel(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &agendav1.CancelReservationResponse{}, nil
}

func (s *Server) CreateBlockedSlot(ctx context.Context, req *agendav1.CreateBlockedSlotRequest) (*agendav1.CreateBlockedSlotResponse, error) {
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid starts_at: %v", err)
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ends_at: %v", err)
	}

	b := &domain.BlockedSlot{
		DoctorID:       req.DoctorId,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		Reason:         req.Reason,
		RecurrenceType: domain.RecurrenceType(req.RecurrenceType),
	}
	if req.RecurrenceUntil != "" {
		t, err := time.Parse(time.RFC3339, req.RecurrenceUntil)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid recurrence_until: %v", err)
		}
		b.RecurrenceUntil = &t
	}

	created, err := s.blockedSlots.Create(ctx, b)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &agendav1.CreateBlockedSlotResponse{BlockedSlot: domainBlockedSlotToProto(created)}, nil
}

func (s *Server) ListBlockedSlots(ctx context.Context, req *agendav1.ListBlockedSlotsRequest) (*agendav1.ListBlockedSlotsResponse, error) {
	from, _ := time.Parse(time.RFC3339, req.From)
	to, _ := time.Parse(time.RFC3339, req.To)
	list, err := s.blockedSlots.List(ctx, req.DoctorId, from, to)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	pb := make([]*agendav1.BlockedSlot, len(list))
	for i, b := range list {
		pb[i] = domainBlockedSlotToProto(b)
	}
	return &agendav1.ListBlockedSlotsResponse{BlockedSlots: pb}, nil
}

func (s *Server) DeleteBlockedSlot(ctx context.Context, req *agendav1.DeleteBlockedSlotRequest) (*agendav1.DeleteBlockedSlotResponse, error) {
	if err := s.blockedSlots.Delete(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &agendav1.DeleteBlockedSlotResponse{}, nil
}

func (s *Server) GetPatient(ctx context.Context, req *agendav1.GetPatientRequest) (*agendav1.GetPatientResponse, error) {
	p, err := s.patients.GetPatient(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return &agendav1.GetPatientResponse{Patient: domainPatientToProto(p)}, nil
}

func (s *Server) ListPatients(ctx context.Context, _ *agendav1.ListPatientsRequest) (*agendav1.ListPatientsResponse, error) {
	ps, err := s.patients.ListPatients(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	pb := make([]*agendav1.Patient, len(ps))
	for i, p := range ps {
		pb[i] = domainPatientToProto(p)
	}
	return &agendav1.ListPatientsResponse{Patients: pb}, nil
}

func (s *Server) CreatePatient(ctx context.Context, req *agendav1.CreatePatientRequest) (*agendav1.CreatePatientResponse, error) {
	p := &domain.Patient{
		ID:    uuid.NewString(),
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	}
	if err := s.patients.CreatePatient(ctx, p); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &agendav1.CreatePatientResponse{Patient: domainPatientToProto(p)}, nil
}

func (s *Server) UpdatePatient(ctx context.Context, req *agendav1.UpdatePatientRequest) (*agendav1.UpdatePatientResponse, error) {
	p := &domain.Patient{
		ID:    req.Id,
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	}
	if err := s.patients.UpdatePatient(ctx, p); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &agendav1.UpdatePatientResponse{Patient: domainPatientToProto(p)}, nil
}

func (s *Server) DeletePatient(ctx context.Context, req *agendav1.DeletePatientRequest) (*agendav1.DeletePatientResponse, error) {
	if err := s.patients.DeletePatient(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &agendav1.DeletePatientResponse{}, nil
}

func domainPatientToProto(p *domain.Patient) *agendav1.Patient {
	return &agendav1.Patient{Id: p.ID, Name: p.Name, Phone: p.Phone, Email: p.Email}
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func domainDoctorToProto(d *domain.Doctor) *agendav1.Doctor {
	pb := &agendav1.Doctor{Id: d.ID, Name: d.Name, Specialty: d.Specialty}
	for _, wh := range d.WorkingHours {
		pb.WorkingHours = append(pb.WorkingHours, &agendav1.WorkingHours{
			Weekday: int32(wh.Weekday),
			From:    wh.From,
			To:      wh.To,
		})
	}
	return pb
}

func domainReservationToProto(r *domain.Reservation) *agendav1.Reservation {
	return &agendav1.Reservation{
		Id:        r.ID,
		DoctorId:  r.DoctorID,
		PatientId: r.PatientID,
		StartsAt:  r.StartsAt.UTC().Format(time.RFC3339),
		EndsAt:    r.EndsAt.UTC().Format(time.RFC3339),
		Type:      agendav1.ReservationType(r.Type),
		Status:    agendav1.ReservationStatus(r.Status),
	}
}

func domainBlockedSlotToProto(b *domain.BlockedSlot) *agendav1.BlockedSlot {
	pb := &agendav1.BlockedSlot{
		Id:             b.ID,
		DoctorId:       b.DoctorID,
		StartsAt:       b.StartsAt.UTC().Format(time.RFC3339),
		EndsAt:         b.EndsAt.UTC().Format(time.RFC3339),
		Reason:         b.Reason,
		RecurrenceType: agendav1.RecurrenceType(b.RecurrenceType),
	}
	if b.RecurrenceUntil != nil {
		pb.RecurrenceUntil = b.RecurrenceUntil.UTC().Format(time.RFC3339)
	}
	return pb
}
