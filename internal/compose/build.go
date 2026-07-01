package compose

import (
	"sort"
	"strconv"
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
			Tmpfs:       tmpfsMounts(c.Mounts),
		}
		if nm := networkMode(c.HostConfig.NetworkMode); nm != "" {
			s.NetworkMode = nm
			s.Networks = nil
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
			switch {
			case b.HostPort == "":
				out = append(out, k)
			case b.HostIP != "":
				out = append(out, b.HostIP+":"+b.HostPort+":"+k)
			default:
				out = append(out, b.HostPort+":"+k)
			}
		}
	}
	return out
}

func mounts(ms []docker.Mount) []string {
	var out []string
	for _, m := range ms {
		if m.Type == "tmpfs" {
			continue
		}
		src := m.Source
		if m.Type == "volume" {
			src = m.Name
		}
		v := src + ":" + m.Destination
		if !m.RW {
			v += ":ro"
		}
		out = append(out, v)
	}
	return out
}

func tmpfsMounts(ms []docker.Mount) []string {
	var out []string
	for _, m := range ms {
		if m.Type == "tmpfs" {
			out = append(out, m.Destination)
		}
	}
	return out
}

// networkMode maps a docker HostConfig.NetworkMode to the compose
// network_mode value, returning "" when the default mode should be
// represented via the networks: list instead.
func networkMode(nm string) string {
	if nm == "" || nm == "default" || nm == "bridge" {
		return ""
	}
	return nm
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

func sortStrings(s []string) { sort.Strings(s) }
func itoa(n int) string      { return strconv.Itoa(n) }
