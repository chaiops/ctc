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
	NetworkMode string            `yaml:"network_mode,omitempty"`
	Tmpfs       []string          `yaml:"tmpfs,omitempty"`
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
