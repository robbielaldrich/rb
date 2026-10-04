# Anki decks changelog

What changed in the generated decks under `anki/`, newest first. Each entry
names the decks and categories it touches, then what changed in them.

Releases are tagged `anki-vMAJOR.MINOR.PATCH` on the commit that ships them:

- **MAJOR** — a re-import leaves notes behind to clean up: a question is
  reworded or dropped (see `to-delete.txt`), or a deck is renamed or removed.
- **MINOR** — something is added: a new deck, or new questions in one.
- **PATCH** — only answers change, so a re-import updates notes in place and
  keeps their scheduling.

## 1.0.0 — 2026-10-04

**Hidden by Domain** — Calm, Mind and Order

- Signature cards are left off every domain's roster: a legend unlocks them,
  not a domain's runes, and the Signature Cards deck already asks after them.
  Fox-Fire drops out of Calm and Mind, and Hostile Takeover out of Mind and
  Order.
- Questions are unchanged, so re-importing updates the three notes in place.
