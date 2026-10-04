package domain

import "time"

type RecurrenceType int

const (
	RecurrenceNone RecurrenceType = iota
	RecurrenceDaily
	RecurrenceWeekly
	RecurrenceMonthly
)

type BlockedSlot struct {
	ID              string
	DoctorID        string
	StartsAt        time.Time
	EndsAt          time.Time
	Reason          string
	RecurrenceType  RecurrenceType
	RecurrenceUntil *time.Time
}

// Occurrences returns all occurrences of this blocked slot that overlap
// [from, to]. Boundaries are inclusive: an occurrence ending exactly at from, or
// starting exactly at to, is returned.
//
// Recurring slots repeat daily, weekly or monthly from StartsAt. If
// RecurrenceUntil is set, no occurrence starting after it is returned. Monthly
// occurrences keep the base day of month, clamped to the last day of shorter
// months (Jan 31 -> Feb 28 -> Mar 31). Every occurrence is derived from the base
// slot, so clamping never carries over into later months.
func (b *BlockedSlot) Occurrences(from, to time.Time) []BlockedSlot {
	dur := b.EndsAt.Sub(b.StartsAt)

	if b.RecurrenceType != RecurrenceDaily &&
		b.RecurrenceType != RecurrenceWeekly &&
		b.RecurrenceType != RecurrenceMonthly {
		if b.StartsAt.After(to) || b.EndsAt.Before(from) {
			return nil
		}
		return []BlockedSlot{*b}
	}

	var out []BlockedSlot
	for k := b.firstCandidate(from, dur); ; k++ {
		start := b.nthStart(k)
		if start.After(to) {
			break
		}
		if b.RecurrenceUntil != nil && start.After(*b.RecurrenceUntil) {
			break
		}
		end := start.Add(dur)
		if end.Before(from) {
			continue
		}
		occ := *b
		occ.StartsAt, occ.EndsAt = start, end
		out = append(out, occ)
	}
	return out
}

// nthStart returns the start of occurrence k (k=0 is the base slot).
func (b *BlockedSlot) nthStart(k int) time.Time {
	switch b.RecurrenceType {
	case RecurrenceDaily:
		return b.StartsAt.AddDate(0, 0, k)
	case RecurrenceWeekly:
		return b.StartsAt.AddDate(0, 0, 7*k)
	default: // RecurrenceMonthly
		y, m, d := b.StartsAt.Date()
		first := time.Date(y, m+time.Month(k), 1, 0, 0, 0, 0, b.StartsAt.Location())
		last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, b.StartsAt.Location()).Day()
		if d > last {
			d = last
		}
		return time.Date(first.Year(), first.Month(), d,
			b.StartsAt.Hour(), b.StartsAt.Minute(), b.StartsAt.Second(), b.StartsAt.Nanosecond(),
			b.StartsAt.Location())
	}
}

// firstCandidate returns a k that is never later than the first occurrence
// ending at or after from, so the loop can skip past far-away history.
func (b *BlockedSlot) firstCandidate(from time.Time, dur time.Duration) int {
	cutoff := from.Add(-dur)
	if !cutoff.After(b.StartsAt) {
		return 0
	}
	var k int
	switch b.RecurrenceType {
	case RecurrenceDaily:
		k = int(cutoff.Sub(b.StartsAt).Hours()/24) - 1
	case RecurrenceWeekly:
		k = int(cutoff.Sub(b.StartsAt).Hours()/(24*7)) - 1
	default:
		sy, sm, _ := b.StartsAt.Date()
		cy, cm, _ := cutoff.In(b.StartsAt.Location()).Date()
		k = (cy-sy)*12 + int(cm-sm) - 1
	}
	if k < 0 {
		return 0
	}
	return k
}
