package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/observability"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newLogger(t *testing.T) (*bytes.Buffer, func() []map[string]any) {
	t.Helper()
	buf := &bytes.Buffer{}
	return buf, func() []map[string]any {
		var out []map[string]any
		dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
		for dec.More() {
			var m map[string]any
			require.NoError(t, dec.Decode(&m))
			out = append(out, m)
		}
		return out
	}
}

type fakeAvail struct {
	res *usecase.AvailabilityResult
	err error
}

func (f fakeAvail) GetAvailability(context.Context, string, time.Time, domain.ReservationType) (*usecase.AvailabilityResult, error) {
	return f.res, f.err
}

type fakeRes struct {
	res *domain.Reservation
	err error
}

func (f fakeRes) Create(context.Context, usecase.CreateReservationInput) (*domain.Reservation, error) {
	return f.res, f.err
}
func (f fakeRes) Get(context.Context, string) (*domain.Reservation, error) { return nil, nil }
func (f fakeRes) List(context.Context, string, time.Time, time.Time) ([]*domain.Reservation, error) {
	return nil, nil
}
func (f fakeRes) ListReservationsByUser(context.Context, string) ([]*domain.Reservation, error) {
	return nil, nil
}
func (f fakeRes) Cancel(context.Context, string) error { return f.err }

func TestNewLogger_LevelAndFormat(t *testing.T) {
	var buf bytes.Buffer
	observability.NewLogger(&buf, "warn", "json").Info("hidden")
	observability.NewLogger(&buf, "warn", "json").Warn("shown")
	assert.NotContains(t, buf.String(), "hidden")
	assert.Contains(t, buf.String(), `"msg":"shown"`)

	buf.Reset()
	observability.NewLogger(&buf, "bogus", "text").Info("fallback")
	assert.Contains(t, buf.String(), "msg=fallback") // bad level -> info, text format
}

func TestObservedAvailability(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		buf, records := newLogger(t)
		want := &usecase.AvailabilityResult{Slots: make([]domain.Reservation, 3), FreeRanges: make([][2]time.Time, 2)}
		l := observability.NewObservedAvailability(fakeAvail{res: want}, observability.NewLogger(buf, "info", "json"), nil)

		got, err := l.GetAvailability(ctx, "doc-001", date, domain.ReservationTypeFirstVisit)
		require.NoError(t, err)
		assert.Same(t, want, got)

		r := records()
		require.Len(t, r, 1)
		assert.Equal(t, "INFO", r[0]["level"])
		assert.Equal(t, "doc-001", r[0]["doctor_id"])
		assert.Equal(t, "2025-06-02", r[0]["date"])
		assert.EqualValues(t, 3, r[0]["slots"])
		assert.EqualValues(t, 2, r[0]["free_ranges"])
	})

	t.Run("internal error", func(t *testing.T) {
		buf, records := newLogger(t)
		boom := errors.New("db down")
		l := observability.NewObservedAvailability(fakeAvail{err: boom}, observability.NewLogger(buf, "info", "json"), nil)

		_, err := l.GetAvailability(ctx, "doc-001", date, domain.ReservationTypeFollowUp)
		assert.ErrorIs(t, err, boom)
		assert.Equal(t, "ERROR", records()[0]["level"])
	})
}

func TestObservedReservation_Create(t *testing.T) {
	ctx := context.Background()
	in := usecase.CreateReservationInput{
		DoctorID: "doc-001", PatientID: "pat-001", PatientName: "Secret Name", PatientPhone: "555-9999",
		StartsAt: time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC), Type: domain.ReservationTypeFollowUp,
	}

	t.Run("created, without PII", func(t *testing.T) {
		buf, records := newLogger(t)
		created := &domain.Reservation{ID: "r1", DoctorID: "doc-001", PatientID: "pat-001", StartsAt: in.StartsAt, EndsAt: in.StartsAt.Add(30 * time.Minute)}
		l := observability.NewObservedReservation(fakeRes{res: created}, observability.NewLogger(buf, "info", "json"), nil)

		got, err := l.Create(ctx, in)
		require.NoError(t, err)
		assert.Same(t, created, got)

		r := records()
		require.Len(t, r, 1)
		assert.Equal(t, "INFO", r[0]["level"])
		assert.Equal(t, "r1", r[0]["reservation_id"])
		assert.NotContains(t, buf.String(), "Secret Name")
		assert.NotContains(t, buf.String(), "555-9999")
	})

	for name, tc := range map[string]struct {
		err   error
		level string
	}{
		"conflict":        {domain.ErrSlotNotAvailable, "WARN"},
		"blocked":         {domain.ErrSlotBlocked, "WARN"},
		"unknown patient": {fmt.Errorf("resolve patient: %w", domain.ErrPatientNotFound), "WARN"},
		"internal":        {errors.New("db down"), "ERROR"},
	} {
		t.Run(name, func(t *testing.T) {
			buf, records := newLogger(t)
			l := observability.NewObservedReservation(fakeRes{err: tc.err}, observability.NewLogger(buf, "info", "json"), nil)

			_, err := l.Create(ctx, in)
			assert.ErrorIs(t, err, tc.err)

			r := records()
			require.Len(t, r, 1)
			assert.Equal(t, tc.level, r[0]["level"])
			assert.Equal(t, tc.err.Error(), r[0]["reason"])
		})
	}
}

func TestObservedReservation_Cancel(t *testing.T) {
	buf, records := newLogger(t)
	l := observability.NewObservedReservation(fakeRes{}, observability.NewLogger(buf, "info", "json"), nil)
	require.NoError(t, l.Cancel(context.Background(), "r1"))
	assert.Equal(t, "reservation cancelled", records()[0]["msg"])
}

func TestUnaryLoggingInterceptor(t *testing.T) {
	info := &grpc.UnaryServerInfo{FullMethod: "/agenda.v1.AgendaService/CreateReservation"}

	for name, tc := range map[string]struct {
		err   error
		code  string
		level string
	}{
		"ok":        {nil, "OK", "INFO"},
		"conflict":  {status.Error(codes.AlreadyExists, "x"), "AlreadyExists", "INFO"},
		"internal":  {status.Error(codes.Internal, "x"), "Internal", "ERROR"},
		"plain err": {errors.New("x"), "Unknown", "ERROR"},
	} {
		t.Run(name, func(t *testing.T) {
			buf, records := newLogger(t)
			ic := observability.UnaryLoggingInterceptor(observability.NewLogger(buf, "info", "json"))

			resp, err := ic(context.Background(), nil, info, func(context.Context, any) (any, error) { return "resp", tc.err })
			assert.Equal(t, "resp", resp)
			assert.Equal(t, tc.err, err)

			r := records()
			require.Len(t, r, 1)
			assert.Equal(t, info.FullMethod, r[0]["method"])
			assert.Equal(t, tc.code, r[0]["code"])
			assert.Equal(t, tc.level, r[0]["level"])
			assert.Contains(t, r[0], "duration")
		})
	}
}
