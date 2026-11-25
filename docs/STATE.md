# State, hydration, and backup contract

Zero host mounts (design invariant 2): every container hydrates its own
state on boot from a REST endpoint backed by Methuselah, and checkpoints
back on the backup interval and again on SIGTERM. A node needs Docker and
a `.env`, nothing else.

## Endpoint contract

A small service on (or in front of) Methuselah serves:

| Method | Path | Behavior |
|---|---|---|
| GET | `{endpoint}/{prefix}/snapshot` | Latest snapshot bytes; 404 when none yet |
| PUT | `{endpoint}/{prefix}/snapshot` | Replace the latest snapshot; history via Synology snapshots underneath |
| PUT | `{endpoint}/{prefix}/lock` | Acquire the boot lock; 412/409 when already held |
| DELETE | `{endpoint}/{prefix}/lock` | Release the boot lock |

Auth: `Authorization: Bearer $STATE_TOKEN` when `STATE_TOKEN` is set.
The first-draft clients talk plain REST over the LAN; if the storage side
becomes S3-compatible (Garage/RustFS), this shim lives in front of it and
its contract stays identical, so the game services never change.

## Semantics

- **Hydrate-if-empty.** On boot, if `DATA_DIR/app.db` does not exist, the
  client serializes on the boot lock, GETs the snapshot, and writes it.
  With no remote snapshot (404), it starts fresh, migrates, and begins
  membership-by-checkpoint. A competing cold boot under a held lock fails
  closed instead of double-hydrating: a divergent boot is worse than a
  short outage.
- **Checkpoint = `VACUUM INTO`.** A fresh copy of the database is written
  locally then uploaded; a half-written database can never be uploaded.
- **Shutdown finalizes once.** SIGTERM stops tickers, drains HTTP, then
  makes exactly one final checkpoint.
- **The boot lock also TTL-expires** so a crashed hydrator cannot wedge a
  fresh node forever (the shim enforces the TTL).

## Restore runbook (fresh Pi or corrupted volume)

1. Fill `.env` (endpoint, token, prefix).
2. `docker compose up -d` (or deploy the stack in Arcane).
3. Watch the log for `state hydrated from snapshot` before routing
   traffic. First boot on an empty endpoint logs a fresh start instead.
4. If state was wrong: roll the Synology snapshot back on Methuselah,
   delete the local volume, then re-up: hydrate picks up the rolled-back
   snapshot.

## Backup layering

1. Live DB: local container volume on the write node (NVMe/SSD: Charon or
   Sirius; never Vega).
2. Periodic + shutdown checkpoints: Methuselah REST endpoint.
3. Synology snapshots of the endpoint share: point-in-time rollback.
4. **RAID is not backup.** restic (or rclone) a second copy to a second
   destination before anything user-precious depends on the world sim.
