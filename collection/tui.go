package collection

import (
	"fmt"

	"rb/catalog"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type editor struct {
	catalog    []catalog.Card
	collection *collection

	ti textinput.Model
}

func cardToSuggestion(catalog.Card) string {
	return fmt.Sprintf("%s %d %s", card.Name, card.CollectorNumber, card.Set)
}

func suggestionToCard(suggestion string) (name, set string, number int) {
	fmt.Sscanf(suggestion, "%s %d %s", &name, &number, &set)
}

func newEditor(collection *collection, cards []catalog.Card) editor {
	ti := textinput.New()

	suggestions := make([]string, len(cards))
	for i, card := range cards {
		suggestions[i] = fmt.Sprintf("%s %d %s", card.Name, card.CollectorNumber, card.Set.Label)
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
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return e, tea.Quit
		case "enter":
			// Select this card to add to collection.
			suggestions := e.ti.MatchedSuggestions()
			if len(suggestions) == 1 {
				// Match the card from the suggestion.
				e.collection.set(
			}

		}
	}

	var cmd tea.Cmd
	e.ti, cmd = e.ti.Update(msg)

	return e, cmd
}

func (e editor) View() tea.View {
	return tea.NewView(e.ti.View())
}
