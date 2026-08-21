package collection

import (
	"fmt"

	"rb/cards"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type editor struct {
	catalog    []cards.Card
	collection *collection

	ti textinput.Model
}

func newEditor(collection *collection, cards []cards.Card) editor {
	ti := textinput.New()

	suggestions := make([]string, len(cards))
	for i, card := range cards {
		suggestions[i] = fmt.Sprintf("%s %d %s", card.Name, card.CollectorNumber, card.Set)
	}
	ti.SetSuggestions(suggestions)
	ti.ShowSuggestions = true
	ti.Prompt = "Choose a card."
	ti.Focus()

	return editor{
		catalog:    cards,
		collection: collection,
		ti:         ti,
	}
}

func (e editor) Init() tea.Cmd { return nil }

func (e editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg:
		if msg == "q" {
			return e, tea.Quit
		}
	}

	var cmd tea.Cmd
	e.ti, cmd = e.ti.Update(msg)

	return e, cmd
}

func (e editor) View() tea.View {
	return tea.NewView(e.ti.View())
}
