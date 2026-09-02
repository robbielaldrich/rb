package ankigen

import (
	"testing"

	"rb/rules"
)

func TestCondense(t *testing.T) {
	for _, tc := range []struct {
		name  string
		in    rules.Ruling
		front string
		back  string
	}{
		{
			name:  "a question that names its topic is left alone",
			in:    rules.Ruling{Category: "cards", Topic: "Alpha Strike", Question: "How are targets chosen when playing Alpha Strike?", Answer: "Its controller chooses them while finalizing the spell."},
			front: "How are targets chosen when playing Alpha Strike?",
			back:  "Its controller chooses them while finalizing the spell.",
		},
		{
			name:  "one that doesn't is told what it is about",
			in:    rules.Ruling{Category: "general-rules", Topic: "Targeting", Question: "What happens if a target becomes illegal?", Answer: "The spell still resolves, but the illegal target is unaffected."},
			front: "Targeting — What happens if a target becomes illegal?",
			back:  "The spell still resolves, but the illegal target is unaffected.",
		},
		{
			name:  "a card is recognised by its short name",
			in:    rules.Ruling{Category: "cards", Topic: "Akshan, Mischievous", Question: "Who controls the gear when either Akshan dies?", Answer: "The surviving Akshan does."},
			front: "Who controls the gear when either Akshan dies?",
			back:  "The surviving Akshan does.",
		},
		{
			name:  "a thread note keeps the question it was written with",
			in:    rules.Ruling{Category: reddit, Topic: "r/Riftbound thread", Question: "Does damage lower a unit's Might?", Answer: "No. Damage never reduces Might — it is marked on the unit."},
			front: "Does damage lower a unit's Might?",
			back:  "No. Damage never reduces Might — it is marked on the unit.",
		},
		{
			name:  "a bare yes or no takes the sentence that explains it",
			in:    rules.Ruling{Topic: "Deathknell", Question: "Deathknell: can it target a unit that died with it?", Answer: "Yes.\nEvery unit killed by the same event is already in the trash.\nA token is not.\n\nDuring cleanup, nothing else has moved."},
			front: "Deathknell: can it target a unit that died with it?",
			back:  "Yes. Every unit killed by the same event is already in the trash.",
		},
		{
			name:  "the reasoning after the first paragraph is left behind",
			in:    rules.Ruling{Topic: "Movement", Question: "Movement: is a bounce a move?", Answer: "No, a bounce is not a move at all.\n\nA move is a permanent changing its position on the board."},
			front: "Movement: is a bounce a move?",
			back:  "No, a bounce is not a move at all.",
		},
		{
			name:  "markdown is flattened, since a note renders as HTML",
			in:    rules.Ruling{Topic: "Flow", Question: "Flow: where does a countered spell go?", Answer: "It goes to the **trash**, per the [Vendetta FAQ](https://example.com/faq)."},
			front: "Flow: where does a countered spell go?",
			back:  "It goes to the trash, per the Vendetta FAQ.",
		},
		{
			name:  "a rule number doesn't end a sentence",
			in:    rules.Ruling{Topic: "Costs and Payments", Question: "Costs and Payments: are costs refunded?", Answer: "No, see 466.1.a. Countering refunds nothing."},
			front: "Costs and Payments: are costs refunded?",
			back:  "No, see 466.1.a. Countering refunds nothing.",
		},
	} {
		front, back := condense(tc.in)
		if front != tc.front {
			t.Errorf("%s:\n front %q\n want  %q", tc.name, front, tc.front)
		}
		if back != tc.back {
			t.Errorf("%s:\n back %q\n want %q", tc.name, back, tc.back)
		}
	}
}

// Every ruling has to draft into something, or the review would offer a blank
// note to approve.
func TestCondenseDraftsEveryRuling(t *testing.T) {
	rs, err := rules.Load("../rules/rulings.json")
	if err != nil {
		t.Fatalf("failed to load the rulings: %v", err)
	}
	for _, r := range rs {
		switch front, back := condense(r); {
		case front == "":
			t.Errorf("%s: no question drafted", r.Key())
		case back == "":
			t.Errorf("%s: no answer drafted", r.Key())
		}
	}
}
