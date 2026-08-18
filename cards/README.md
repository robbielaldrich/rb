# riftcodex_cards

Card data and images downloaded from the [Riftcodex API](https://riftcodex.com/docs/endpoints/cards/).

The JSON files and `images/` in this directory are gitignored and not committed. To (re)populate it, run:

```
just riftcodex-dl
```

This writes `sets.json`, a combined `cards.json`, and one image per card under `images/<riftbound_id>.png`.
Re-running only downloads images that aren't already on disk, so it's safe to run repeatedly.
