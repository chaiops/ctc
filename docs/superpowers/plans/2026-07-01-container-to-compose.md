# ctc — Container to Compose Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A terminal UI that converts selected running/stopped Docker containers into a `docker-compose.yml` with full runtime fidelity.

**Architecture:** Go + Bubble Tea. Three pure/near-pure library packages (`docker`, `compose`, `tui`) wired by `main`. The `docker` package shells out to the docker CLI and parses JSON; `compose` is a pure transform from parsed structs to YAML; `tui` is the Bubble Tea model; `main` handles `$EDITOR` and file IO.

**Tech Stack:** Go 1.25, Bubble Tea (`github.com/charmbracelet/bubbletea`), `gopkg.in/yaml.v3`, stdlib `os/exec` + `encoding/json`.

## Global Constraints

- Module path: `github.com/centerseat/ctc`.
- Go 1.25.
- Compose output: v3-style services, **no** top-level `version:` key.
- The `compose` package must have **zero** IO and **zero** docker dependencies — pure functions only.
- The `docker` package must be parseable from fixture JSON without a live daemon (parsing split from command execution).
- All tests run with `go test ./...` and must pass with no network/daemon access.

---

## File Structure

- `go.mod` — module definition.
- `internal/docker/types.go` — `Container`, `Network`, `Volume`, `ContainerSummary` structs mirroring `docker inspect` JSON.
- `internal/docker/parse.go` — pure JSON→struct parsers (no exec).
- `internal/docker/client.go` — CLI execution (`ps`, `inspect`, `network inspect`, `volume inspect`).
- `internal/docker/parse_test.go`, `internal/docker/testdata/*.json` — fixtures + parse tests.
- `internal/compose/model.go` — `File`, `Service`, `DeviceRequest` output structs with yaml tags.
- `internal/compose/build.go` — `Build(containers, networks, volumes) File` transform.
- `internal/compose/build_test.go`, `internal/compose/testdata/*.golden.yml` — golden tests.
- `internal/tui/model.go` — Bubble Tea model, screens, keybinds.
- `internal/tui/model_test.go` — update-function state-transition tests.
- `main.go` — wiring, `$EDITOR` spawn, file write, overwrite prompt.

---

### Task 1: Module + docker types and parsers

**Files:**
- Create: `go.mod`, `internal/docker/types.go`, `internal/docker/parse.go`
- Test: `internal/docker/parse_test.go`, `internal/docker/testdata/inspect_gpu.json`

**Interfaces:**
- Produces:
  - `type ContainerSummary struct { ID, Names, Image, State, Status string }`
  - `type Container struct { ID, Name string; Config Config; HostConfig HostConfig; NetworkSettings NetworkSettings; Mounts []Mount }`
  - `Config{ Image string; Env []string; Cmd []string; Entrypoint []string; Labels map[string]string }`
  - `HostConfig{ RestartPolicy RestartPolicy; Privileged bool; CapAdd []string; CapDrop []string; Runtime string; DeviceRequests []DeviceRequest; PortBindings map[string][]PortBinding }`
  - `RestartPolicy{ Name string; MaximumRetryCount int }`
  - `DeviceRequest{ Driver string; Count int; Capabilities [][]string }`
  - `PortBinding{ HostIP, HostPort string }`
  - `Mount{ Type, Name, Source, Destination string; RW bool }`
  - `NetworkSettings{ Networks map[string]NetworkEndpoint }`, `NetworkEndpoint{ Aliases []string }`
  - `type Network struct { Name, Driver string }`, `type Volume struct { Name, Driver string }`
  - `func ParseInspect(b []byte) ([]Container, error)` — `docker inspect` returns a JSON array.
  - `func ParsePS(b []byte) ([]ContainerSummary, error)` — parses newline-delimited `{"ID":...}` JSON objects (from `docker ps --format '{{json .}}'`).

- [ ] **Step 1: Init module**

```bash
cd /home/anuj/code/ctc
go mod init github.com/centerseat/ctc
```

- [ ] **Step 2: Add fixture** `internal/docker/testdata/inspect_gpu.json`

```json
[
  {
    "Id": "abc123",
    "Name": "/web",
    "Config": {
      "Image": "nginx:1.27",
      "Env": ["FOO=bar", "BAZ=qux"],
      "Cmd": ["nginx", "-g", "daemon off;"],
      "Entrypoint": null,
      "Labels": {"com.example.role": "frontend"}
    },
    "HostConfig": {
      "RestartPolicy": {"Name": "unless-stopped", "MaximumRetryCount": 0},
      "Privileged": false,
      "CapAdd": ["NET_ADMIN"],
      "CapDrop": null,
      "Runtime": "nvidia",
      "DeviceRequests": [
        {"Driver": "nvidia", "Count": -1, "Capabilities": [["gpu"]]}
      ],
      "PortBindings": {"80/tcp": [{"HostIp": "0.0.0.0", "HostPort": "8080"}]}
    },
    "NetworkSettings": {
      "Networks": {"appnet": {"Aliases": ["web"]}}
    },
    "Mounts": [
      {"Type": "bind", "Source": "/srv/html", "Destination": "/usr/share/nginx/html", "RW": true},
      {"Type": "volume", "Name": "datavol", "Destination": "/data", "RW": true}
    ]
  }
]
```

- [ ] **Step 3: Write the failing test** `internal/docker/parse_test.go`

```go
package docker

import (
	"os"
	"testing"
)

func TestParseInspectGPU(t *testing.T) {
	b, err := os.ReadFile("testdata/inspect_gpu.json")
	if err != nil {
		t.Fatal(err)
	}
	cs, err := ParseInspect(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("want 1 container, got %d", len(cs))
	}
	c := cs[0]
	if c.Name != "/web" {
		t.Errorf("name: got %q", c.Name)
	}
	if c.Config.Image != "nginx:1.27" {
		t.Errorf("image: got %q", c.Config.Image)
	}
	if c.HostConfig.Runtime != "nvidia" {
		t.Errorf("runtime: got %q", c.HostConfig.Runtime)
	}
	if len(c.HostConfig.DeviceRequests) != 1 || c.HostConfig.DeviceRequests[0].Count != -1 {
		t.Errorf("device requests: got %+v", c.HostConfig.DeviceRequests)
	}
	if c.HostConfig.PortBindings["80/tcp"][0].HostPort != "8080" {
		t.Errorf("port binding: got %+v", c.HostConfig.PortBindings)
	}
	if c.Mounts[1].Name != "datavol" {
		t.Errorf("mount name: got %q", c.Mounts[1].Name)
	}
}

func TestParsePS(t *testing.T) {
	in := []byte(`{"ID":"abc","Names":"web","Image":"nginx:1.27","State":"running","Status":"Up 2h"}
{"ID":"def","Names":"db","Image":"postgres:16","State":"exited","Status":"Exited (0)"}`)
	got, err := ParsePS(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Image != "postgres:16" {
		t.Fatalf("got %+v", got)
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/docker/`
Expected: FAIL — undefined `ParseInspect` / `ParsePS`.

- [ ] **Step 5: Write `internal/docker/types.go`**

```go
package docker

type ContainerSummary struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

type Container struct {
	ID              string `json:"Id"`
	Name            string `json:"Name"`
	Config          Config
	HostConfig      HostConfig
	NetworkSettings NetworkSettings
	Mounts          []Mount
}

type Config struct {
	Image      string
	Env        []string
	Cmd        []string
	Entrypoint []string
	Labels     map[string]string
}

type HostConfig struct {
	RestartPolicy  RestartPolicy
	Privileged     bool
	CapAdd         []string
	CapDrop        []string
	Runtime        string
	DeviceRequests []DeviceRequest
	PortBindings   map[string][]PortBinding
}

type RestartPolicy struct {
	Name              string
	MaximumRetryCount int
}

type DeviceRequest struct {
	Driver       string
	Count        int
	Capabilities [][]string
}

type PortBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type Mount struct {
	Type        string
	Name        string
	Source      string
	Destination string
	RW          bool
}

type NetworkSettings struct {
	Networks map[string]NetworkEndpoint
}

type NetworkEndpoint struct {
	Aliases []string
}

type Network struct {
	Name   string
	Driver string
}

type Volume struct {
	Name   string
	Driver string
}
```

- [ ] **Step 6: Write `internal/docker/parse.go`**

```go
package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

// ParseInspect parses the JSON array emitted by `docker inspect`.
func ParseInspect(b []byte) ([]Container, error) {
	var cs []Container
	if err := json.Unmarshal(b, &cs); err != nil {
		return nil, fmt.Errorf("parse inspect: %w", err)
	}
	return cs, nil
}

// ParsePS parses newline-delimited JSON objects from
// `docker ps --format '{{json .}}'`.
func ParsePS(b []byte) ([]ContainerSummary, error) {
	var out []ContainerSummary
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var s ContainerSummary
		if err := json.Unmarshal(line, &s); err != nil {
			return nil, fmt.Errorf("parse ps line: %w", err)
		}
		out = append(out, s)
	}
	return out, sc.Err()
}
```

- [ ] **Step 7: Run tests**

Run: `go test ./internal/docker/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add go.mod internal/docker/
git commit -m "feat: docker inspect/ps JSON parsers and types"
```

---

### Task 2: docker CLI client

**Files:**
- Create: `internal/docker/client.go`
- Test: covered by manual smoke (no daemon in CI); add a table test for command construction only.

**Interfaces:**
- Consumes: types + parsers from Task 1.
- Produces:
  - `type Runner func(name string, args ...string) ([]byte, error)`
  - `func DefaultRunner(name string, args ...string) ([]byte, error)`
  - `func Available(run Runner) error` — returns error if docker CLI/daemon unusable.
  - `func List(run Runner) ([]ContainerSummary, error)`
  - `func Inspect(run Runner, ids []string) ([]Container, error)`
  - `func InspectNetwork(run Runner, name string) (Network, error)`
  - `func InspectVolume(run Runner, name string) (Volume, error)`

- [ ] **Step 1: Write the failing test** `internal/docker/client_test.go`

```go
package docker

import (
	"encoding/json"
	"testing"
)

func TestListUsesRunner(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		if name != "docker" || args[0] != "ps" {
			t.Fatalf("unexpected call: %s %v", name, args)
		}
		return []byte(`{"ID":"x","Names":"n","Image":"i","State":"running","Status":"Up"}`), nil
	}
	got, err := List(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "x" {
		t.Fatalf("got %+v", got)
	}
}

func TestInspectNetwork(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		b, _ := json.Marshal([]Network{{Name: "appnet", Driver: "bridge"}})
		return b, nil
	}
	n, err := InspectNetwork(run, "appnet")
	if err != nil {
		t.Fatal(err)
	}
	if n.Driver != "bridge" {
		t.Fatalf("got %+v", n)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/docker/ -run TestList`
Expected: FAIL — undefined `List`.

- [ ] **Step 3: Write `internal/docker/client.go`**

```go
package docker

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Runner executes an external command and returns its stdout.
type Runner func(name string, args ...string) ([]byte, error)

// DefaultRunner runs the real command via os/exec.
func DefaultRunner(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s %v: %w: %s", name, args, err, ee.Stderr)
		}
		return nil, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}

// Available checks the docker CLI and daemon are reachable.
func Available(run Runner) error {
	if _, err := run("docker", "version", "--format", "{{.Server.Version}}"); err != nil {
		return fmt.Errorf("docker unavailable: %w", err)
	}
	return nil
}

func List(run Runner) ([]ContainerSummary, error) {
	out, err := run("docker", "ps", "-a", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	return ParsePS(out)
}

func Inspect(run Runner, ids []string) ([]Container, error) {
	args := append([]string{"inspect"}, ids...)
	out, err := run("docker", args...)
	if err != nil {
		return nil, err
	}
	return ParseInspect(out)
}

func InspectNetwork(run Runner, name string) (Network, error) {
	out, err := run("docker", "network", "inspect", name)
	if err != nil {
		return Network{}, err
	}
	var ns []Network
	if err := json.Unmarshal(out, &ns); err != nil {
		return Network{}, fmt.Errorf("parse network inspect: %w", err)
	}
	if len(ns) == 0 {
		return Network{}, fmt.Errorf("network %q not found", name)
	}
	return ns[0], nil
}

func InspectVolume(run Runner, name string) (Volume, error) {
	out, err := run("docker", "volume", "inspect", name)
	if err != nil {
		return Volume{}, err
	}
	var vs []Volume
	if err := json.Unmarshal(out, &vs); err != nil {
		return Volume{}, fmt.Errorf("parse volume inspect: %w", err)
	}
	if len(vs) == 0 {
		return Volume{}, fmt.Errorf("volume %q not found", name)
	}
	return vs[0], nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/docker/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/docker/client.go internal/docker/client_test.go
git commit -m "feat: docker CLI client with injectable runner"
```

---

### Task 3: compose output model + builder

**Files:**
- Create: `internal/compose/model.go`, `internal/compose/build.go`
- Test: `internal/compose/build_test.go`, `internal/compose/testdata/gpu.golden.yml`

**Interfaces:**
- Consumes: `docker.Container`, `docker.Network`, `docker.Volume` from Task 1.
- Produces:
  - `type File struct { Services map[string]Service; Networks map[string]NetworkDef; Volumes map[string]VolumeDef }` with yaml tags (`services`, `networks`, `volumes`, `omitempty`).
  - `type Service struct { Image string; Environment []string; Ports []string; Volumes []string; Networks []string; Command []string; Entrypoint []string; Restart string; Labels map[string]string; Privileged bool; CapAdd []string; CapDrop []string; Runtime string; Deploy *Deploy }`
  - `type Deploy struct { Resources Resources }`, `Resources{ Reservations Reservations }`, `Reservations{ Devices []Device }`, `Device{ Driver string; Count int/string; Capabilities []string }`
  - `func Build(cs []docker.Container, nets []docker.Network, vols []docker.Volume) File`
  - `func (f File) YAML() ([]byte, error)`

- [ ] **Step 1: Golden fixture** `internal/compose/testdata/gpu.golden.yml`

```yaml
services:
    web:
        image: nginx:1.27
        command:
            - nginx
            - -g
            - daemon off;
        environment:
            - FOO=bar
            - BAZ=qux
        ports:
            - 0.0.0.0:8080:80/tcp
        volumes:
            - /srv/html:/usr/share/nginx/html
            - datavol:/data
        networks:
            - appnet
        restart: unless-stopped
        labels:
            com.example.role: frontend
        cap_add:
            - NET_ADMIN
        runtime: nvidia
        deploy:
            resources:
                reservations:
                    devices:
                        - driver: nvidia
                          count: all
                          capabilities:
                            - gpu
networks:
    appnet:
        driver: bridge
volumes:
    datavol:
        driver: local
```

- [ ] **Step 2: Write the failing test** `internal/compose/build_test.go`

```go
package compose

import (
	"os"
	"testing"

	"github.com/centerseat/ctc/internal/docker"
)

func sampleContainer() docker.Container {
	return docker.Container{
		Name: "/web",
		Config: docker.Config{
			Image:  "nginx:1.27",
			Env:    []string{"FOO=bar", "BAZ=qux"},
			Cmd:    []string{"nginx", "-g", "daemon off;"},
			Labels: map[string]string{"com.example.role": "frontend"},
		},
		HostConfig: docker.HostConfig{
			RestartPolicy:  docker.RestartPolicy{Name: "unless-stopped"},
			CapAdd:         []string{"NET_ADMIN"},
			Runtime:        "nvidia",
			DeviceRequests: []docker.DeviceRequest{{Driver: "nvidia", Count: -1, Capabilities: [][]string{{"gpu"}}}},
			PortBindings:   map[string][]docker.PortBinding{"80/tcp": {{HostIP: "0.0.0.0", HostPort: "8080"}}},
		},
		NetworkSettings: docker.NetworkSettings{Networks: map[string]docker.NetworkEndpoint{"appnet": {}}},
		Mounts: []docker.Mount{
			{Type: "bind", Source: "/srv/html", Destination: "/usr/share/nginx/html"},
			{Type: "volume", Name: "datavol", Destination: "/data"},
		},
	}
}

func TestBuildGolden(t *testing.T) {
	f := Build(
		[]docker.Container{sampleContainer()},
		[]docker.Network{{Name: "appnet", Driver: "bridge"}},
		[]docker.Volume{{Name: "datavol", Driver: "local"}},
	)
	got, err := f.YAML()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/gpu.golden.yml")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("YAML mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/compose/`
Expected: FAIL — undefined `Build`.

- [ ] **Step 4: Add yaml dep**

```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 5: Write `internal/compose/model.go`**

```go
package compose

import "gopkg.in/yaml.v3"

type File struct {
	Services map[string]Service    `yaml:"services,omitempty"`
	Networks map[string]NetworkDef `yaml:"networks,omitempty"`
	Volumes  map[string]VolumeDef  `yaml:"volumes,omitempty"`
}

type Service struct {
	Image       string            `yaml:"image,omitempty"`
	Command     []string          `yaml:"command,omitempty"`
	Entrypoint  []string          `yaml:"entrypoint,omitempty"`
	Environment []string          `yaml:"environment,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Privileged  bool              `yaml:"privileged,omitempty"`
	CapAdd      []string          `yaml:"cap_add,omitempty"`
	CapDrop     []string          `yaml:"cap_drop,omitempty"`
	Runtime     string            `yaml:"runtime,omitempty"`
	Deploy      *Deploy           `yaml:"deploy,omitempty"`
}

type Deploy struct {
	Resources Resources `yaml:"resources"`
}
type Resources struct {
	Reservations Reservations `yaml:"reservations"`
}
type Reservations struct {
	Devices []Device `yaml:"devices"`
}
type Device struct {
	Driver       string   `yaml:"driver,omitempty"`
	Count        string   `yaml:"count,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty"`
}

type NetworkDef struct {
	Driver string `yaml:"driver,omitempty"`
}
type VolumeDef struct {
	Driver string `yaml:"driver,omitempty"`
}

func (f File) YAML() ([]byte, error) {
	return yaml.Marshal(f)
}
```

- [ ] **Step 6: Write `internal/compose/build.go`**

```go
package compose

import (
	"strings"

	"github.com/centerseat/ctc/internal/docker"
)

// Build converts inspected containers and their related networks/volumes into
// a compose File. Pure: no IO, no docker calls.
func Build(cs []docker.Container, nets []docker.Network, vols []docker.Volume) File {
	f := File{Services: map[string]Service{}}

	for _, c := range cs {
		name := strings.TrimPrefix(c.Name, "/")
		s := Service{
			Image:       c.Config.Image,
			Command:     c.Config.Cmd,
			Entrypoint:  c.Config.Entrypoint,
			Environment: c.Config.Env,
			Labels:      c.Config.Labels,
			Privileged:  c.HostConfig.Privileged,
			CapAdd:      c.HostConfig.CapAdd,
			CapDrop:     c.HostConfig.CapDrop,
			Runtime:     c.HostConfig.Runtime,
			Restart:     restart(c.HostConfig.RestartPolicy),
			Ports:       ports(c.HostConfig.PortBindings),
			Volumes:     mounts(c.Mounts),
			Networks:    networkNames(c.NetworkSettings.Networks),
		}
		if d := devices(c.HostConfig.DeviceRequests); d != nil {
			s.Deploy = &Deploy{Resources{Reservations{Devices: d}}}
		}
		f.Services[name] = s
	}

	if len(nets) > 0 {
		f.Networks = map[string]NetworkDef{}
		for _, n := range nets {
			f.Networks[n.Name] = NetworkDef{Driver: n.Driver}
		}
	}
	if len(vols) > 0 {
		f.Volumes = map[string]VolumeDef{}
		for _, v := range vols {
			f.Volumes[v.Name] = VolumeDef{Driver: v.Driver}
		}
	}
	return f
}

func restart(p docker.RestartPolicy) string {
	if p.Name == "" || p.Name == "no" {
		return ""
	}
	return p.Name
}

func ports(pb map[string][]docker.PortBinding) []string {
	if len(pb) == 0 {
		return nil
	}
	// Sort container-port keys for deterministic output.
	keys := make([]string, 0, len(pb))
	for k := range pb {
		keys = append(keys, k)
	}
	sortStrings(keys)
	var out []string
	for _, k := range keys {
		for _, b := range pb[k] {
			if b.HostIP != "" {
				out = append(out, b.HostIP+":"+b.HostPort+":"+k)
			} else {
				out = append(out, b.HostPort+":"+k)
			}
		}
	}
	return out
}

func mounts(ms []docker.Mount) []string {
	var out []string
	for _, m := range ms {
		src := m.Source
		if m.Type == "volume" {
			src = m.Name
		}
		out = append(out, src+":"+m.Destination)
	}
	return out
}

func networkNames(n map[string]docker.NetworkEndpoint) []string {
	if len(n) == 0 {
		return nil
	}
	out := make([]string, 0, len(n))
	for name := range n {
		out = append(out, name)
	}
	sortStrings(out)
	return out
}

func devices(reqs []docker.DeviceRequest) []Device {
	if len(reqs) == 0 {
		return nil
	}
	var out []Device
	for _, r := range reqs {
		caps := []string{}
		for _, group := range r.Capabilities {
			caps = append(caps, group...)
		}
		count := ""
		if r.Count == -1 {
			count = "all"
		} else if r.Count > 0 {
			count = itoa(r.Count)
		}
		out = append(out, Device{Driver: r.Driver, Count: count, Capabilities: caps})
	}
	return out
}
```

- [ ] **Step 7: Add small helpers** append to `internal/compose/build.go`

```go
import "sort"    // add to the import block; and "strconv"

func sortStrings(s []string) { sort.Strings(s) }
func itoa(n int) string      { return strconv.Itoa(n) }
```

(Fold `sort` and `strconv` into the existing import block rather than a second `import`.)

- [ ] **Step 8: Run test**

Run: `go test ./internal/compose/`
Expected: PASS. If the golden file differs only in field ordering/indent, update `gpu.golden.yml` to match `yaml.v3` output exactly, then re-run.

- [ ] **Step 9: Commit**

```bash
git add internal/compose/ go.mod go.sum
git commit -m "feat: pure compose builder with runtime/GPU mapping"
```

---

### Task 4: TUI model — list + preview

**Files:**
- Create: `internal/tui/model.go`
- Test: `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `docker.ContainerSummary`, `compose.File`.
- Produces:
  - `type Screen int` with `ScreenList`, `ScreenPreview`.
  - `func New(items []docker.ContainerSummary) Model`
  - `Model` implements `tea.Model` (`Init`, `Update`, `View`).
  - `func (m Model) Selected() []string` — IDs of checked rows.
  - Exposed for tests: `Model.screen Screen`, `Model.cursor int`, `Model.checked map[int]bool`, `Model.yaml string`, `func (m *Model) SetPreview(y string)`.
  - Messages: `type PreviewReadyMsg struct { YAML string }`.

- [ ] **Step 1: Add bubbletea dep**

```bash
go get github.com/charmbracelet/bubbletea
```

- [ ] **Step 2: Write the failing test** `internal/tui/model_test.go`

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/centerseat/ctc/internal/docker"
)

func newModel() Model {
	return New([]docker.ContainerSummary{
		{ID: "a", Names: "web", Image: "nginx"},
		{ID: "b", Names: "db", Image: "postgres"},
	})
}

func key(r rune) tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestToggleAndSelect(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(key(' ')) // check row 0
	m = m2.(Model)
	sel := m.Selected()
	if len(sel) != 1 || sel[0] != "a" {
		t.Fatalf("selected: %v", sel)
	}
}

func TestCursorDown(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m2.(Model)
	if m.cursor != 1 {
		t.Fatalf("cursor: %d", m.cursor)
	}
}

func TestPreviewReadySwitchesScreen(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(PreviewReadyMsg{YAML: "services: {}"})
	m = m2.(Model)
	if m.screen != ScreenPreview || m.yaml != "services: {}" {
		t.Fatalf("screen=%d yaml=%q", m.screen, m.yaml)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/tui/`
Expected: FAIL — undefined `New`.

- [ ] **Step 4: Write `internal/tui/model.go`**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/centerseat/ctc/internal/docker"
)

type Screen int

const (
	ScreenList Screen = iota
	ScreenPreview
)

type PreviewReadyMsg struct{ YAML string }

// BuildFunc is injected by main: given selected IDs, produce compose YAML.
type BuildFunc func(ids []string) tea.Cmd

type Model struct {
	items   []docker.ContainerSummary
	cursor  int
	checked map[int]bool
	screen  Screen
	yaml    string
	offset  int // preview scroll
	build   BuildFunc
	err     string
}

func New(items []docker.ContainerSummary) Model {
	return Model{items: items, checked: map[int]bool{}, screen: ScreenList}
}

// WithBuild attaches the compose-build command factory.
func (m Model) WithBuild(b BuildFunc) Model { m.build = b; return m }

func (m *Model) SetPreview(y string) { m.yaml = y; m.screen = ScreenPreview }

func (m Model) Selected() []string {
	var out []string
	for i, c := range m.items {
		if m.checked[i] {
			out = append(out, c.ID)
		}
	}
	return out
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PreviewReadyMsg:
		m.yaml = msg.YAML
		m.screen = ScreenPreview
		m.offset = 0
		return m, nil
	case tea.KeyMsg:
		if m.screen == ScreenList {
			return m.updateList(msg)
		}
		return m.updatePreview(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if len(m.Selected()) > 0 && m.build != nil {
			return m, m.build(m.Selected())
		}
	case tea.KeyRunes:
		switch msg.Runes[0] {
		case ' ':
			m.checked[m.cursor] = !m.checked[m.cursor]
		case 'a':
			all := len(m.Selected()) != len(m.items)
			for i := range m.items {
				m.checked[i] = all
			}
		case 'q':
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.offset > 0 {
			m.offset--
		}
	case tea.KeyDown:
		m.offset++
	case tea.KeyEsc:
		m.screen = ScreenList
	case tea.KeyRunes:
		if msg.Runes[0] == 'q' {
			return m, tea.Quit
		}
	}
	return m, nil
}
```

- [ ] **Step 5: Write `View`** append to `internal/tui/model.go`

```go
import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	if m.screen == ScreenPreview {
		var b strings.Builder
		b.WriteString("Preview  [e]dit  [s]ave  [esc] back  [q]uit\n\n")
		if m.err != "" {
			b.WriteString("! " + m.err + "\n\n")
		}
		lines := strings.Split(m.yaml, "\n")
		for i := m.offset; i < len(lines) && i < m.offset+30; i++ {
			b.WriteString(lines[i] + "\n")
		}
		return b.String()
	}
	var b strings.Builder
	b.WriteString("Select containers  [space] toggle  [a] all  [enter] build  [q]uit\n\n")
	for i, c := range m.items {
		cur := " "
		if i == m.cursor {
			cur = ">"
		}
		box := "[ ]"
		if m.checked[i] {
			box = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s %s %-20s %-25s %s\n", cur, box, c.Names, c.Image, c.State))
	}
	return b.String()
}
```

(Fold `fmt` and `strings` into the file's existing import block.)

- [ ] **Step 6: Run tests**

Run: `go test ./internal/tui/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: bubbletea list + preview model"
```

---

### Task 5: main wiring — build cmd, $EDITOR, save

**Files:**
- Create: `main.go`
- Test: `main_test.go` (unit-test the pure helpers only).

**Interfaces:**
- Consumes: `docker.*`, `compose.Build`, `tui.New/.WithBuild`, `tui.PreviewReadyMsg`.
- Produces:
  - `func relatedNetworks(run docker.Runner, cs []docker.Container) []docker.Network`
  - `func relatedVolumes(run docker.Runner, cs []docker.Container) []docker.Volume`
  - `func editInEditor(path string) error` — spawns `$EDITOR` (fallback `vi`).
  - `func save(path string, data []byte, confirm func() bool) (bool, error)` — prompts on existing file via `confirm`.

- [ ] **Step 1: Write the failing test** `main_test.go`

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveNewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	ok, err := save(p, []byte("services: {}\n"), func() bool { return false })
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "services: {}\n" {
		t.Fatalf("content=%q", b)
	}
}

func TestSaveExistingDeclined(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	os.WriteFile(p, []byte("old"), 0o644)
	ok, err := save(p, []byte("new"), func() bool { return false })
	if err != nil || ok {
		t.Fatalf("expected declined: ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "old" {
		t.Fatalf("should not overwrite, got %q", b)
	}
}

func TestSaveExistingConfirmed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	os.WriteFile(p, []byte("old"), 0o644)
	ok, err := save(p, []byte("new"), func() bool { return true })
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "new" {
		t.Fatalf("got %q", b)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test .`
Expected: FAIL — undefined `save`.

- [ ] **Step 3: Write `main.go`**

```go
package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/centerseat/ctc/internal/compose"
	"github.com/centerseat/ctc/internal/docker"
	"github.com/centerseat/ctc/internal/tui"
)

func main() {
	run := docker.DefaultRunner
	if err := docker.Available(run); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	items, err := docker.List(run)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "no containers found")
		os.Exit(0)
	}

	build := func(ids []string) tea.Cmd {
		return func() tea.Msg {
			cs, err := docker.Inspect(run, ids)
			if err != nil {
				return tui.PreviewReadyMsg{YAML: "# error: " + err.Error()}
			}
			nets := relatedNetworks(run, cs)
			vols := relatedVolumes(run, cs)
			y, err := compose.Build(cs, nets, vols).YAML()
			if err != nil {
				return tui.PreviewReadyMsg{YAML: "# error: " + err.Error()}
			}
			return tui.PreviewReadyMsg{YAML: string(y)}
		}
	}

	m := tui.New(items).WithBuild(build)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func relatedNetworks(run docker.Runner, cs []docker.Container) []docker.Network {
	seen := map[string]bool{}
	var out []docker.Network
	for _, c := range cs {
		for name := range c.NetworkSettings.Networks {
			if seen[name] {
				continue
			}
			seen[name] = true
			if n, err := docker.InspectNetwork(run, name); err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}

func relatedVolumes(run docker.Runner, cs []docker.Container) []docker.Volume {
	seen := map[string]bool{}
	var out []docker.Volume
	for _, c := range cs {
		for _, mnt := range c.Mounts {
			if mnt.Type != "volume" || mnt.Name == "" || seen[mnt.Name] {
				continue
			}
			seen[mnt.Name] = true
			if v, err := docker.InspectVolume(run, mnt.Name); err == nil {
				out = append(out, v)
			}
		}
	}
	return out
}

func editInEditor(path string) error {
	ed := os.Getenv("EDITOR")
	if ed == "" {
		ed = "vi"
	}
	cmd := exec.Command(ed, path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func save(path string, data []byte, confirm func() bool) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		if !confirm() {
			return false, nil
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
```

Note: `editInEditor` is wired into the preview screen in the next task; it is defined here and unused at this step — Go allows unused package-level funcs, so the build passes.

- [ ] **Step 4: Run tests + build**

Run: `go test . && go build ./...`
Expected: tests PASS, build succeeds.

- [ ] **Step 5: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: main wiring, network/volume expansion, save with overwrite prompt"
```

---

### Task 6: wire edit + save into preview screen

**Files:**
- Modify: `internal/tui/model.go` (preview key handling, save/edit hooks), `main.go` (inject hooks).

**Interfaces:**
- Consumes: `editInEditor`, `save` from Task 5.
- Produces:
  - `tui` adds injectable hooks on `Model`: `SaveFunc func(yaml string) tea.Cmd`, `EditFunc func(yaml string) tea.Cmd`, set via `WithSave` / `WithEdit`.
  - New messages: `type SavedMsg struct{ Path string; OK bool; Err string }`, `type EditedMsg struct{ YAML string; Err string }`.
  - Preview keys: `s` → `SaveFunc(m.yaml)`, `e` → `EditFunc(m.yaml)`.

- [ ] **Step 1: Write the failing test** append to `internal/tui/model_test.go`

```go
func TestPreviewSaveInvokesHook(t *testing.T) {
	called := false
	m := newModel().WithSave(func(y string) tea.Cmd {
		return func() tea.Msg { called = true; return SavedMsg{Path: "x", OK: true} }
	})
	m.SetPreview("services: {}")
	_, cmd := m.Update(key('s'))
	if cmd == nil {
		t.Fatal("expected save cmd")
	}
	cmd() // run it
	if !called {
		t.Fatal("save hook not called")
	}
}

func TestEditedMsgUpdatesYAML(t *testing.T) {
	m := newModel()
	m.SetPreview("old")
	m2, _ := m.Update(EditedMsg{YAML: "new"})
	m = m2.(Model)
	if m.yaml != "new" {
		t.Fatalf("yaml=%q", m.yaml)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestPreviewSave`
Expected: FAIL — undefined `WithSave` / `SavedMsg`.

- [ ] **Step 3: Add hooks + messages** in `internal/tui/model.go`

Add fields to `Model`:

```go
	save    SaveFunc
	edit    EditFunc
	status  string
```

Add types + builders:

```go
type SaveFunc func(yaml string) tea.Cmd
type EditFunc func(yaml string) tea.Cmd

type SavedMsg struct {
	Path string
	OK   bool
	Err  string
}
type EditedMsg struct {
	YAML string
	Err  string
}

func (m Model) WithSave(f SaveFunc) Model { m.save = f; return m }
func (m Model) WithEdit(f EditFunc) Model { m.edit = f; return m }
```

- [ ] **Step 4: Handle messages + keys** in `Update` add cases:

```go
	case SavedMsg:
		if msg.OK {
			m.status = "saved: " + msg.Path
		} else if msg.Err != "" {
			m.status = "save error: " + msg.Err
		} else {
			m.status = "save cancelled"
		}
		return m, nil
	case EditedMsg:
		if msg.Err != "" {
			m.status = "edit error: " + msg.Err
		} else {
			m.yaml = msg.YAML
			m.offset = 0
		}
		return m, nil
```

In `updatePreview`, extend the `tea.KeyRunes` handling:

```go
	case tea.KeyRunes:
		switch msg.Runes[0] {
		case 'q':
			return m, tea.Quit
		case 's':
			if m.save != nil {
				return m, m.save(m.yaml)
			}
		case 'e':
			if m.edit != nil {
				return m, m.edit(m.yaml)
			}
		}
```

(Replace the previous `if msg.Runes[0] == 'q'` block with this switch.)

In `View` preview branch, render `m.status` if set (below the header):

```go
		if m.status != "" {
			b.WriteString(m.status + "\n\n")
		}
```

- [ ] **Step 5: Run tui tests**

Run: `go test ./internal/tui/`
Expected: PASS.

- [ ] **Step 6: Inject hooks in `main.go`** — extend the model construction:

```go
	editFn := func(y string) tea.Cmd {
		return func() tea.Msg {
			tmp, err := os.CreateTemp("", "ctc-*.yml")
			if err != nil {
				return tui.EditedMsg{Err: err.Error()}
			}
			tmp.WriteString(y)
			tmp.Close()
			defer os.Remove(tmp.Name())
			if err := editInEditor(tmp.Name()); err != nil {
				return tui.EditedMsg{Err: err.Error()}
			}
			b, err := os.ReadFile(tmp.Name())
			if err != nil {
				return tui.EditedMsg{Err: err.Error()}
			}
			return tui.EditedMsg{YAML: string(b)}
		}
	}
	saveFn := func(y string) tea.Cmd {
		return func() tea.Msg {
			ok, err := save("docker-compose.yml", []byte(y), func() bool { return true })
			if err != nil {
				return tui.SavedMsg{Err: err.Error()}
			}
			return tui.SavedMsg{Path: "docker-compose.yml", OK: ok}
		}
	}

	m := tui.New(items).WithBuild(build).WithEdit(editFn).WithSave(saveFn)
```

(Replace the old single `m := tui.New(...)` line.)

Note: `$EDITOR` requires releasing the terminal; wrap the edit command with `tea.ExecProcess`-equivalent behavior. Simplest correct approach: change `editFn` to return `tea.Exec(exec.Command(...), func(err error) tea.Msg {...})`. Update `editFn`:

```go
	editFn := func(y string) tea.Cmd {
		tmp, err := os.CreateTemp("", "ctc-*.yml")
		if err != nil {
			return func() tea.Msg { return tui.EditedMsg{Err: err.Error()} }
		}
		tmp.WriteString(y)
		tmp.Close()
		ed := os.Getenv("EDITOR")
		if ed == "" {
			ed = "vi"
		}
		c := exec.Command(ed, tmp.Name())
		return tea.ExecProcess(c, func(err error) tea.Msg {
			defer os.Remove(tmp.Name())
			if err != nil {
				return tui.EditedMsg{Err: err.Error()}
			}
			b, rerr := os.ReadFile(tmp.Name())
			if rerr != nil {
				return tui.EditedMsg{Err: rerr.Error()}
			}
			return tui.EditedMsg{YAML: string(b)}
		})
	}
```

This makes `editInEditor` redundant; delete it from `main.go` and its call site. Keep `save`.

- [ ] **Step 7: Build + full test**

Run: `go build ./... && go test ./...`
Expected: build OK, all tests PASS.

- [ ] **Step 8: Manual smoke (documented, not CI)**

Run: `go run .` on a host with docker + at least one container. Verify: list renders, space checks, enter shows YAML, `e` opens editor, `s` writes `docker-compose.yml`.

- [ ] **Step 9: Commit**

```bash
git add internal/tui/model.go main.go
git commit -m "feat: wire edit ($EDITOR) and save into preview screen"
```

---

## Self-Review

**Spec coverage:**
- Multi-select list → Task 4 (`space`, `a`, cursor). ✓
- Full runtime capture (network, volumes, ports, env, command, runtime, GPU) → Task 3 builder. ✓
- Runtime + `--gpus`/`DeviceRequests` mapping → Task 3 `devices()` + golden. ✓
- Preview then edit in `$EDITOR` → Task 6. ✓
- `docker inspect` data source (CLI shell-out) → Tasks 1–2. ✓
- Error handling (daemon down, no containers, editor fallback, overwrite prompt, malformed YAML reload) → Task 5 (`save`, availability, empty check), Task 6 (`EditedMsg.Err` reload path). ✓
- Compose v3, no `version:` key → Task 3 model has no version field. ✓
- Testing split (docker fixtures, compose golden, tui update tests) → Tasks 1,3,4,6. ✓

**Placeholder scan:** No TBD/TODO; all code shown. ✓

**Type consistency:** `PreviewReadyMsg`, `SavedMsg`, `EditedMsg`, `WithBuild/WithEdit/WithSave`, `Selected()`, `SetPreview` used consistently across Tasks 4–6 and main. `save` signature identical in Tasks 5 and 6. `docker.Runner` used uniformly. ✓

Note: golden-file exact bytes depend on `yaml.v3` formatting; Task 3 Step 8 instructs reconciling the golden to actual output — the one place exact bytes can't be pre-known.
