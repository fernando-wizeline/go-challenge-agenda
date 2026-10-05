package observability_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/observability"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func counter(t *testing.T, reg *prometheus.Registry, result string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != "agenda_reservation_attempts_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if m.GetLabel()[0].GetValue() == result {
				return m.GetCounter().GetValue()
			}
		}
	}
	t.Fatalf("no series for result=%s", result)
	return 0
}

func TestReservationAttemptsCounter(t *testing.T) {
	ctx := context.Background()
	in := usecase.CreateReservationInput{DoctorID: "doc-001", StartsAt: time.Now()}

	for result, err := range map[string]error{
		observability.ResultCreated:         nil,
		observability.ResultSlotConflict:    domain.ErrSlotNotAvailable,
		observability.ResultSlotBlocked:     domain.ErrSlotBlocked,
		observability.ResultPatientNotFound: fmt.Errorf("resolve patient: %w", domain.ErrPatientNotFound),
		observability.ResultError:           errors.New("db down"),
	} {
		t.Run(result, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			buf, _ := newLogger(t)
			l := observability.NewObservedReservation(
				fakeRes{res: &domain.Reservation{}, err: err},
				observability.NewLogger(buf, "info", "json"),
				observability.NewMetrics(reg),
			)

			_, _ = l.Create(ctx, in)

			assert.Equal(t, 1.0, counter(t, reg, result))
			for _, other := range []string{"created", "slot_conflict", "slot_blocked", "patient_not_found", "error"} {
				if other != result {
					assert.Equal(t, 0.0, counter(t, reg, other), other)
				}
			}
		})
	}
}

func TestAvailabilityDurationHistogram(t *testing.T) {
	ctx := context.Background()
	reg := prometheus.NewRegistry()
	buf, _ := newLogger(t)
	logger := observability.NewLogger(buf, "info", "json")
	m := observability.NewMetrics(reg)

	ok := observability.NewObservedAvailability(fakeAvail{res: &usecase.AvailabilityResult{}}, logger, m)
	bad := observability.NewObservedAvailability(fakeAvail{err: errors.New("boom")}, logger, m)
	_, _ = ok.GetAvailability(ctx, "doc-001", time.Now(), domain.ReservationTypeFollowUp)
	_, _ = ok.GetAvailability(ctx, "doc-001", time.Now(), domain.ReservationTypeFollowUp)
	_, _ = bad.GetAvailability(ctx, "doc-001", time.Now(), domain.ReservationTypeFollowUp)

	count, err := testutil.GatherAndCount(reg, "agenda_availability_duration_seconds")
	require.NoError(t, err)
	assert.Equal(t, 2, count) // one histogram series per result label

	mfs, err := reg.Gather()
	require.NoError(t, err)
	got := map[string]uint64{}
	for _, mf := range mfs {
		if mf.GetName() == "agenda_availability_duration_seconds" {
			for _, mm := range mf.GetMetric() {
				got[mm.GetLabel()[0].GetValue()] = mm.GetHistogram().GetSampleCount()
			}
		}
	}
	assert.Equal(t, map[string]uint64{"ok": 2, "error": 1}, got)
}

func TestMetricsHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	observability.NewMetrics(reg)

	rec := httptest.NewRecorder()
	observability.Handler(reg).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))

	assert.Equal(t, 200, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `agenda_reservation_attempts_total{result="created"} 0`)
	assert.Contains(t, body, "agenda_availability_duration_seconds")
	assert.True(t, strings.Contains(body, "go_goroutines"), "Go runtime metrics should be exported")
}

func TestNilMetricsIsSafe(t *testing.T) {
	buf, _ := newLogger(t)
	l := observability.NewObservedReservation(fakeRes{res: &domain.Reservation{}}, observability.NewLogger(buf, "info", "json"), nil)
	_, err := l.Create(context.Background(), usecase.CreateReservationInput{})
	assert.NoError(t, err)
}
