package collection

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type shoppingModel struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func NewModel(_ ...any) shoppingModel {
	return shoppingModel{
		choices:  []string{"card 1", "card 2", "card 3"},
		selected: map[int]struct{}{},
	}
}

func (m shoppingModel) Init() tea.Cmd {
	return nil
}

func (m shoppingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", "space":
			if _, alreadySelected := m.selected[m.cursor]; alreadySelected {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m shoppingModel) View() tea.View {
	s := "What should we buy?\n\n"

	for i := range m.choices {
		cursorCol := " "
		if m.cursor == i {
			cursorCol = ">"
		}

		selectedCol := " "
		if _, selected := m.selected[i]; selected {
			selectedCol = "x"
		}

		s += fmt.Sprintf("%s %s %s\n", cursorCol, selectedCol, m.choices[i])
	}

	return tea.NewView(s)
}
