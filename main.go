package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/docs"
	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers"
	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers/gitrepository"
	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers/health"
	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers/pipelinepermission"
	"github.com/krateoplatformops/plumbing/env"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	serviceName = "azuredevops-plugin"
)

// @title           Azure DevOps Plugin API for Krateo Operator Generator (KOG)
// @version         1.0
// @description     Simple wrapper around Azure DevOps API to provide consisentency of API response for Krateo Operator Generator (KOG)
// @termsOfService  http://swagger.io/terms/

// @contact.name   Krateo Support
// @contact.url    https://krateo.io
// @contact.email  contact@krateoplatformops.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host			localhost:8080
// @BasePath		/
// @schemes 	 	http

// @securityDefinitions.basic  Bearer

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	debugOn := flag.Bool("debug", env.Bool("DEBUG", true), "dump verbose output")
	port := flag.Int("port", env.Int("PORT", 8080), "port to listen on")
	noColor := flag.Bool("no-color", env.Bool("NO_COLOR", false), "disable color output")

	flag.Parse()

	mux := http.NewServeMux()

	// Initialize the logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Default level for this log is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debugOn {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: *noColor,
	}).With().Timestamp().Logger()

	opts := handlers.HandlerOptions{
		Log:    &log.Logger,
		Client: http.DefaultClient,
	}

	// Health status flags
	healthy := int32(0)
	ready := int32(0)

	// Business logic routes to handle some Azure DevOps API's endpoints

	// PipelinePermission
	mux.Handle("GET /api/{organization}/{project}/pipelines/pipelinepermissions/{resourceType}/{resourceId}", pipelinepermission.GetPipelinePermission(opts))

	// GitRepository
	mux.Handle("POST /api/{organization}/{projectId}/git/repositories", gitrepository.PostGitRepository(opts))

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Kubernetes health check endpoints
	mux.HandleFunc("GET /healthz", health.LivenessHandler(&healthy))
	mux.HandleFunc("GET /readyz", health.ReadinessHandler(&ready, opts.Client.(*http.Client)))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 50 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), []os.Signal{
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGHUP,
		syscall.SIGQUIT,
	}...)
	defer stop()

	go func() {
		// Mark as healthy and ready when server starts
		atomic.StoreInt32(&healthy, 1)
		atomic.StoreInt32(&ready, 1)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("could not listen on %s", server.Addr)
		}
	}()

	// Listen for the interrupt signal.
	log.Info().Msgf("server is ready to handle requests at %s", server.Addr)
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info().Msg("server is shutting down gracefully, press Ctrl+C again to force")

	// Mark as not ready during shutdown, but keep liveness active for graceful shutdown
	atomic.StoreInt32(&ready, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	// Mark as unhealthy after shutdown
	atomic.StoreInt32(&healthy, 0)
	log.Info().Msg("server gracefully stopped")
}
