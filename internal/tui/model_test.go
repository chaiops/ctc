package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chaiops/ctc/internal/docker"
)

// newModel returns a model already past the startup screen with two
// containers loaded, as if ContainersLoadedMsg had arrived.
func newModel() Model {
	m := New()
	m.items = []docker.ContainerSummary{
		{ID: "a", Names: "web", Image: "nginx"},
		{ID: "b", Names: "db", Image: "postgres"},
	}
	m.screen = ScreenList
	return m
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

func TestStartsOnStartupScreen(t *testing.T) {
	m := New()
	if m.screen != ScreenStartup {
		t.Fatalf("expected startup screen, got %d", m.screen)
	}
}

func TestContainersLoadedShowsList(t *testing.T) {
	m := New()
	m2, _ := m.Update(ContainersLoadedMsg{Items: []docker.ContainerSummary{
		{ID: "a", Names: "web", Image: "nginx"},
	}})
	m = m2.(Model)
	if m.screen != ScreenList {
		t.Fatalf("expected list screen, got %d", m.screen)
	}
	if len(m.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.items))
	}
}

func TestContainersLoadErrorShowsError(t *testing.T) {
	m := New()
	m2, _ := m.Update(ContainersLoadedMsg{Err: "docker daemon not running"})
	m = m2.(Model)
	if m.screen != ScreenList || m.loadErr == "" {
		t.Fatalf("expected list screen with loadErr, got screen=%d err=%q", m.screen, m.loadErr)
	}
}

func TestEnterShowsLoadingScreen(t *testing.T) {
	m := newModel().WithBuild(func(ids []string) tea.Cmd {
		return func() tea.Msg { return PreviewReadyMsg{YAML: "services: {}"} }
	})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace}) // select row 0
	m = m2.(Model)
	m3, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m3.(Model)
	if m.screen != ScreenLoading {
		t.Fatalf("expected loading screen, got %d", m.screen)
	}
	if cmd == nil {
		t.Fatal("expected build+spinner batch command")
	}
}

func TestPreviewReadyLeavesLoading(t *testing.T) {
	m := newModel()
	m.screen = ScreenLoading
	m2, _ := m.Update(PreviewReadyMsg{YAML: "services: {}"})
	m = m2.(Model)
	if m.screen != ScreenPreview {
		t.Fatalf("expected preview after build, got %d", m.screen)
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
