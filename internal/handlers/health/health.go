package health

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// LivenessHandler implements Kubernetes liveness probe
// Returns 200 if the application is running and hasn't deadlocked
func LivenessHandler(healthy *int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(healthy) == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
	}
}

// ReadinessHandler implements Kubernetes readiness probe
// Returns 200 if the application is ready to serve traffic
// For a proxy service like this one, this includes checking connectivity to Azure DevOps API
func ReadinessHandler(ready *int32, client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First check if the service is marked as ready
		if atomic.LoadInt32(ready) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service Not Ready"))
			return
		}

		// For a AzureDevOps API proxy, we verify we can reach AzureDevOps
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", "https://status.dev.azure.com/_apis/status/health?api-version=7.1-preview.1", nil)
		if err != nil {
			log.Debug().Err(err).Msg("failed to create AzureDevOps API request for readiness check")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("AzureDevOps API Unreachable"))
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Debug().Err(err).Msg("failed to reach AzureDevOps API for readiness check")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("AzureDevOps API Unreachable"))
			return
		}
		defer resp.Body.Close()

		// AzureDevOps API should respond with some 2xx or 4xx status (4xx is still fine, means AzureDevOps API is up)
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ready"))
			return
		}

		log.Debug().Int("status", resp.StatusCode).Msg("AzureDevOps API returned unexpected status for readiness check")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("AzureDevOps API Error"))
	}
}
