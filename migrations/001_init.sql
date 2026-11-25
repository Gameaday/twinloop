-- 001_init.sql: twinloop v1 draft schema (docs/DESIGN.md, section 7).
-- The cross-game bridge tables exist as a seam only; both games run
-- independently until the bridge deliverable after v1 playtesting.

CREATE TABLE accounts (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  recovery_code_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  expires_at TEXT NOT NULL
);

CREATE TABLE characters (
  id INTEGER PRIMARY KEY,
  account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  location_x INTEGER NOT NULL DEFAULT 0,
  location_y INTEGER NOT NULL DEFAULT 0,
  last_seen_at TEXT
);

CREATE TABLE runs (
  id INTEGER PRIMARY KEY,
  account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  day_index INTEGER NOT NULL,
  seed_fingerprint TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','dead','extracted')),
  state_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX idx_runs_account ON runs(account_id, day_index);

-- Corpse remnants persist inside the shared daily dungeon. Claims are
-- atomic: at most one run holds claimed_by_run_id; reservation lapses
-- at reserved_until; an unbanked claimer's death returns the remnant.
CREATE TABLE corpse_remnants (
  id INTEGER PRIMARY KEY,
  day_index INTEGER NOT NULL,
  floor INTEGER NOT NULL,
  trigger_kind TEXT NOT NULL CHECK (trigger_kind IN ('trap','enemy')),
  trigger_key TEXT NOT NULL,
  item_id TEXT NOT NULL,
  epitaph TEXT NOT NULL,
  claimed_by_run_id INTEGER REFERENCES runs(id),
  reserved_until TEXT,
  consumed_at TEXT
);
CREATE INDEX idx_remnants_lookup
  ON corpse_remnants(day_index, floor, trigger_kind, trigger_key);

CREATE TABLE character_ledger (
  id INTEGER PRIMARY KEY,
  character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  tick_id INTEGER NOT NULL,
  at TEXT NOT NULL,
  kind TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_ledger_character ON character_ledger(character_id, id);

CREATE TABLE settlement_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  stockpile_json TEXT NOT NULL DEFAULT '{}',
  buildings_json TEXT NOT NULL DEFAULT '{}',
  dungeon_threat REAL NOT NULL DEFAULT 0,
  chronicle_cursor INTEGER NOT NULL DEFAULT 0,
  last_tick_at TEXT
);

-- Idempotent bridge contract: retries never double-credit an event.
CREATE TABLE expedition_reports (
  id INTEGER PRIMARY KEY,
  run_id INTEGER NOT NULL,
  event_id INTEGER NOT NULL,
  payload_json TEXT NOT NULL,
  applied_at TEXT,
  UNIQUE (run_id, event_id)
);
