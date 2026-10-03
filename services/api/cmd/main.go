// @title           Clinic Scheduling API
// @version         1.0
// @description     Public-facing HTTP API for medical appointment scheduling.
// @BasePath        /v1
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-challenge-agenda/services/api/config"
	_ "go-challenge-agenda/services/api/docs" // swagger generated docs
	apiclient "go-challenge-agenda/services/api/internal/grpc"
	apihttp "go-challenge-agenda/services/api/internal/http"
)

func main() {
	cfg := config.Load()

	agendaClient, conn, err := apiclient.NewAgendaClient(cfg.AgendaGRPCAddr)
	if err != nil {
		log.Fatalf("connect to agenda: %v", err)
	}
	defer conn.Close()

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: apihttp.NewRouter(apiclient.NewAgendaAdapter(agendaClient)),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("api HTTP server listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down api service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}

	log.Println("api service stopped")
}
