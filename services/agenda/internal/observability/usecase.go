package observability

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	agendagrpc "go-challenge-agenda/services/agenda/internal/grpc"
	"go-challenge-agenda/services/agenda/internal/usecase"
)

var (
	_ agendagrpc.AvailabilityService = (*ObservedAvailability)(nil)
	_ agendagrpc.ReservationService  = (*ObservedReservation)(nil)
)

// errLevel classifies an error: expected business rejections are warnings,
// anything else is an error. Only ids are logged, never patient details.
func errLevel(err error) slog.Level {
	if errors.Is(err, domain.ErrSlotNotAvailable) || errors.Is(err, domain.ErrPatientNotFound) {
		return slog.LevelWarn
	}
	return slog.LevelError
}

// ObservedAvailability logs and measures availability lookups and delegates to
// the wrapped service. metrics may be nil.
type ObservedAvailability struct {
	next    agendagrpc.AvailabilityService
	logger  *slog.Logger
	metrics *Metrics
}

func NewObservedAvailability(next agendagrpc.AvailabilityService, logger *slog.Logger, metrics *Metrics) *ObservedAvailability {
	return &ObservedAvailability{next: next, logger: logger, metrics: metrics}
}

func (l *ObservedAvailability) GetAvailability(ctx context.Context, doctorID string, date time.Time, resType domain.ReservationType) (*usecase.AvailabilityResult, error) {
	start := time.Now()
	res, err := l.next.GetAvailability(ctx, doctorID, date, resType)
	elapsed := time.Since(start)
	l.metrics.observeAvailability(elapsed.Seconds(), err)

	attrs := []any{
		"doctor_id", doctorID,
		"date", date.Format("2006-01-02"),
		"type", int(resType),
		"duration", time.Since(start),
	}
	if err != nil {
		l.logger.Log(ctx, errLevel(err), "availability lookup failed", append(attrs, "error", err)...)
		return res, err
	}
	l.logger.InfoContext(ctx, "availability computed",
		append(attrs, "slots", len(res.Slots), "free_ranges", len(res.FreeRanges))...)
	return res, nil
}

// ObservedReservation logs and measures the booking flow and delegates to the
// wrapped service. metrics may be nil.
type ObservedReservation struct {
	next    agendagrpc.ReservationService
	logger  *slog.Logger
	metrics *Metrics
}

func NewObservedReservation(next agendagrpc.ReservationService, logger *slog.Logger, metrics *Metrics) *ObservedReservation {
	return &ObservedReservation{next: next, logger: logger, metrics: metrics}
}

func (l *ObservedReservation) Create(ctx context.Context, in usecase.CreateReservationInput) (*domain.Reservation, error) {
	start := time.Now()
	res, err := l.next.Create(ctx, in)
	l.metrics.observeReservation(err)

	attrs := []any{
		"doctor_id", in.DoctorID,
		"patient_id", in.PatientID,
		"starts_at", in.StartsAt.UTC().Format(time.RFC3339),
		"type", int(in.Type),
		"duration", time.Since(start),
	}
	if err != nil {
		l.logger.Log(ctx, errLevel(err), "reservation rejected", append(attrs, "reason", err.Error(), "error", err)...)
		return res, err
	}
	l.logger.InfoContext(ctx, "reservation created",
		"reservation_id", res.ID, "doctor_id", res.DoctorID, "patient_id", res.PatientID,
		"starts_at", res.StartsAt.UTC().Format(time.RFC3339),
		"ends_at", res.EndsAt.UTC().Format(time.RFC3339),
		"type", int(res.Type), "duration", time.Since(start))
	return res, nil
}

func (l *ObservedReservation) Cancel(ctx context.Context, id string) error {
	err := l.next.Cancel(ctx, id)
	if err != nil {
		l.logger.Log(ctx, errLevel(err), "reservation cancel failed", "reservation_id", id, "error", err)
		return err
	}
	l.logger.InfoContext(ctx, "reservation cancelled", "reservation_id", id)
	return nil
}

func (l *ObservedReservation) Get(ctx context.Context, id string) (*domain.Reservation, error) {
	return l.next.Get(ctx, id)
}

func (l *ObservedReservation) List(ctx context.Context, doctorID string, from, to time.Time) ([]*domain.Reservation, error) {
	return l.next.List(ctx, doctorID, from, to)
}

func (l *ObservedReservation) ListReservationsByUser(ctx context.Context, patientID string) ([]*domain.Reservation, error) {
	return l.next.ListReservationsByUser(ctx, patientID)
}
