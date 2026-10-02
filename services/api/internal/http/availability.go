package http

import (
	"net/http"
	"time"

	agendav1 "go-challenge-agenda/gen/agenda/v1"
	"go-challenge-agenda/services/api/internal/domain"
	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AvailabilityHandler struct {
	uc           *usecase.AvailabilityUsecase
	agendaClient agendav1.AgendaServiceClient
}

func NewAvailabilityHandler(uc *usecase.AvailabilityUsecase, client agendav1.AgendaServiceClient) *AvailabilityHandler {
	return &AvailabilityHandler{uc: uc, agendaClient: client}
}

// Get godoc
// @Summary     Get available slots for a doctor on a given date
// @Tags        availability
// @Produce     json
// @Param       id    path      string  true   "Doctor ID"
// @Param       date  query     string  true   "Date (YYYY-MM-DD)"
// @Param       type  query     string  false  "Reservation type: first_visit or follow_up"
// @Success     200   {object}  domain.AvailabilityResponse
// @Failure     400   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /doctors/{id}/availability [get]
func (h *AvailabilityHandler) Get(c *gin.Context) {

	date := c.Query("date")
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required and must be in YYYY-MM-DD format"})
		return
	}

	var reservationType agendav1.ReservationType
	switch c.Query("type") {
	case "", "first_visit": //defaulting missing value to first_visit
		reservationType = agendav1.ReservationType_RESERVATION_TYPE_FIRST_VISIT
	case "follow_up":
		reservationType = agendav1.ReservationType_RESERVATION_TYPE_FOLLOW_UP
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be first_visit or follow_up"})
		return
	}

	resp, err := h.agendaClient.GetAvailability(c.Request.Context(), &agendav1.GetAvailabilityRequest{
		DoctorId:        c.Param("id"),
		Date:            date,
		ReservationType: reservationType,
	})

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, protoAvailabilityToDTO(resp))
}

func protoAvailabilityToDTO(a *agendav1.GetAvailabilityResponse) *domain.AvailabilityResponse {
	if a == nil {
		return nil
	}

	slots := make([]domain.AvailableSlot, 0, len(a.Slots))
	for _, s := range a.Slots {
		slots = append(slots, domain.AvailableSlot{
			StartsAt: s.GetStartsAt(),
			EndsAt:   s.GetEndsAt(),
		})
	}

	freeRanges := make([]domain.TimeRange, 0, len(a.FreeRanges))
	for _, r := range a.FreeRanges {
		freeRanges = append(freeRanges, domain.TimeRange{
			From: r.GetFrom(),
			To:   r.GetTo(),
		})
	}

	return &domain.AvailabilityResponse{
		Slots:      slots,
		FreeRanges: freeRanges,
	}
}
