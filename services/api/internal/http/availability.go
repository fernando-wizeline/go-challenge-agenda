package http

import (
	"net/http"
	"time"

	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AvailabilityHandler struct {
	uc *usecase.AvailabilityUsecase
}

func NewAvailabilityHandler(uc *usecase.AvailabilityUsecase) *AvailabilityHandler {
	return &AvailabilityHandler{uc: uc}
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

	reservationType := c.Query("type")
	switch reservationType {
	case "":
		reservationType = "first_visit" // default for a missing value
	case "first_visit", "follow_up":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be first_visit or follow_up"})
		return
	}

	resp, err := h.uc.GetAvailability(c.Request.Context(), c.Param("id"), date, reservationType)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
