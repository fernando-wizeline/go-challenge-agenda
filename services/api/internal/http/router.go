package http

import (
	agendav1 "go-challenge-agenda/gen/agenda/v1"
	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(agendaClient agendav1.AgendaServiceClient) *gin.Engine {
	r := gin.Default()
	r.Use(ErrorMiddleware())

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	availUC := usecase.NewAvailabilityUsecase(agendaClient)
	resUC := usecase.NewReservationUsecase(agendaClient)

	doctorH := NewDoctorHandler(agendaClient)
	availH := NewAvailabilityHandler(availUC, agendaClient)
	resH := NewReservationHandler(resUC, agendaClient)
	userH := NewUserHandler(usecase.NewUserUsecase(agendaClient))

	v1 := r.Group("/v1")
	{
		v1.GET("/doctors", doctorH.List)
		v1.GET("/doctors/:id", doctorH.Get)
		v1.GET("/doctors/:id/availability", availH.Get)

		v1.POST("/reservations", resH.Create)
		v1.GET("/reservations/:id", resH.Get)
		v1.GET("/reservations", resH.List)
		v1.DELETE("/reservations/:id", resH.Cancel)

		v1.GET("/users", userH.List)
		v1.POST("/users", userH.Create)
		v1.GET("/users/:id", userH.Get)
		v1.PATCH("/users/:id", userH.Update)
		v1.DELETE("/users/:id", userH.Delete)
		v1.GET("/users/:id/reservations", userH.ListReservations)
	}

	return r
}
