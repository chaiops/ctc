package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/centerseat/ctc/internal/docker"
)

func newModel() Model {
	return New([]docker.ContainerSummary{
		{ID: "a", Names: "web", Image: "nginx"},
		{ID: "b", Names: "db", Image: "postgres"},
	})
}

func key(r rune) tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestToggleAndSelect(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(key(' ')) // check row 0
	m = m2.(Model)
	sel := m.Selected()
	if len(sel) != 1 || sel[0] != "a" {
		t.Fatalf("selected: %v", sel)
	}
}

func TestCursorDown(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m2.(Model)
	if m.cursor != 1 {
		t.Fatalf("cursor: %d", m.cursor)
	}
}

func TestPreviewReadySwitchesScreen(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(PreviewReadyMsg{YAML: "services: {}"})
	m = m2.(Model)
	if m.screen != ScreenPreview || m.yaml != "services: {}" {
		t.Fatalf("screen=%d yaml=%q", m.screen, m.yaml)
	}
}
