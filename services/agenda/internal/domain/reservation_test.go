package domain_test

import (
	"testing"
	"time"

	"go-challenge-agenda/services/agenda/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestSlotDuration(t *testing.T) {
	tests := []struct {
		name string
		typ  domain.ReservationType
		want time.Duration
	}{
		{"first visit", domain.ReservationTypeFirstVisit, 60 * time.Minute},
		{"follow up", domain.ReservationTypeFollowUp, 30 * time.Minute},
		{"unspecified", domain.ReservationTypeUnspecified, 30 * time.Minute},
		{"unknown", domain.ReservationType(99), 30 * time.Minute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.typ.SlotDuration())
		})
	}
}
