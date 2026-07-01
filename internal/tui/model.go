package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/centerseat/ctc/internal/docker"
)

type Screen int

const (
	ScreenList Screen = iota
	ScreenPreview
)

type PreviewReadyMsg struct{ YAML string }

// BuildFunc is injected by main: given selected IDs, produce compose YAML.
type BuildFunc func(ids []string) tea.Cmd

type Model struct {
	items   []docker.ContainerSummary
	cursor  int
	checked map[int]bool
	screen  Screen
	yaml    string
	offset  int // preview scroll
	build   BuildFunc
	err     string
}

func New(items []docker.ContainerSummary) Model {
	return Model{items: items, checked: map[int]bool{}, screen: ScreenList}
}

// WithBuild attaches the compose-build command factory.
func (m Model) WithBuild(b BuildFunc) Model { m.build = b; return m }

func (m *Model) SetPreview(y string) { m.yaml = y; m.screen = ScreenPreview }

func (m Model) Selected() []string {
	var out []string
	for i, c := range m.items {
		if m.checked[i] {
			out = append(out, c.ID)
		}
	}
	return out
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PreviewReadyMsg:
		m.yaml = msg.YAML
		m.screen = ScreenPreview
		m.offset = 0
		return m, nil
	case tea.KeyMsg:
		if m.screen == ScreenList {
			return m.updateList(msg)
		}
		return m.updatePreview(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if len(m.Selected()) > 0 && m.build != nil {
			return m, m.build(m.Selected())
		}
	case tea.KeyRunes:
		switch msg.Runes[0] {
		case ' ':
			m.checked[m.cursor] = !m.checked[m.cursor]
		case 'a':
			all := len(m.Selected()) != len(m.items)
			for i := range m.items {
				m.checked[i] = all
			}
		case 'q':
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.offset > 0 {
			m.offset--
		}
	case tea.KeyDown:
		m.offset++
	case tea.KeyEsc:
		m.screen = ScreenList
	case tea.KeyRunes:
		if msg.Runes[0] == 'q' {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.screen == ScreenPreview {
		var b strings.Builder
		b.WriteString("Preview  [e]dit  [s]ave  [esc] back  [q]uit\n\n")
		if m.err != "" {
			b.WriteString("! " + m.err + "\n\n")
		}
		lines := strings.Split(m.yaml, "\n")
		for i := m.offset; i < len(lines) && i < m.offset+30; i++ {
			b.WriteString(lines[i] + "\n")
		}
		return b.String()
	}
	var b strings.Builder
	b.WriteString("Select containers  [space] toggle  [a] all  [enter] build  [q]uit\n\n")
	for i, c := range m.items {
		cur := " "
		if i == m.cursor {
			cur = ">"
		}
		box := "[ ]"
		if m.checked[i] {
			box = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s %s %-20s %-25s %s\n", cur, box, c.Names, c.Image, c.State))
	}
	return b.String()
}
