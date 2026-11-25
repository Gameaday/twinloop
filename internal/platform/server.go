package platform

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Serve runs srv until SIGINT/SIGTERM, then drains it and calls finalize
// (the final snapshot checkpoint) before returning. Exactly one finalize
// per shutdown: double flushing is how snapshots get clobbered.
func Serve(srv *http.Server, finalize func(context.Context) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(drain); err != nil {
		slog.Error("drain failed", "error", err)
	}
	if finalize != nil {
		if err := finalize(drain); err != nil {
			return err
		}
	}
	return nil
}

// Healthz returns a handler reporting service liveness for Traefik / a
// future compose healthcheck. Scratch images have no shell, so health
// probes must be HTTP (or traefik's service-level checks).
func Healthz(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"service": name, "ok": true})
	}
}

// WriteState writes a JSON state/response body with 200.
func WriteState(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusOK, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response", "error", err)
	}
}

// RunTicks calls fn on a fixed interval until ctx is done. Ticks fire on
// the wall clock, so a downed node catches up on the next tick by design:
// fn receives the real elapsed time since the last tick and is expected
// to no-op quickly when the world is quiet (design invariant 3).
func RunTicks(ctx context.Context, interval time.Duration, fn func(ctx context.Context, elapsed time.Duration) error) {
	if interval <= 0 {
		return
	}
	last := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			elapsed := now.Sub(last)
			last = now
			if err := fn(ctx, elapsed); err != nil {
				slog.Error("tick failed", "error", err, "elapsed", elapsed.String())
			}
		}
	}
}
