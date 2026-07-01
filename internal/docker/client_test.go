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
