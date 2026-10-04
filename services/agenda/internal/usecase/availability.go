package usecase

import (
	"context"
	"fmt"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"
)

type AvailabilityUsecase struct {
	doctors      domain.DoctorRepository
	reservations domain.ReservationRepository
	blockedSlots domain.BlockedSlotRepository
}

func NewAvailabilityUsecase(
	doctors domain.DoctorRepository,
	reservations domain.ReservationRepository,
	blockedSlots domain.BlockedSlotRepository,
) *AvailabilityUsecase {
	return &AvailabilityUsecase{
		doctors:      doctors,
		reservations: reservations,
		blockedSlots: blockedSlots,
	}
}

type AvailabilityResult struct {
	Slots      []domain.Reservation // discrete available start/end times
	FreeRanges [][2]time.Time       // continuous free ranges
}

func (u *AvailabilityUsecase) GetAvailability(
	ctx context.Context,
	doctorID string,
	date time.Time,
	resType domain.ReservationType,
) (*AvailabilityResult, error) {
	doctor, err := u.doctors.GetDoctor(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("get doctor: %w", err)
	}

	dayStart, dayEnd, ok := workingWindow(doctor, date)
	if !ok {
		return &AvailabilityResult{}, nil
	}

	// Get existing reservations for the day
	existing, err := u.reservations.ListReservations(ctx, doctorID, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	blocked, err := u.blockedSlots.ListBlockedSlots(ctx, doctorID, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("list blocked slots: %w", err)
	}

	busy := reservationsToBusy(existing)
	busy = append(busy, blockedToBusy(blocked, dayStart, dayEnd)...)
	free := subtractBusy(dayStart, dayEnd, busy)
	slots := slicesFromFreeRanges(free, resType.SlotDuration())

	return &AvailabilityResult{
		Slots:      slots,
		FreeRanges: free,
	}, nil
}

func workingWindow(doctor *domain.Doctor, date time.Time) (time.Time, time.Time, bool) {
	wd := domain.Weekday(date.Weekday())
	for _, wh := range doctor.WorkingHours {
		if wh.Weekday == wd {
			from := parseTimeOnDate(date, wh.From)
			to := parseTimeOnDate(date, wh.To)
			return from, to, true
		}
	}
	return time.Time{}, time.Time{}, false
}

func parseTimeOnDate(date time.Time, hhmm string) time.Time {
	var h, m int
	fmt.Sscanf(hhmm, "%d:%d", &h, &m)
	return time.Date(date.Year(), date.Month(), date.Day(), h, m, 0, 0, date.Location())
}

func reservationsToBusy(reservations []*domain.Reservation) [][2]time.Time {
	busy := make([][2]time.Time, 0, len(reservations))
	for _, r := range reservations {
		busy = append(busy, [2]time.Time{r.StartsAt, r.EndsAt})
	}
	return busy
}

// blockedToBusy expands recurring blocked slots and returns every occurrence
// overlapping [from, to] as a busy period.
func blockedToBusy(blocked []*domain.BlockedSlot, from, to time.Time) [][2]time.Time {
	var busy [][2]time.Time
	for _, b := range blocked {
		for _, occ := range b.Occurrences(from, to) {
			busy = append(busy, [2]time.Time{occ.StartsAt, occ.EndsAt})
		}
	}
	return busy
}

func subtractBusy(start, end time.Time, busy [][2]time.Time) [][2]time.Time {
	free := [][2]time.Time{{start, end}}
	for _, b := range busy {
		free = subtractInterval(free, b)
	}
	return free
}

func subtractInterval(free [][2]time.Time, busy [2]time.Time) [][2]time.Time {
	result := make([][2]time.Time, 0, len(free))
	for _, f := range free {
		if busy[1].Before(f[0]) || busy[0].After(f[1]) {
			result = append(result, f)
			continue
		}
		if f[0].Before(busy[0]) {
			result = append(result, [2]time.Time{f[0], busy[0]})
		}
		if busy[1].Before(f[1]) {
			result = append(result, [2]time.Time{busy[1], f[1]})
		}
	}
	return result
}

func slicesFromFreeRanges(free [][2]time.Time, duration time.Duration) []domain.Reservation {
	var slots []domain.Reservation
	for _, f := range free {
		cur := f[0]
		for !cur.Add(duration).After(f[1]) {
			slots = append(slots, domain.Reservation{
				StartsAt: cur,
				EndsAt:   cur.Add(duration),
			})
			cur = cur.Add(duration)
		}
	}
	return slots
}
