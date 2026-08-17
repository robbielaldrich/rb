# riftcodex_cards

Card data downloaded from the [Riftcodex API](https://riftcodex.com/docs/endpoints/cards/).

The JSON files in this directory are gitignored and not committed. To (re)populate it, run:

```
just riftcodex-dl
```

This writes `sets.json` and a combined `cards.json`.
