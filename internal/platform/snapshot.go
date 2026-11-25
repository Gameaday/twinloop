package platform

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SnapshotClient is the zero-mount state layer (design invariant 2): each
// service hydrates its database from, and checkpoints it to, a REST
// endpoint backed by Methuselah. Contract in docs/STATE.md:
//
//	GET    {endpoint}/{prefix}/snapshot   latest snapshot bytes (404 = none yet)
//	PUT    {endpoint}/{prefix}/snapshot   replace the latest snapshot
//	PUT    {endpoint}/{prefix}/lock       acquire the boot lock (412 = held)
//	DELETE {endpoint}/{prefix}/lock       release the boot lock
type SnapshotClient struct {
	Endpoint string
	Token    string
	Prefix   string
	HTTP     *http.Client
}

const maxSnapshotBytes = 2 << 30 // 2 GiB ceiling for the draft; raise when worlds grow

func (c *SnapshotClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *SnapshotClient) enabled() bool { return c.Endpoint != "" && c.Prefix != "" }

func (c *SnapshotClient) url(name string) string {
	return fmt.Sprintf("%s/%s/%s", c.Endpoint, c.Prefix, name)
}

func (c *SnapshotClient) authorize(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

// Hydrate restores app.db from the remote snapshot when the local data
// directory has no database yet. It is a no-op when snapshotting is not
// configured or the DB already exists, so a dev box just works. Cold boots
// from two containers at once serialize on the remote boot lock.
func (c *SnapshotClient) Hydrate(ctx context.Context, dataDir string) error {
	dbPath := filepath.Join(dataDir, "app.db")
	if _, err := os.Stat(dbPath); err == nil {
		return nil // already hydrated
	}
	if !c.enabled() {
		return nil
	}
	release, err := c.acquireLock(ctx)
	if err != nil {
		return err
	}
	released := false
	defer func() {
		if released {
			return
		}
		rctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = release(rctx) // best effort; the lock also carries an expiry
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("snapshot"), nil)
	if err != nil {
		return fmt.Errorf("snapshot get: %w", err)
	}
	c.authorize(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("snapshot get: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil // first boot on a fresh endpoint
	case http.StatusOK:
		// fall through to restore
	default:
		return fmt.Errorf("snapshot get: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSnapshotBytes+1))
	if err != nil {
		return fmt.Errorf("snapshot read: %w", err)
	}
	if len(body) > maxSnapshotBytes {
		return fmt.Errorf("snapshot larger than %d bytes", maxSnapshotBytes)
	}
	tmp := dbPath + ".hydrating"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return fmt.Errorf("snapshot write: %w", err)
	}
	if err := os.Rename(tmp, dbPath); err != nil {
		return fmt.Errorf("snapshot restore: %w", err)
	}
	released = true
	slog.Info("state hydrated from snapshot", "bytes", len(body))
	return nil
}

func (c *SnapshotClient) acquireLock(ctx context.Context) (release func(context.Context) error, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.url("lock"), nil)
	if err != nil {
		return nil, fmt.Errorf("boot lock: %w", err)
	}
	c.authorize(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("boot lock: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusCreated:
		// acquired
	case http.StatusPreconditionFailed, http.StatusConflict:
		return nil, errors.New("boot lock held: another instance is hydrating; retry shortly")
	default:
		return nil, fmt.Errorf("boot lock: unexpected status %d", resp.StatusCode)
	}
	return func(rctx context.Context) error {
		req, err := http.NewRequestWithContext(rctx, http.MethodDelete, c.url("lock"), nil)
		if err != nil {
			return err
		}
		c.authorize(req)
		resp, err := c.httpClient().Do(req)
		if err != nil {
			return fmt.Errorf("boot lock release: %w", err)
		}
		resp.Body.Close()
		return nil
	}, nil
}

// CheckpointSnapshot writes a consistent copy of db via VACUUM INTO and
// uploads it. VACUUM INTO never copies a half-written database, so this is
// safe to call on the backup interval and again at SIGTERM.
func CheckpointSnapshot(ctx context.Context, db *sql.DB, c *SnapshotClient, dataDir string) error {
	snap := filepath.Join(dataDir, "snapshot.db")
	if err := os.Remove(snap); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear old snapshot: %w", err)
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", snap); err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}
	if !c.enabled() {
		return nil
	}
	body, err := os.ReadFile(snap)
	if err != nil {
		return fmt.Errorf("snapshot read: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.url("snapshot"), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("snapshot put: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	c.authorize(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("snapshot put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("snapshot put: unexpected status %d", resp.StatusCode)
	}
	slog.Info("snapshot pushed", "bytes", len(body))
	return nil
}
