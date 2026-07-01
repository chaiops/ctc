package compose

import (
	"os"
	"strings"
	"testing"

	"github.com/chaiops/ctc/internal/docker"
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
			{Type: "bind", Source: "/srv/html", Destination: "/usr/share/nginx/html", RW: true},
			{Type: "volume", Name: "datavol", Destination: "/data", RW: true},
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

func TestNetworkModeHostOmitsNetworks(t *testing.T) {
	c := sampleContainer()
	c.HostConfig.NetworkMode = "host"
	f := Build([]docker.Container{c}, nil, nil)
	svc := f.Services["web"]
	if svc.NetworkMode != "host" {
		t.Fatalf("expected NetworkMode host, got %q", svc.NetworkMode)
	}
	if len(svc.Networks) != 0 {
		t.Fatalf("expected no networks, got %+v", svc.Networks)
	}
	y, err := f.YAML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(y), "networks:") && strings.Contains(string(y), "        networks:") {
		t.Fatalf("expected no networks: key in service, got:\n%s", y)
	}
}

func TestNetworkModeDefaultOmitted(t *testing.T) {
	c := sampleContainer()
	c.HostConfig.NetworkMode = "default"
	f := Build([]docker.Container{c}, nil, nil)
	svc := f.Services["web"]
	if svc.NetworkMode != "" {
		t.Fatalf("expected empty NetworkMode, got %q", svc.NetworkMode)
	}
}

func TestReadOnlyMount(t *testing.T) {
	c := sampleContainer()
	c.Mounts = []docker.Mount{
		{Type: "bind", Source: "/ro/src", Destination: "/ro/dst", RW: false},
		{Type: "bind", Source: "/rw/src", Destination: "/rw/dst", RW: true},
	}
	f := Build([]docker.Container{c}, nil, nil)
	svc := f.Services["web"]
	foundRO, foundRW := false, false
	for _, v := range svc.Volumes {
		if v == "/ro/src:/ro/dst:ro" {
			foundRO = true
		}
		if v == "/rw/src:/rw/dst" {
			foundRW = true
		}
	}
	if !foundRO {
		t.Fatalf("expected ro mount suffix, got %+v", svc.Volumes)
	}
	if !foundRW {
		t.Fatalf("expected rw mount without suffix, got %+v", svc.Volumes)
	}
}

func TestTmpfsMount(t *testing.T) {
	c := sampleContainer()
	c.Mounts = []docker.Mount{
		{Type: "tmpfs", Destination: "/dest"},
	}
	f := Build([]docker.Container{c}, nil, nil)
	svc := f.Services["web"]
	if len(svc.Tmpfs) != 1 || svc.Tmpfs[0] != "/dest" {
		t.Fatalf("expected Tmpfs [/dest], got %+v", svc.Tmpfs)
	}
	for _, v := range svc.Volumes {
		if strings.Contains(v, ":/dest") {
			t.Fatalf("tmpfs mount should not appear in volumes, got %+v", svc.Volumes)
		}
	}
}

func TestEphemeralPortNoLeadingColon(t *testing.T) {
	pb := map[string][]docker.PortBinding{
		"80/tcp": {{HostIP: "", HostPort: ""}},
	}
	got := ports(pb)
	if len(got) != 1 || got[0] != "80/tcp" {
		t.Fatalf("expected [80/tcp], got %+v", got)
	}
}

func TestFilterUserEnv(t *testing.T) {
	image := []string{"PATH=/usr/bin", "NGINX_VERSION=1.27", "FOO=default"}
	container := []string{
		"PATH=/usr/bin",       // identical image default -> drop
		"NGINX_VERSION=1.27",  // identical image default -> drop
		"FOO=override",        // user changed value -> keep
		"APP_KEY=secret",      // user-added, not in image -> keep
	}
	got := FilterUserEnv(container, image)
	want := []string{"FOO=override", "APP_KEY=secret"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestFilterUserEnvNilImage(t *testing.T) {
	// No image env known -> keep everything (can't distinguish).
	container := []string{"A=1", "B=2"}
	got := FilterUserEnv(container, nil)
	if len(got) != 2 {
		t.Fatalf("expected passthrough, got %v", got)
	}
}
