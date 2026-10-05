package observability

import (
	"errors"
	"net/http"

	"go-challenge-agenda/services/agenda/internal/domain"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Results recorded by agenda_reservation_attempts_total. Only slot_conflict,
// slot_blocked and patient_not_found are expected business rejections; "error"
// is the one that means something is broken.
const (
	ResultCreated         = "created"
	ResultSlotConflict    = "slot_conflict"
	ResultSlotBlocked     = "slot_blocked"
	ResultPatientNotFound = "patient_not_found"
	ResultError           = "error"
)

var reservationResults = []string{
	ResultCreated, ResultSlotConflict, ResultSlotBlocked, ResultPatientNotFound, ResultError,
}

// Metrics holds the Prometheus instruments for the availability and booking flows.
type Metrics struct {
	reservationAttempts  *prometheus.CounterVec
	availabilityDuration *prometheus.HistogramVec
}

// NewMetrics registers the instruments on reg. Every label value is initialised
// to zero so the series exist from startup.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		reservationAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "agenda_reservation_attempts_total",
			Help: "Reservation attempts by outcome.",
		}, []string{"result"}),
		availabilityDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "agenda_availability_duration_seconds",
			Help:    "Time spent computing doctor availability.",
			Buckets: prometheus.DefBuckets,
		}, []string{"result"}),
	}
	reg.MustRegister(m.reservationAttempts, m.availabilityDuration)

	for _, r := range reservationResults {
		m.reservationAttempts.WithLabelValues(r)
	}
	m.availabilityDuration.WithLabelValues("ok")
	m.availabilityDuration.WithLabelValues("error")
	return m
}

// Handler serves /metrics for a registry that also exports Go runtime and
// process metrics.
func Handler(reg *prometheus.Registry) http.Handler {
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	return mux
}

// reservationResult maps a Create outcome to a metric label.
func reservationResult(err error) string {
	switch {
	case err == nil:
		return ResultCreated
	case errors.Is(err, domain.ErrSlotBlocked):
		return ResultSlotBlocked
	case errors.Is(err, domain.ErrSlotNotAvailable):
		return ResultSlotConflict
	case errors.Is(err, domain.ErrPatientNotFound):
		return ResultPatientNotFound
	default:
		return ResultError
	}
}

// A nil *Metrics is valid and records nothing, so the decorators can be used
// with logging only.
func (m *Metrics) observeReservation(err error) {
	if m != nil {
		m.reservationAttempts.WithLabelValues(reservationResult(err)).Inc()
	}
}

func (m *Metrics) observeAvailability(seconds float64, err error) {
	if m == nil {
		return
	}
	result := "ok"
	if err != nil {
		result = "error"
	}
	m.availabilityDuration.WithLabelValues(result).Observe(seconds)
}
