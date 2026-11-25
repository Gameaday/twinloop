package platform

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// newFakeState implements the docs/STATE.md contract exactly, so a
// failure here predicts a failure against the real Methuselah shim.
func newFakeState(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	snapshots := map[string][]byte{}
	locked := false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /twinloop/test/snapshot", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, ok := snapshots["snapshot"]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	})
	mux.HandleFunc("PUT /twinloop/test/snapshot", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		snapshots["snapshot"] = body
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("PUT /twinloop/test/lock", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if locked {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		locked = true
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /twinloop/test/lock", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		locked = false
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestHydrateCheckpointRoundTrip(t *testing.T) {
	srv := newFakeState(t)
	c := &SnapshotClient{Endpoint: srv.URL, Prefix: "twinloop/test"}
	ctx := context.Background()

	// First boot against an empty endpoint: nothing to hydrate.
	dir1 := t.TempDir()
	if err := c.Hydrate(ctx, dir1); err != nil {
		t.Fatalf("cold hydrate: %v", err)
	}
	db1, err := Open(dir1, testMigrations)
	if err != nil {
		t.Fatalf("cold open: %v", err)
	}
	if _, err := db1.Exec(`INSERT INTO accounts (name, password_hash, recovery_code_hash) VALUES ('carl','h','r')`); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if err := CheckpointSnapshot(ctx, db1, c, dir1); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// A fresh node restores the world from the remote and reopens cleanly.
	dir2 := t.TempDir()
	if err := c.Hydrate(ctx, dir2); err != nil {
		t.Fatalf("restore hydrate: %v", err)
	}
	db2, err := Open(dir2, testMigrations)
	if err != nil {
		t.Fatalf("restored open: %v", err)
	}
	defer db2.Close()
	var name string
	if err := db2.QueryRow(`SELECT name FROM accounts`).Scan(&name); err != nil || name != "carl" {
		t.Fatalf("restored account = %q (err %v)", name, err)
	}
}

func TestHydrateSkipsExistingDatabase(t *testing.T) {
	srv := newFakeState(t)
	c := &SnapshotClient{Endpoint: srv.URL, Prefix: "twinloop/test"}
	dir := t.TempDir()
	db, err := Open(dir, testMigrations)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	// Local DB exists: hydrate must be a no-op that never clobbers it,
	// even though the remote has a snapshot.
	if err := CheckpointSnapshot(context.Background(), db, c, dir); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if err := c.Hydrate(context.Background(), dir); err != nil {
		t.Fatalf("hydrate over existing: %v", err)
	}
}

func TestHydrateSerializesOnBootLock(t *testing.T) {
	srv := newFakeState(t)
	c := &SnapshotClient{Endpoint: srv.URL, Prefix: "twinloop/test"}
	ctx := context.Background()
	// Someone else holds the boot lock: a competing cold boot must fail
	// closed instead of double-hydrating (red-team hydration race).
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, srv.URL+"/twinloop/test/lock", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if err := c.Hydrate(ctx, t.TempDir()); err == nil {
		t.Fatal("cold boot under a held lock must fail, not double-hydrate")
	}
}

func TestSnapshotFileIsWrittenAlongsideDatabase(t *testing.T) {
	srv := newFakeState(t)
	c := &SnapshotClient{Endpoint: srv.URL, Prefix: "twinloop/test"}
	dir := t.TempDir()
	db, err := Open(dir, testMigrations)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := CheckpointSnapshot(context.Background(), db, c, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "snapshot.db")); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}
}
