// sunderd is the dungeon roguelike daemon (Sunder). Runs are turn-based
// and persisted in SQLite, so the process idles at zero CPU between
// actions. The full turn state machine lands after this skeleton; this
// draft already proves daily seeding, deterministic generation, and the
// hydrate/checkpoint lifecycle end to end.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Gameaday/twinloop/internal/delve"
	"github.com/Gameaday/twinloop/internal/platform"
)

// defaultEpoch matches the fleet convention (daily.historylabs.dev epoch
// 2026-11-02); configurable so tests and replays can pin any day.
const defaultEpoch = "2026-11-02"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(nil, nil)))
	cfg := platform.Load()
	snap := &platform.SnapshotClient{
		Endpoint: cfg.StateEndpoint,
		Token:    cfg.StateToken,
		Prefix:   cfg.StatePrefix,
	}

	seedSecret := []byte(cfg.DailySeedSecret)
	if len(seedSecret) == 0 {
		slog.Warn("DAILY_SEED_SECRET not set; using an insecure dev secret, set it in .env before public testing")
		seedSecret = []byte("dev-only-insecure-secret")
	}
	epoch, err := time.Parse("2006-01-02", platform.EnvOr("DAILY_EPOCH", defaultEpoch))
	if err != nil {
		slog.Error("DAILY_EPOCH must be YYYY-MM-DD", "error", err)
		return
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	if err := snap.Hydrate(ctx, cfg.DataDir); err != nil {
		slog.Error("hydrate failed; refusing to boot a divergent dungeon over a good remote one (see docs/STATE.md)", "error", err)
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", platform.Healthz("sunderd"))
	mux.HandleFunc("GET /api/delve/daily", func(w http.ResponseWriter, r *http.Request) {
		dayIndex := delve.DayIndex(time.Now(), epoch)
		seed := delve.DailySeed(seedSecret, dayIndex)
		// Layout stays server-side; the client learns screens only as
		// its run visits them. This endpoint is a draft read-only peek
		// for integration testing and will become run/screen endpoints.
		floor := delve.GenerateFloor(delve.RandFromSeed(seed))
		start := [2]int{floor.Width / 2, floor.Height / 2}
		platform.WriteState(w, map[string]any{
			"day_index": dayIndex,
			"theme":     floor.Theme,
			"start":     map[string]int{"x": start[0], "y": start[1]},
			"note":      "the layout is never sent up front; screens are revealed by visited responses only",
		})
	})

	go platform.RunTicks(ctx, cfg.BackupInterval, func(bctx context.Context, _ time.Duration) error {
		return platform.CheckpointSnapshot(bctx, db, snap, cfg.DataDir)
	})

	srv := &http.Server{Addr: cfg.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	err = platform.Serve(srv, func(fctx context.Context) error {
		stop()
		return platform.CheckpointSnapshot(fctx, db, snap, cfg.DataDir)
	})
	if err != nil {
		slog.Error("serve", "error", err)
	}
}
