package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/centerseat/ctc/internal/compose"
	"github.com/centerseat/ctc/internal/docker"
	"github.com/centerseat/ctc/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
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
			cs, failed, err := docker.Inspect(run, ids)
			if err != nil {
				return tui.PreviewReadyMsg{YAML: "# error: " + err.Error()}
			}
			nets := relatedNetworks(run, cs)
			vols := relatedVolumes(run, cs)
			y, err := compose.Build(cs, nets, vols).YAML()
			if err != nil {
				return tui.PreviewReadyMsg{YAML: "# error: " + err.Error()}
			}
			out := string(y)
			if len(failed) > 0 {
				out = "# warning: could not inspect: " + strings.Join(failed, ",") + "\n" + out
			}
			return tui.PreviewReadyMsg{YAML: out}
		}
	}

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
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
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
