# Twin Loop (twinloopgames.com)

Two games that feed one world, at two speeds:

| Game | Daemon | Loop | Feel |
|---|---|---|---|
| **Mortar** | `mortard` | Slow, tick-driven settlement sim. Queue actions, world resolves on each tick, Chronicle tells what happened. | Defense, building, permanence |
| **Sunder** | `sunderd` | Fast, procedural screen-based dungeon crawler with daily shared-seed runs. | Attack, descent, risk |

Progress in one shapes the other, but only after both loops are independently fun. See `docs/DESIGN.md` for the full design, the red-team findings, and the phased build order.

## Core invariants

1. **Server is the source of truth.** Clients send intents; the server validates, rolls, and stores.
2. **Zero host-level mounts.** State hydrates on container boot from a snapshot endpoint (Methuselah) via Compose env vars. A new node needs Docker and a `.env`, nothing else.
3. **Idle costs zero.** No loop runs when nobody is playing.
4. **Actions cost real-world minutes, never ticks.** Tick cadence can change (1/day, 1/3h, near real-time) in the future without changing action pacing.
5. **Ad-free, non-predatory.** Any supporter tier grants cosmetics only, never leaderboard power.

## Layout

```
cmd/mortard         settlement sim daemon (tick loop, digests)
cmd/sunderd         dungeon daemon (daily seeds, floor generation)
internal/platform   config-from-env, SQLite (WAL), snapshot hydrate/push, tick loop
internal/world      duration-based action queue, ledger digests
internal/delve      HMAC daily seeds, deterministic screen-grid floor generation
migrations/         SQLite schema (applied in filename order, tracked)
deploy/compose/     Arcane-friendly compose stack + .env.example
docs/               DESIGN.md (design + red team), STATE.md (state contract)
```

## Quickstart

```sh
make test          # everything in CI except the image builds
make build         # native binaries into bin/
make run-mortar    # http://localhost:8091/healthz
make run-sunder    # http://localhost:8092/healthz
```

## CI/CD

- `ci.yml`: on every push/PR. gofmt, vet, tests, then both images built (not pushed) for linux/amd64 + linux/arm64.
- `release.yml`: on `v*` tags. Full test gate, then multi-arch images pushed to GHCR:
  - `ghcr.io/gameaday/twinloop-mortar:<tag>` (+ `latest` for final tags)
  - `ghcr.io/gameaday/twinloop-sunder:<tag>` (+ `latest`)

A deploy is: copy `deploy/compose/` + a filled `.env` to a node, `docker compose pull && docker compose up -d`. Rollback is `TAG=<old>` in `.env`.

## Runtime images

Both images are `FROM scratch`: no shell, no libc, no package manager. `modernc.org/sqlite` is pure Go, so there is no CGO and cross-compiling for arm64 on amd64 CI runners is trivial.

## Fleet placement (draft)

| Node | Role |
|---|---|
| Methuselah (DS1621+) | Snapshot/backup endpoint (see `docs/STATE.md`) |
| Charon (x86_64) | Forgejo + runner, local registry cache; suspend disabled |
| Sirius (Pi 5, USB SSD) | Traefik, Arcane management; tick writer fallback |
| Altair (Pi 4, 4GB) | App replicas, light workload |
| Vega (Pi 4, 2GB, SD) | Read-only territory: no stateful workload, ever |

## License

All rights reserved. This repository is proprietary; it may move to a
self-hosted or private code host later. Do not redistribute copies.
