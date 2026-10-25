# cards

Card data and images downloaded from [riftbound.gg](https://riftbound.gg/), through the DotGG API that feeds its pages
(`https://api.dotgg.gg/cgfw/getcards?game=riftbound` and `getsets`). The API is undocumented and may change without notice.

To (re)populate this directory, run:

```
just download-cards
```

This writes `sets.json`, a combined `cards.json`, and one image per card under `images/<riftbound_id>.webp`.
`images/` is gitignored. Re-running only downloads images that aren't already on disk, so it's safe to run repeatedly.
Older scans downloaded from Riftcodex, the source before riftbound.gg, are `.png` and are kept where present.

Cards keep the ids Riftcodex gave them (`ven-044-166`, `ven-044a-166`, `unl-226*-219`, `ven-r01`, `ven-sp3-006`), so the
collection and decks recorded against it still line up. Promos are filed under the set whose card they reprint, with a `p`
in the number (`unl-009p-219`, `rad-015p2-166`). Tokens, and the bundle and event printings riftbound.gg files under sets
of their own (ARC, SGN, T1A, ...), are left out.
