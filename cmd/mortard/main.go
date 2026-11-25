// mortard is the settlement world sim tick daemon (Mortar). It holds the
// world truth: the action queue, the chronicle, and the dungeon threat
// meter. The cross-game bridge to sunderd is intentionally not wired yet
// (red-team R1): both games must be fun alone first.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Gameaday/twinloop/internal/platform"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(nil, nil)))
	cfg := platform.Load()
	snap := &platform.SnapshotClient{
		Endpoint: cfg.StateEndpoint,
		Token:    cfg.StateToken,
		Prefix:   cfg.StatePrefix,
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	if err := snap.Hydrate(ctx, cfg.DataDir); err != nil {
		slog.Error("hydrate failed; booting a divergent world over a good remote one is worse than staying down (see docs/STATE.md)", "error", err)
		if cfg.StateEndpoint != "" && cfg.StatePrefix != "" {
			return
		}
	}
	db, err := platform.Open(cfg.DataDir, cfg.MigrationsDir)
	if err != nil {
		slog.Error("open store", "error", err)
		return
	}
	defer db.Close()

	var ticks int64
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", platform.Healthz("mortard"))
	mux.HandleFunc("GET /api/world/state", func(w http.ResponseWriter, r *http.Request) {
		platform.WriteState(w, map[string]any{
			"ticks":             ticks,
			"tick_interval_sec": int(cfg.TickInterval.Seconds()),
			"note":              "state model lands with deliverable 5 (daily tick + ledger)",
		})
	})

	// Periodic checkpoint: hydrate-on-boot pairs with push-on-interval.
	go platform.RunTicks(ctx, cfg.BackupInterval, func(bctx context.Context, _ time.Duration) error {
		return platform.CheckpointSnapshot(bctx, db, snap, cfg.DataDir)
	})

	// World tick loop. The real resolution engine (queue advance,
	// chamber digest, chronicle) lands with the worldd deliverable; a
	// quiet world no-ops here by design.
	go platform.RunTicks(ctx, cfg.TickInterval, func(_ context.Context, elapsed time.Duration) error {
		ticks++
		slog.Info("tick", "elapsed", elapsed.String())
		return nil
	})

	srv := &http.Server{Addr: cfg.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	err = platform.Serve(srv, func(fctx context.Context) error {
		stop() // stop tickers before the final checkpoint
		return platform.CheckpointSnapshot(fctx, db, snap, cfg.DataDir)
	})
	if err != nil {
		slog.Error("serve", "error", err)
	}
}
