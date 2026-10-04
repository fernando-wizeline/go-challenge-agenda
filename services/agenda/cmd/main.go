package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	agendav1 "go-challenge-agenda/gen/agenda/v1"
	"go-challenge-agenda/services/agenda/config"
	agendagrpc "go-challenge-agenda/services/agenda/internal/grpc"
	"go-challenge-agenda/services/agenda/internal/repository/sqlite"
	"go-challenge-agenda/services/agenda/internal/usecase"

	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := openDB(cfg)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	if err := sqlite.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := sqlite.Seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	doctorRepo := sqlite.NewDoctorRepository(db)
	patientRepo := sqlite.NewPatientRepository(db)
	reservationRepo := sqlite.NewReservationRepository(db)
	blockedSlotRepo := sqlite.NewBlockedSlotRepository(db)

	availUC := usecase.NewAvailabilityUsecase(doctorRepo, reservationRepo, blockedSlotRepo)
	reservationUC := usecase.NewReservationUsecase(reservationRepo, patientRepo, blockedSlotRepo)
	blockedSlotUC := usecase.NewBlockedSlotUsecase(blockedSlotRepo)

	srv := agendagrpc.NewServer(doctorRepo, availUC, reservationUC, blockedSlotUC, patientRepo)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	agendav1.RegisterAgendaServiceServer(grpcServer, srv)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("agenda gRPC server listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC serve stopped: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down agenda service...")

	grpcServer.GracefulStop()

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	log.Println("agenda service stopped")
}

func openDB(cfg config.Config) (*gorm.DB, error) {
	switch cfg.DBDriver {
	case "sqlite3":
		return sqlite.Open(cfg.DBSource)
	case "postgres":
		return nil, fmt.Errorf("postgres driver not yet implemented — see services/agenda/internal/repository/postgres/")
	default:
		return nil, fmt.Errorf("unknown DB driver: %s", cfg.DBDriver)
	}
}
