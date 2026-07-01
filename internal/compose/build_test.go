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
