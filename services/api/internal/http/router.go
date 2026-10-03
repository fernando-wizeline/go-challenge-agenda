package http

import (
	"go-challenge-agenda/services/api/internal/usecase"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter wires the HTTP handlers. Handlers depend on usecases only, and the
// usecases on the AgendaPort, so no gRPC or protobuf types appear in this package.
func NewRouter(agenda usecase.AgendaPort) *gin.Engine {
	r := gin.Default()
	r.Use(ErrorMiddleware())

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	availUC := usecase.NewAvailabilityUsecase(agenda)
	resUC := usecase.NewReservationUsecase(agenda)

	doctorH := NewDoctorHandler(usecase.NewDoctorUsecase(agenda))
	availH := NewAvailabilityHandler(availUC)
	resH := NewReservationHandler(resUC)
	userH := NewUserHandler(usecase.NewUserUsecase(agenda))

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
