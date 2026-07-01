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
	NetworkMode    string `json:"NetworkMode"`
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
