package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// SaveFunc is injected by main: given the current YAML, persist it.
type SaveFunc func(yaml string) tea.Cmd

// EditFunc is injected by main: given the current YAML, open it in an editor.
type EditFunc func(yaml string) tea.Cmd

type SavedMsg struct {
	Path string
	OK   bool
	Err  string
}

type EditedMsg struct {
	YAML string
	Err  string
}

type Model struct {
	items   []docker.ContainerSummary
	cursor  int
	checked map[int]bool
	screen  Screen
	yaml    string
	offset  int // preview scroll
	build   BuildFunc
	save    SaveFunc
	edit    EditFunc
	status  string
	err     string
	width   int
	height  int
}

func New(items []docker.ContainerSummary) Model {
	return Model{items: items, checked: map[int]bool{}, screen: ScreenList}
}

// WithBuild attaches the compose-build command factory.
func (m Model) WithBuild(b BuildFunc) Model { m.build = b; return m }

// WithSave attaches the save command factory.
func (m Model) WithSave(f SaveFunc) Model { m.save = f; return m }

// WithEdit attaches the edit command factory.
func (m Model) WithEdit(f EditFunc) Model { m.edit = f; return m }

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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case PreviewReadyMsg:
		m.yaml = msg.YAML
		m.screen = ScreenPreview
		m.offset = 0
		return m, nil
	case SavedMsg:
		if msg.OK {
			m.status = "saved: " + msg.Path
		} else if msg.Err != "" {
			m.status = "save error: " + msg.Err
		} else {
			m.status = "save cancelled"
		}
		return m, nil
	case EditedMsg:
		if msg.Err != "" {
			m.status = "edit error: " + msg.Err
		} else {
			m.yaml = msg.YAML
			m.offset = 0
		}
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
	case tea.KeySpace:
		m.checked[m.cursor] = !m.checked[m.cursor]
	case tea.KeyEnter:
		if len(m.Selected()) > 0 && m.build != nil {
			return m, m.build(m.Selected())
		}
	case tea.KeyRunes:
		switch msg.Runes[0] {
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
		switch msg.Runes[0] {
		case 'q':
			return m, tea.Quit
		case 's':
			if m.save != nil {
				return m, m.save(m.yaml)
			}
		case 'e':
			if m.edit != nil {
				return m, m.edit(m.yaml)
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.screen == ScreenPreview {
		return m.previewView()
	}
	return m.listView()
}

func (m Model) listView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  ctc · container → compose  "))
	b.WriteString("\n")
	n := len(m.Selected())
	b.WriteString(subtitleStyle.Render(fmt.Sprintf(
		"%d container(s) · %d selected", len(m.items), n)))
	b.WriteString("\n\n")

	for i, c := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("▸ ")
		}

		box := "○"
		if m.checked[i] {
			box = checkedStyle.Render("●")
		} else {
			box = metaStyle.Render("○")
		}

		name := c.Names
		if i == m.cursor {
			name = rowSelStyle.Render(name)
		} else {
			name = rowStyle.Render(name)
		}

		badge := stoppedBadge.Render("○ " + c.State)
		if isRunning(c.State) {
			badge = runningBadge.Render("● " + c.State)
		}

		line := fmt.Sprintf("%s%s  %-24s  %-28s  %s",
			cursor, box,
			truncate(name, 24),
			imageStyle.Render(truncate(c.Image, 28)),
			badge,
		)
		if i == m.cursor {
			line = selectedRowBg.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(help(
		"space", "toggle", "a", "all",
		"↑↓", "move", "enter", "build", "q", "quit",
	))
	return b.String()
}

func (m Model) previewView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  docker-compose.yml preview  "))
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString(statusWarnStyle.Render("⚠ "+m.err) + "\n")
	}
	if m.status != "" {
		style := statusOKStyle
		if strings.HasPrefix(m.status, "save error") || strings.HasPrefix(m.status, "edit error") {
			style = statusErrStyle
		} else if m.status == "save cancelled" {
			style = statusWarnStyle
		}
		b.WriteString(style.Render(m.status) + "\n")
	}
	b.WriteString("\n")

	lines := strings.Split(m.yaml, "\n")
	visible := 24
	if m.height > 12 {
		visible = m.height - 10
	}
	var body strings.Builder
	for i := m.offset; i < len(lines) && i < m.offset+visible; i++ {
		body.WriteString(highlightYAML(lines[i]) + "\n")
	}

	frame := previewFrame
	if m.width > 4 {
		frame = frame.Width(m.width - 4)
	}
	b.WriteString(frame.Render(strings.TrimRight(body.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(help(
		"e", "edit", "s", "save", "↑↓", "scroll", "esc", "back", "q", "quit",
	))
	return b.String()
}

func isRunning(state string) bool {
	return strings.EqualFold(state, "running")
}

func truncate(s string, max int) string {
	// max counts visible runes; s may already carry ANSI styling, so measure
	// with lipgloss.Width and only cut plain strings.
	if lipgloss.Width(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// help renders alternating key/label pairs into a footer line.
func help(pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteString(helpStyle.Render("  ·  "))
		}
		b.WriteString(helpKeyStyle.Render(pairs[i]))
		b.WriteString(helpStyle.Render(" " + pairs[i+1]))
	}
	return b.String()
}

// highlightYAML applies light syntax coloring to one YAML line.
func highlightYAML(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}
	if strings.HasPrefix(trimmed, "#") {
		return yamlCommentStyle.Render(line)
	}

	indent := line[:len(line)-len(strings.TrimLeft(line, " "))]

	// List item: "- value"
	if strings.HasPrefix(trimmed, "- ") {
		return indent + yamlListStyle.Render("- ") + yamlValueStyle.Render(trimmed[2:])
	}

	// "key: value" or "key:"
	if idx := strings.Index(trimmed, ":"); idx >= 0 {
		key := trimmed[:idx]
		rest := trimmed[idx+1:]
		out := indent + yamlKeyStyle.Render(key) + yamlValueStyle.Render(":")
		if rest != "" {
			out += yamlValueStyle.Render(rest)
		}
		return out
	}
	return yamlValueStyle.Render(line)
}
