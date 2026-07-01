package docker

import (
	"encoding/json"
	"fmt"
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

func TestInspectSkipsFailedIDs(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		if name != "docker" || args[0] != "inspect" {
			t.Fatalf("unexpected call: %s %v", name, args)
		}
		id := args[1]
		if id == "badid" {
			return nil, fmt.Errorf("no such object: %s", id)
		}
		b, _ := json.Marshal([]Container{{ID: id, Name: "/" + id}})
		return b, nil
	}
	cs, failed, err := Inspect(run, []string{"goodid", "badid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs) != 1 || cs[0].ID != "goodid" {
		t.Fatalf("got containers %+v", cs)
	}
	if len(failed) != 1 || failed[0] != "badid" {
		t.Fatalf("got failed %+v", failed)
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
