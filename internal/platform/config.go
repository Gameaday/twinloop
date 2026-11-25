package platform

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config carries every runtime setting. All values arrive from the
// environment so a Compose stack configures a node with no host-level
// mounts and no manual prep (design invariant 2). Nothing else is read.
type Config struct {
	Addr            string        // HTTP listen address
	DataDir         string        // local dir holding app.db (never network-backed)
	MigrationsDir   string        // baked into the image at /migrations
	StateEndpoint   string        // REST snapshot endpoint on Methuselah; "" disables
	StatePrefix     string        // object prefix, e.g. twinloop/mortard
	StateToken      string        // bearer token for the snapshot endpoint
	BackupInterval  time.Duration // periodic checkpoint push; 0 disables
	TickInterval    time.Duration // world sim tick cadence; 0 disables
	DailySeedSecret string        // sunderd only: see internal/delve/seed.go
}

// Load reads the environment. Invalid numerics fail fast: a node with a
// broken configuration must refuse to boot rather than half-run.
func Load() Config {
	return Config{
		Addr:            env("ADDR", ":8091"),
		DataDir:         env("DATA_DIR", "./data"),
		MigrationsDir:   env("MIGRATIONS_DIR", "./migrations"),
		StateEndpoint:   env("STATE_ENDPOINT", ""),
		StatePrefix:     env("STATE_PREFIX", ""),
		StateToken:      env("STATE_TOKEN", ""),
		BackupInterval:  seconds("BACKUP_INTERVAL_SEC", time.Hour),
		TickInterval:    seconds("TICK_INTERVAL_SEC", 0),
		DailySeedSecret: env("DAILY_SEED_SECRET", ""),
	}
}

// EnvOr reads one variable with a fallback; exported for callers that
// need a string option outside the shared struct.
func EnvOr(key, fallback string) string { return env(key, fallback) }

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func seconds(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		panic(fmt.Sprintf("%s must be a non-negative integer of seconds (got %q)", key, raw))
	}
	return time.Duration(n) * time.Second
}
