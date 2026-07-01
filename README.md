<div align="center">

# ctc

**container → compose**

A terminal UI that turns your running Docker containers into a `docker-compose.yml` — with full runtime fidelity.

```
curl https://ctc.dothis.online | bash
```

</div>

---

## What it does

You have containers running that were started with a long `docker run` command you no longer remember. `ctc` reads them straight from the Docker daemon, lets you pick which ones to capture, and writes out a clean, accurate `docker-compose.yml`.

- **Pick from a live list** — every container (running and stopped), multi-select with the spacebar.
- **Full runtime fidelity** — image, ports, volumes, networks, command, entrypoint, restart policy, labels, capabilities, and more.
- **GPU & runtime aware** — captures `--gpus`, `--runtime nvidia`, and `--network host/none/container:` correctly.
- **Only your env vars** — filters out image-default environment variables, keeping just the ones you passed with `-e`.
- **Preview, then save** — review the generated YAML, tweak it in `$EDITOR`, and write it to `docker-compose.yml`.

## Install

Requires **Go 1.21+** on your `PATH` (the installer builds from source).

```bash
curl https://ctc.dothis.online | bash
```

The script builds `ctc`, installs it to your Go bin directory, and launches it. To install without launching:

```bash
go install github.com/chaiops/ctc@latest
```

## Usage

Run it:

```bash
ctc
```

### Keys

**Container list**

| Key            | Action              |
| -------------- | ------------------- |
| `↑` / `↓`      | Move cursor         |
| `space`        | Toggle selection    |
| `a`            | Select / clear all  |
| `enter`        | Build compose file  |
| `q`            | Quit                |

**Preview**

| Key            | Action                    |
| -------------- | ------------------------- |
| `↑` / `↓`      | Scroll                    |
| `e`            | Edit in `$EDITOR`         |
| `s`            | Save to `docker-compose.yml` |
| `esc`          | Back to list              |
| `q`            | Quit                      |

## What gets captured

| `docker run` flag / setting | compose field |
| --------------------------- | ------------- |
| image                       | `image` |
| `-e KEY=VALUE`              | `environment` *(image defaults filtered out)* |
| `-p host:container`         | `ports` |
| `-v` / `--mount` (bind)     | `volumes` *(read-only mounts keep `:ro`)* |
| named volumes               | `volumes` + top-level `volumes:` |
| `--tmpfs`                   | `tmpfs` |
| `--network`                 | `networks` + top-level `networks:` |
| `--network host/none`       | `network_mode` |
| command / entrypoint        | `command` / `entrypoint` |
| `--restart`                 | `restart` |
| `-l` / labels               | `labels` |
| `--privileged`              | `privileged` |
| `--cap-add` / `--cap-drop`  | `cap_add` / `cap_drop` |
| `--runtime`                 | `runtime` |
| `--gpus`                    | `deploy.resources.reservations.devices` |

The output targets modern Compose (v3-style, no deprecated `version:` key).

## How it works

`ctc` shells out to the `docker` CLI (`docker inspect`, `docker network inspect`, …) and transforms the JSON into a compose file. It never talks to the daemon socket directly, so it works anywhere the `docker` command does.

```
docker ps ─▶ select ─▶ docker inspect ─▶ build YAML ─▶ preview ─▶ edit / save
```

Under the hood it is split into small, focused packages:

- `internal/docker` — runs the CLI and parses its JSON.
- `internal/compose` — a pure transform from container structs to compose YAML.
- `internal/tui` — the [Bubble Tea](https://github.com/charmbracelet/bubbletea) interface.

## Development

```bash
git clone https://github.com/chaiops/ctc
cd ctc
go test ./...     # run the suite
go run .          # launch against your local Docker
go build -o ctc . # produce a binary
```

## Limitations

- If you passed `-e FOO=<value>` where `<value>` is byte-identical to the image's own default for `FOO`, it can't be distinguished from the default and is dropped — Docker records no provenance for env vars.
- Linux and macOS only.

## License

MIT
