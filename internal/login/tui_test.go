package login

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"magic/internal/provider"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestListModelMultiSelect(t *testing.T) {
	items := []provider.RemoteModel{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	m := newListModel("t", "live", items, []string{"b"}, true)

	// Preselection applied.
	if !m.selected[1] {
		t.Fatal("preselection b missing")
	}
	// Toggle a on, move down, toggle b off.
	next, _ := m.Update(keyMsg(" "))
	m = next.(listModel)
	if !m.selected[0] {
		t.Fatal("space did not select a")
	}
	next, _ = m.Update(keyMsg("down"))
	m = next.(listModel)
	next, _ = m.Update(keyMsg(" "))
	m = next.(listModel)
	if m.selected[1] {
		t.Fatal("space did not deselect b")
	}
	// Enter confirms.
	next, _ = m.Update(keyMsg("enter"))
	m = next.(listModel)
	if !m.done || m.aborted {
		t.Fatalf("done=%v aborted=%v", m.done, m.aborted)
	}
	if v := m.View(); !strings.Contains(v, "a") || !strings.Contains(v, "c") {
		t.Fatalf("view missing ids:\n%s", v)
	}
}

func TestListModelSelectAllAndAbort(t *testing.T) {
	items := []provider.RemoteModel{{ID: "a"}, {ID: "b"}}
	m := newListModel("t", "", items, nil, true)
	next, _ := m.Update(keyMsg("a"))
	m = next.(listModel)
	if m.count() != 2 {
		t.Fatalf("select all count=%d", m.count())
	}
	next, _ = m.Update(keyMsg("a"))
	m = next.(listModel)
	if m.count() != 0 {
		t.Fatalf("toggle-all-off count=%d", m.count())
	}
	next, _ = m.Update(keyMsg("esc"))
	m = next.(listModel)
	if !m.aborted {
		t.Fatal("esc should abort")
	}
}

func TestListModelSingleChoose(t *testing.T) {
	items := []provider.RemoteModel{{ID: "a"}, {ID: "b"}}
	m := newListModel("t", "", items, nil, false)
	next, _ := m.Update(keyMsg("down"))
	m = next.(listModel)
	next, _ = m.Update(keyMsg("enter"))
	m = next.(listModel)
	if m.chosen != "b" {
		t.Fatalf("chosen=%q", m.chosen)
	}
}

func TestListModelEnterWithoutSelectionIsNoop(t *testing.T) {
	items := []provider.RemoteModel{{ID: "a"}}
	m := newListModel("t", "", items, nil, true)
	next, _ := m.Update(keyMsg("enter"))
	m = next.(listModel)
	if m.done {
		t.Fatal("enter with no selection must not confirm")
	}
}

func TestConfirmModel(t *testing.T) {
	m := confirmModel{}
	next, _ := m.Update(keyMsg("y"))
	m = next.(confirmModel)
	if !m.done || !m.value {
		t.Fatalf("done=%v value=%v", m.done, m.value)
	}
	m2 := confirmModel{}
	next, _ = m2.Update(keyMsg("n"))
	m2 = next.(confirmModel)
	if !m2.aborted {
		t.Fatal("n should abort")
	}
}

func TestSplitIDs(t *testing.T) {
	got := splitIDs("a, b\n c,,b")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("%v", got)
	}
}
