package http

import (
	"net/http"

	"go-challenge-agenda/services/api/internal/domain"
	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	uc *usecase.UserUsecase
}

func NewUserHandler(uc *usecase.UserUsecase) *UserHandler {
	return &UserHandler{uc: uc}
}

// List godoc
// @Summary     List all users
// @Tags        users
// @Produce     json
// @Success     200  {array}   domain.UserResponse
// @Failure     500  {object}  map[string]string
// @Router      /users [get]
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.uc.List(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, users)
}

// Get godoc
// @Summary     Get a user by ID
// @Tags        users
// @Produce     json
// @Param       id   path      string  true  "User ID"
// @Success     200  {object}  domain.UserResponse
// @Failure     404  {object}  map[string]string
// @Router      /users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	user, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// Create godoc
// @Summary     Create a new user
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       body  body      domain.CreateUserRequest  true  "User data"
// @Success     201   {object}  domain.UserResponse
// @Failure     400   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.uc.Create(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

// Update godoc
// @Summary     Update a user
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       id    path      string                  true  "User ID"
// @Param       body  body      domain.UpdateUserRequest true  "Fields to update"
// @Success     200   {object}  domain.UserResponse
// @Failure     400   {object}  map[string]string
// @Failure     404   {object}  map[string]string
// @Router      /users/{id} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.uc.Update(c.Request.Context(), c.Param("id"), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// ListReservations godoc
// @Summary     List reservations for a user
// @Tags        users
// @Produce     json
// @Param       id   path      string  true  "User ID"
// @Success     200  {array}   domain.ReservationResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /users/{id}/reservations [get]
func (h *UserHandler) ListReservations(c *gin.Context) {
	reservations, err := h.uc.ListReservations(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, reservations)
}

// Delete godoc
// @Summary     Delete a user
// @Tags        users
// @Param       id   path  string  true  "User ID"
// @Success     204
// @Failure     500  {object}  map[string]string
// @Router      /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.uc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
