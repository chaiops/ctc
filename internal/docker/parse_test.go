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
