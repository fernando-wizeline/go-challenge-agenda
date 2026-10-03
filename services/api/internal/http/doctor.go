package http

import (
	"net/http"

	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type DoctorHandler struct {
	uc *usecase.DoctorUsecase
}

func NewDoctorHandler(uc *usecase.DoctorUsecase) *DoctorHandler {
	return &DoctorHandler{uc: uc}
}

// List godoc
// @Summary     List all doctors
// @Tags        doctors
// @Produce     json
// @Success     200  {array}   domain.DoctorResponse
// @Failure     500  {object}  map[string]string
// @Router      /doctors [get]
func (h *DoctorHandler) List(c *gin.Context) {
	doctors, err := h.uc.List(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, doctors)
}

// Get godoc
// @Summary     Get a doctor by ID
// @Tags        doctors
// @Produce     json
// @Param       id   path      string  true  "Doctor ID"
// @Success     200  {object}  domain.DoctorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /doctors/{id} [get]
func (h *DoctorHandler) Get(c *gin.Context) {
	doctor, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, doctor)
}
