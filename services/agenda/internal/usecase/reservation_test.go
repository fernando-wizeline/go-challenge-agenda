package usecase_test

import (
	"context"
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
	"go-challenge-agenda/services/agenda/internal/domain/mocks"
	"go-challenge-agenda/services/agenda/internal/repository/sqlite"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateReservation_ConflictDetected(t *testing.T) {
	base := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)

	existing := &domain.Reservation{
		ID:       "existing",
		DoctorID: "doc-001",
		StartsAt: base,
		EndsAt:   base.Add(30 * time.Minute),
		Status:   domain.ReservationStatus(domain.ReservationStatusConfirmed),
	}

	reservationRepo := mocks.NewReservationRepository(t)
	patientRepo := mocks.NewPatientRepository(t)

	patientRepo.EXPECT().
		GetPatientByPhone(context.Background(), "555-0001").
		Return(nil, nil)
	patientRepo.EXPECT().
		CreatePatient(context.Background(), mock.MatchedBy(func(_ *domain.Patient) bool { return true })).
		Return(nil).Maybe()

	// hasConflict uses: ListReservations(ctx, doctorID, startsAt-24h, endsAt+24h)
	// startsAt = base+15m = 10:15, endsAt = 10:15+30m = 10:45 (SlotDuration always 30m due to bug)
	newStart := base.Add(15 * time.Minute)
	newEnd := newStart.Add(30 * time.Minute) // SlotDuration bug: always 30m
	reservationRepo.EXPECT().
		ListReservations(context.Background(), "doc-001", newStart.Add(-24*time.Hour), newEnd.Add(24*time.Hour)).
		Return([]*domain.Reservation{existing}, nil)

	uc := usecase.NewReservationUsecase(reservationRepo, patientRepo)

	_, err := uc.Create(context.Background(), usecase.CreateReservationInput{
		DoctorID:     "doc-001",
		StartsAt:     newStart,
		Type:         domain.ReservationTypeFollowUp,
		PatientPhone: "555-0001",
		PatientName:  "New Patient",
		PatientEmail: "new@example.com",
	})

	assert.ErrorIs(t, err, domain.ErrSlotNotAvailable, "expected conflict error")
}

// TestCreateReservation_BoundaryConflict verifies adjacent booking (starts exactly when prior ends) is ALLOWED.
// This test FAILS due to the known boundary bug in hasConflict.
func TestCreateReservation_BoundaryConflict(t *testing.T) {
	base := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	adjacentStart := base.Add(30 * time.Minute)

	existing := &domain.Reservation{
		ID: "existing", DoctorID: "doc-001",
		StartsAt: base, EndsAt: base.Add(30 * time.Minute),
		Status: domain.ReservationStatus(domain.ReservationStatusConfirmed),
	}

	reservationRepo := mocks.NewReservationRepository(t)
	patientRepo := mocks.NewPatientRepository(t)

	patientRepo.EXPECT().GetPatientByPhone(context.Background(), "555-0002").Return(nil, nil)
	patientRepo.EXPECT().CreatePatient(context.Background(), mock.MatchedBy(func(_ *domain.Patient) bool { return true })).Return(nil).Maybe()

	// hasConflict: startsAt=10:30, endsAt=11:00 (30m bug). Window: [10:30-24h, 11:00+24h]
	newEnd := adjacentStart.Add(30 * time.Minute) // SlotDuration bug: always 30m
	reservationRepo.EXPECT().
		ListReservations(context.Background(), "doc-001", adjacentStart.Add(-24*time.Hour), newEnd.Add(24*time.Hour)).
		Return([]*domain.Reservation{existing}, nil)
	reservationRepo.EXPECT().
		CreateReservation(context.Background(), mock.MatchedBy(func(_ *domain.Reservation) bool { return true })).
		Return(nil).Maybe()

	uc := usecase.NewReservationUsecase(reservationRepo, patientRepo)

	res, err := uc.Create(context.Background(), usecase.CreateReservationInput{
		DoctorID: "doc-001", StartsAt: adjacentStart,
		Type:         domain.ReservationTypeFollowUp,
		PatientPhone: "555-0002", PatientName: "Another Patient", PatientEmail: "another@example.com",
	})

	require.NoError(t, err, "adjacent booking should be allowed")
	assert.NotNil(t, res)
}

// TestCreateReservation_FirstVisitDuration checks that a first visit allocates 60 minutes.
// This test FAILS because SlotDuration always returns 30 minutes.
func TestCreateReservation_FirstVisitDuration(t *testing.T) {
	base := time.Date(2025, 6, 2, 9, 0, 0, 0, time.UTC)

	reservationRepo := mocks.NewReservationRepository(t)
	patientRepo := mocks.NewPatientRepository(t)

	patientRepo.EXPECT().GetPatientByPhone(context.Background(), "555-0003").Return(nil, nil)
	patientRepo.EXPECT().CreatePatient(context.Background(), mock.MatchedBy(func(_ *domain.Patient) bool { return true })).Return(nil)

	// hasConflict: startsAt=9:00, endsAt=9:30 (bug: SlotDuration always 30m for FirstVisit).
	// Window: [9:00-24h, 9:30+24h]
	buggyEnd := base.Add(30 * time.Minute) // SlotDuration bug: should be 60m but is 30m
	reservationRepo.EXPECT().
		ListReservations(context.Background(), "doc-001", base.Add(-24*time.Hour), buggyEnd.Add(24*time.Hour)).
		Return(nil, nil)
	reservationRepo.EXPECT().
		CreateReservation(context.Background(), mock.MatchedBy(func(_ *domain.Reservation) bool { return true })).
		Return(nil)

	uc := usecase.NewReservationUsecase(reservationRepo, patientRepo)

	res, err := uc.Create(context.Background(), usecase.CreateReservationInput{
		DoctorID: "doc-001", StartsAt: base,
		Type:         domain.ReservationTypeFirstVisit,
		PatientPhone: "555-0003", PatientName: "First Timer", PatientEmail: "first@example.com",
	})
	require.NoError(t, err)

	expected := 60 * time.Minute
	actual := res.EndsAt.Sub(res.StartsAt)
	assert.Equal(t, expected, actual, "first visit should be 60 minutes")
}

// TestListReservations verifies the SQLite repository returns only the doctor's
// reservations overlapping the range, ordered by start time.
func TestListReservations(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	repo := sqlite.NewReservationRepository(db)
	ctx := context.Background()
	base := time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)

	create := func(id, doctorID string, offsetMin, durMin int) {
		require.NoError(t, repo.CreateReservation(ctx, &domain.Reservation{
			ID:        id,
			DoctorID:  doctorID,
			PatientID: "patient-001",
			StartsAt:  base.Add(time.Duration(offsetMin) * time.Minute),
			EndsAt:    base.Add(time.Duration(offsetMin+durMin) * time.Minute),
			Status:    domain.ReservationStatus(domain.ReservationStatusConfirmed),
		}))
	}
	create("before", "doc-001", -60, 30)      // 08:00-08:30, entirely before range
	create("touch-start", "doc-001", -30, 30) // 08:30-09:00, ends exactly at range start
	create("straddle", "doc-001", 100, 60)    // 10:40-11:40, crosses range end
	create("inside", "doc-001", 30, 30)       // 09:30-10:00, inside range
	create("other-doctor", "doc-002", 30, 30)

	got, err := repo.ListReservations(ctx, "doc-001", base, base.Add(2*time.Hour))
	require.NoError(t, err)

	ids := make([]string, len(got))
	for i, r := range got {
		ids[i] = r.ID
	}
	assert.Equal(t, []string{"inside", "straddle"}, ids)
}

func TestListReservationsByUser(t *testing.T) {
	ctx := context.Background()
	want := []*domain.Reservation{{ID: "r1", PatientID: "pat-001"}}

	reservationRepo := mocks.NewReservationRepository(t)
	patientRepo := mocks.NewPatientRepository(t)
	patientRepo.EXPECT().GetPatient(ctx, "pat-001").Return(&domain.Patient{ID: "pat-001"}, nil)
	reservationRepo.EXPECT().ListReservationsByUser(ctx, "pat-001").Return(want, nil)

	uc := usecase.NewReservationUsecase(reservationRepo, patientRepo)
	got, err := uc.ListReservationsByUser(ctx, "pat-001")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListReservationsByUser_UnknownPatient(t *testing.T) {
	ctx := context.Background()

	// reservationRepo has no expectations: it must not be queried for an unknown patient.
	reservationRepo := mocks.NewReservationRepository(t)
	patientRepo := mocks.NewPatientRepository(t)
	patientRepo.EXPECT().GetPatient(ctx, "missing").Return(nil, domain.ErrPatientNotFound)

	uc := usecase.NewReservationUsecase(reservationRepo, patientRepo)
	_, err := uc.ListReservationsByUser(ctx, "missing")

	assert.ErrorIs(t, err, domain.ErrPatientNotFound)
}

func TestListReservationsByUser_Repository(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	repo := sqlite.NewReservationRepository(db)
	ctx := context.Background()
	base := time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)

	create := func(id, patientID string, offsetMin int) {
		require.NoError(t, repo.CreateReservation(ctx, &domain.Reservation{
			ID:        id,
			DoctorID:  "doc-001",
			PatientID: patientID,
			StartsAt:  base.Add(time.Duration(offsetMin) * time.Minute),
			EndsAt:    base.Add(time.Duration(offsetMin+30) * time.Minute),
			Status:    domain.ReservationStatus(domain.ReservationStatusConfirmed),
		}))
	}
	create("later", "pat-001", 120)
	create("other-patient", "pat-002", 30)
	create("earlier", "pat-001", 0)

	got, err := repo.ListReservationsByUser(ctx, "pat-001")
	require.NoError(t, err)

	ids := make([]string, len(got))
	for i, r := range got {
		ids[i] = r.ID
	}
	assert.Equal(t, []string{"earlier", "later"}, ids)
}

func TestCreateReservation_UnknownPatient(t *testing.T) {
	ctx := context.Background()

	// reservationRepo has no expectations: nothing may be queried or created.
	reservationRepo := mocks.NewReservationRepository(t)
	patientRepo := mocks.NewPatientRepository(t)
	patientRepo.EXPECT().GetPatient(ctx, "missing").Return(nil, domain.ErrPatientNotFound)

	uc := usecase.NewReservationUsecase(reservationRepo, patientRepo)
	_, err := uc.Create(ctx, usecase.CreateReservationInput{
		DoctorID:  "doc-001",
		StartsAt:  time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
		Type:      domain.ReservationTypeFollowUp,
		PatientID: "missing",
	})

	assert.ErrorIs(t, err, domain.ErrPatientNotFound)
}
