package domain

import "time"

type ReservationType int

const (
	ReservationTypeUnspecified ReservationType = iota
	ReservationTypeFirstVisit
	ReservationTypeFollowUp
)

const (
	firstVisitDuration = 60 * time.Minute
	followUpDuration   = 30 * time.Minute
)

// SlotDuration returns the duration for a reservation type. Unspecified or
// unknown types get the follow-up duration.
func (t ReservationType) SlotDuration() time.Duration {
	switch t {
	case ReservationTypeFirstVisit:
		return firstVisitDuration
	default:
		return followUpDuration
	}
}

type ReservationStatus int

const (
	ReservationStatusUnspecified ReservationStatus = iota
	ReservationStatusConfirmed
	ReservationStatusCancelled
)

type Reservation struct {
	ID        string
	DoctorID  string
	PatientID string
	StartsAt  time.Time
	EndsAt    time.Time
	Type      ReservationType
	Status    ReservationStatus
}
