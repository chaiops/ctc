# ctc — Container to Compose (Design)

Date: 2026-07-01

## Purpose

A terminal UI that converts one or more existing Docker containers into a
`docker-compose.yml` file with full runtime fidelity. User selects containers
from a list, previews the generated YAML, optionally edits it in `$EDITOR`, and
saves it.

## Language & Stack

- Go + Bubble Tea (TUI).
- Data source: shell out to the `docker` CLI (`docker ps`, `docker inspect`,
  `docker network inspect`, `docker volume inspect`). Parse JSON output.
- Distributed as a single static binary.

## Architecture

Four units with clear boundaries:

1. **`docker` pkg** — runs docker CLI commands, parses JSON into typed structs
   (`Container`, `Network`, `Volume`). No TUI knowledge. Testable with fixture
   JSON.
2. **`compose` pkg** — pure transform: `[]Container + related Networks/Volumes
   → composeFile struct → YAML bytes`. No IO, no docker calls. Fully
   unit-testable.
3. **`tui` pkg** — Bubble Tea model. Screens: list-select → preview. Owns state
   and keybindings.
4. **`main`** — wires the packages, handles `$EDITOR` spawn and file write.

## Data Flow

```
docker ps -a --format json         → []containerSummary
  ↓ (user checks N containers)
docker inspect <ids>               → []Container (full config)
docker network/volume inspect      → attached Networks, Volumes
  ↓
compose.Build(containers, nets, vols) → composeFile struct → YAML
  ↓
preview screen (scrollable)
  ↓ 'e'  → write temp .yml, spawn $EDITOR, reload on exit
  ↓ 's'  → write to ./docker-compose.yml (prompt if exists)
```

## Field Mapping (`docker inspect` → compose)

- `Config.Image` → `image`
- `Config.Env` → `environment`
- `NetworkSettings.Ports` / `HostConfig.PortBindings` → `ports`
- `Mounts` → `volumes` (bind = host:container path; named → top-level `volumes:`)
- `NetworkSettings.Networks` → `networks` (+ top-level `networks:`)
- `Config.Cmd` → `command`
- `Config.Entrypoint` → `entrypoint`
- `HostConfig.RestartPolicy` → `restart`
- `Config.Labels` → `labels`
- `HostConfig.Privileged` → `privileged`
- `HostConfig.CapAdd` / `CapDrop` → `cap_add` / `cap_drop`
- `HostConfig.Runtime` (e.g. `nvidia`) → `runtime`
- `HostConfig.DeviceRequests` (from `--gpus`) → `deploy.resources.reservations.devices`
  (driver, count, capabilities). `--gpus all` appears as a DeviceRequests entry
  with driver `nvidia`, `Count: -1`, `Capabilities: [[gpu]]`. Runtime and
  DeviceRequests are mapped independently — a container may have either or both.

Compose target: v3-style, no top-level `version:` key (deprecated in current
Compose).

Expansion is full: every attached network, named volume, port, env var, and
command is expanded into the service and top-level blocks as needed.

## Screens & Keybinds

**List screen:** rows show name, image, status. `↑/↓` move, `space` toggle
check, `a` toggle all, `enter` confirm → build + preview, `q` quit.

**Preview screen:** scrollable YAML. `↑/↓` / `pgup` / `pgdn` scroll, `e` edit in
`$EDITOR`, `s` save, `esc` back to list, `q` quit.

## Error Handling

- docker CLI missing / daemon down → fatal message on launch, exit 1.
- No containers → empty-state message, not a crash.
- `inspect` fails for one container → skip it, warn in preview header, continue
  with the rest.
- `$EDITOR` unset → fall back to `vi`; spawn fails → error message, stay in
  preview.
- Save target exists → confirm overwrite (`y/n`).
- Malformed YAML after external edit → reload raw text, show parse warning,
  don't block save.

## Testing

- `docker` pkg: fixture JSON files → parse assertions.
- `compose` pkg: table + golden-file tests, container struct → expected YAML.
  This is the core logic and gets the heaviest coverage (including runtime/GPU
  mapping).
- `tui` pkg: model update tests (key msg → state transition) via Bubble Tea
  test helpers.
- No live-docker integration test in CI; manual smoke test.

## Non-Goals (YAGNI)

- No secret detection / redaction.
- No multi-host or swarm support.
- No reverse round-trip (compose → container).
- No config persistence between runs.
