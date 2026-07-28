// Package cmd — `metrics` subcommand serves L4 healing/trust counters for Prometheus scraping.
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

const metricsContentType = "text/plain; version=0.0.4; charset=utf-8"

// metricsListenAndServe is overridable in tests to avoid binding a real port.
var metricsListenAndServe = func(srv *http.Server) error {
	return srv.ListenAndServe()
}

func runMetrics(args []string) error {
	fs := newFlagSet("metrics")
	// Default to loopback only (code-review HIGH: :9090 binds all interfaces).
	addr := fs.String("addr", "127.0.0.1:9090", "HTTP listen address (default loopback-only)")
	root := fs.String("root", ".", "workspace root — reads .l4-memory/metrics.json written by l4 handle")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return serveMetrics(*addr, *root)
}

func serveMetrics(addr, root string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHTTPHandler(root))
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- metricsListenAndServe(srv)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "metrics: received %v, shutting down\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func metricsHTTPHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", metricsContentType)
		// Prefer persisted snapshot so a scrape process sees l4 handle counters.
		if err := l4.WritePrometheusFromRoot(w, root); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
