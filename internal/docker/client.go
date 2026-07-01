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

// Inspect inspects each id individually so that a single bad id does not
// abort the whole batch. It returns the successfully inspected containers
// and a separate list of ids that failed to inspect. A non-nil error is
// returned only for unexpected/total failures, not per-id misses.
func Inspect(run Runner, ids []string) ([]Container, []string, error) {
	var cs []Container
	var failed []string
	for _, id := range ids {
		out, err := run("docker", "inspect", id)
		if err != nil {
			failed = append(failed, id)
			continue
		}
		one, perr := ParseInspect(out)
		if perr != nil {
			failed = append(failed, id)
			continue
		}
		cs = append(cs, one...)
	}
	return cs, failed, nil
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

// imageInspect is the subset of `docker image inspect` we need.
type imageInspect struct {
	Config struct {
		Env []string
	}
}

// InspectImage returns the default environment baked into an image (its
// Dockerfile ENV entries, fully resolved through base images).
func InspectImage(run Runner, ref string) ([]string, error) {
	out, err := run("docker", "image", "inspect", ref)
	if err != nil {
		return nil, err
	}
	var imgs []imageInspect
	if err := json.Unmarshal(out, &imgs); err != nil {
		return nil, fmt.Errorf("parse image inspect: %w", err)
	}
	if len(imgs) == 0 {
		return nil, fmt.Errorf("image %q not found", ref)
	}
	return imgs[0].Config.Env, nil
}
