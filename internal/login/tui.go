package login

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"magic/internal/provider"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	checkedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	noteStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

const listPageSize = 15

// --- key input ---

type keyModel struct {
	ti      textinput.Model
	title   string
	keyEnv  string
	value   string
	done    bool
	aborted bool
}

func newKeyModel(title, keyEnv string) keyModel {
	ti := textinput.New()
	ti.Placeholder = "paste API key"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.Prompt = "> "
	return keyModel{ti: ti, title: title, keyEnv: keyEnv}
}

func (m keyModel) Init() tea.Cmd { return textinput.Blink }

func (m keyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			m.value = strings.TrimSpace(m.ti.Value())
			if m.value == "" {
				return m, nil
			}
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m keyModel) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render(m.title))
	if m.keyEnv != "" {
		fmt.Fprintf(&b, "%s\n\n", hintStyle.Render("stored as "+m.keyEnv+" in secrets/providers.json (plaintext, 0600)"))
	}
	b.WriteString(m.ti.View())
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("enter confirm · esc cancel"))
	b.WriteString("\n")
	return b.String()
}

// PromptKey runs the masked key input TUI. Empty result means aborted.
func PromptKey(title, keyEnv string) (string, error) {
	res, err := tea.NewProgram(newKeyModel(title, keyEnv)).Run()
	if err != nil {
		return "", err
	}
	m := res.(keyModel)
	if m.aborted {
		return "", errAborted
	}
	return m.value, nil
}

// --- list (select one or many) ---

type listModel struct {
	title    string
	source   string
	items    []provider.RemoteModel
	cursor   int
	selected map[int]bool
	multi    bool
	height   int
	done     bool
	aborted  bool
	chosen   string
}

func newListModel(title, source string, items []provider.RemoteModel, preselected []string, multi bool) listModel {
	m := listModel{title: title, source: source, items: items, selected: map[int]bool{}, multi: multi, height: 24}
	preset := map[string]bool{}
	for _, id := range preselected {
		preset[id] = true
	}
	for i, it := range items {
		if preset[it.ID] {
			m.selected[i] = true
		}
	}
	return m
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		if m.height <= 0 {
			m.height = 24
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "pgup":
			m.cursor -= m.pageSize()
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown":
			m.cursor += m.pageSize()
			if m.cursor > len(m.items)-1 {
				m.cursor = len(m.items) - 1
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len(m.items) - 1
		case " ":
			if m.multi {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "a":
			if m.multi {
				all := len(m.selected) != len(m.items)
				m.selected = map[int]bool{}
				if all {
					for i := range m.items {
						m.selected[i] = true
					}
				}
			}
		case "enter":
			if len(m.items) == 0 {
				return m, nil
			}
			if m.multi {
				if m.count() == 0 {
					return m, nil
				}
			} else {
				m.chosen = m.items[m.cursor].ID
			}
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m listModel) pageSize() int {
	p := m.height - 6
	if p < 3 {
		p = 3
	}
	if p > listPageSize {
		p = listPageSize
	}
	return p
}

func (m listModel) count() int { return len(m.selected) }

func (m listModel) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", titleStyle.Render(m.title))
	if m.source != "" {
		fmt.Fprintf(&b, " %s", hintStyle.Render("("+m.source+")"))
	}
	b.WriteString("\n")
	if m.multi {
		fmt.Fprintf(&b, "%s\n\n", hintStyle.Render(fmt.Sprintf("%d selected · space toggle · a all · enter confirm · esc cancel", m.count())))
	} else {
		fmt.Fprintf(&b, "%s\n\n", hintStyle.Render("↑/↓ move · enter confirm · esc cancel"))
	}

	if len(m.items) == 0 {
		b.WriteString(noteStyle.Render("(no candidates — type model ids manually)"))
		b.WriteString("\n")
		return b.String()
	}

	page := m.pageSize()
	start := 0
	if m.cursor >= page {
		start = m.cursor - page + 1
	}
	end := start + page
	if end > len(m.items) {
		end = len(m.items)
	}
	for i := start; i < end; i++ {
		it := m.items[i]
		marker := "  "
		if m.multi {
			if m.selected[i] {
				marker = checkedStyle.Render("◉ ")
			} else {
				marker = "○ "
			}
		}
		line := it.ID
		if it.Name != "" && it.Name != it.ID {
			line = it.ID + "  " + hintStyle.Render(it.Name)
		}
		if i == m.cursor {
			line = cursorStyle.Render("› ") + line
		} else {
			line = "  " + line
		}
		b.WriteString(marker + line + "\n")
	}
	if end < len(m.items) || start > 0 {
		fmt.Fprintf(&b, "\n%s", hintStyle.Render(fmt.Sprintf("showing %d–%d of %d", start+1, end, len(m.items))))
		b.WriteString("\n")
	}
	return b.String()
}

var errAborted = fmt.Errorf("cancelled")

// SelectModels runs the multi-select picker. Returns the selected ids.
func SelectModels(title, source string, models []provider.RemoteModel, preselected []string) ([]string, error) {
	res, err := tea.NewProgram(newListModel(title, source, models, preselected, true)).Run()
	if err != nil {
		return nil, err
	}
	m := res.(listModel)
	if m.aborted {
		return nil, errAborted
	}
	var out []string
	for i, it := range m.items {
		if m.selected[i] {
			out = append(out, it.ID)
		}
	}
	return out, nil
}

// ChooseProvider runs the single-select picker.
func ChooseProvider(title string, names []string) (string, error) {
	items := make([]provider.RemoteModel, 0, len(names))
	for _, n := range names {
		items = append(items, provider.RemoteModel{ID: n})
	}
	res, err := tea.NewProgram(newListModel(title, "", items, nil, false)).Run()
	if err != nil {
		return "", err
	}
	m := res.(listModel)
	if m.aborted {
		return "", errAborted
	}
	return m.chosen, nil
}

// --- confirm ---

type confirmModel struct {
	summary string
	value   bool
	done    bool
	aborted bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "y", "Y", "enter":
			m.value = true
			m.done = true
			return m, tea.Quit
		case "n", "N", "q", "esc", "ctrl+c":
			m.aborted = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	return titleStyle.Render("Write to the config library?") + "\n\n" + m.summary + "\n\n" + hintStyle.Render("y confirm · n cancel") + "\n"
}

// Confirm shows a summary and asks to proceed.
func Confirm(summary string) (bool, error) {
	res, err := tea.NewProgram(confirmModel{summary: summary}).Run()
	if err != nil {
		return false, err
	}
	m := res.(confirmModel)
	if m.aborted {
		return false, errAborted
	}
	return m.value, nil
}

// --- manual ids ---

// PromptIDs asks for comma/space separated model ids (manual fallback).
func PromptIDs(title string) ([]string, error) {
	res, err := tea.NewProgram(newKeyModel(title, "")).Run()
	if err != nil {
		return nil, err
	}
	m := res.(keyModel)
	if m.aborted {
		return nil, errAborted
	}
	return splitIDs(m.value), nil
}

func splitIDs(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	return cleanIDs(fields)
}

// IsAborted reports whether err is the user-cancel sentinel.
func IsAborted(err error) bool { return err == errAborted }
