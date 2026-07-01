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

// Bubble Tea delivers the spacebar as KeySpace, not KeyRunes. This mirrors
// what a real terminal sends.
func TestSpaceKeyTogglesSelection(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = m2.(Model)
	sel := m.Selected()
	if len(sel) != 1 || sel[0] != "a" {
		t.Fatalf("KeySpace should toggle row 0, got selected: %v", sel)
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

func TestPreviewSaveInvokesHook(t *testing.T) {
	called := false
	m := newModel().WithSave(func(y string) tea.Cmd {
		return func() tea.Msg { called = true; return SavedMsg{Path: "x", OK: true} }
	})
	m.SetPreview("services: {}")
	_, cmd := m.Update(key('s'))
	if cmd == nil {
		t.Fatal("expected save cmd")
	}
	cmd() // run it
	if !called {
		t.Fatal("save hook not called")
	}
}

func TestEditedMsgUpdatesYAML(t *testing.T) {
	m := newModel()
	m.SetPreview("old")
	m2, _ := m.Update(EditedMsg{YAML: "new"})
	m = m2.(Model)
	if m.yaml != "new" {
		t.Fatalf("yaml=%q", m.yaml)
	}
}
