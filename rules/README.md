# Rules

174 tricky Riftbound interactions — damage vs. Might, focus and the chain,
showdown cleanup, Deathknell ordering, targeting legality — each traced to a
numbered Core Rules citation. Served as the **Rules** tab of `../index.html`.

`rulings.json` is the source of truth. The tab fetches it lazily, the first time
the tab is opened, and groups it by topic; entries must stay sorted so that all
rulings sharing a topic are contiguous, since the renderer starts a new heading
whenever the topic changes rather than regrouping.

Each entry:

```json
{
  "category": "cards | mechanics | general-rules | reddit-field-notes",
  "topic":    "Elder Dragon",
  "slug":     "elder-dragon",
  "question": "Does damage from earlier in the turn become lethal…",
  "answer":   "Plain text. Blank lines split paragraphs, \"- \" lines make a bullet list, **bold** marks the phrase carrying the ruling.",
  "rules":    ["466.1.a.1", "402.2"],
  "source":   "https://…"
}
```

## Where it came from

**165 — community Riftbound FAQ** (<https://www.riftboundfaq.com/>, source at
`ChristianIvicevic/riftboundfaq`), reviewed against Core Rules 1.4. Parsed from
the upstream MDX: the JSX rule and card components were flattened to text and
the rule numbers lifted into the `rules` array. The upstream repo is not
vendored here — only this derived dataset.

**9 — r/Riftbound threads**, scraped with comment trees and written up by hand
in `reddit-field-notes.md`. They stay in their own category and are badged
"community thread" in the UI, because forum consensus is not a ruling. One is
flagged *contested*: two threads disagree on whether focus auto-passes when a
chain resolves.

## Licensing

The FAQ content is **CC BY-SA 4.0**, and this repo and its Pages site are
public, so that redistribution is live: attribution and the licence are shown in
the Rules tab itself, and share-alike binds anyone who takes the dataset onward.
Keep the credit line in `index.html` if you rework the tab.

## Unfinished

Collection stopped early — the scraper hit a 30-request free-trial cap, with no
fallback (`reddit.com/*.json` returns 403, WebFetch is domain-blocked).
`reddit-threads-queue.txt` lists threads already identified as rules discussions
but never read, including all of r/riftboundtcg — 178K members and denser in
rules content than r/Riftbound, never enumerated at all. Resuming needs only a
working scraper.

Next per `../PLANNING.md`: cluster these into Anki notes via `rb gen-anki`.
