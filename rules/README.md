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

## Anki notes

`rb gen-rules-anki` (`just gen-rules-anki`) walks the rulings one at a time and
drafts a short question and a short answer from each: the ruling's own
question, and the sentences that open its answer. Drafting is mechanical, since
a ruling explains itself over several paragraphs and a note has to say the same
thing in a sentence — so each draft is shown under the whole ruling and kept,
reworded or skipped by hand.

Every decision is written through to `anki-review.json` as it is made, so a
pass can stop at any point (`q`, or ctrl+d) and pick up where it left off, and
a note reworded once stays reworded. `-revisit` offers the settled rulings
again, each prefilled with the note it was given. The approved notes are
written to `../anki/riftbound-rulings.txt` at the end of every pass, tagged by
category and topic, with the Core Rules citation added to the back.

Decisions are filed under a ruling's slug and anchor, so rewording a ruling
keeps its note; moving it under another anchor makes it a new one to review.

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
Keep the credit line in `index.html` if you rework the tab. `anki-review.json`
and the deck built from it are derived from that content as well, so
share-alike follows them into this repo.

## Unfinished

Collection stopped early — the scraper hit a 30-request free-trial cap, with no
fallback (`reddit.com/*.json` returns 403, WebFetch is domain-blocked).
`reddit-threads-queue.txt` lists threads already identified as rules discussions
but never read, including all of r/riftboundtcg — 178K members and denser in
rules content than r/Riftbound, never enumerated at all. Resuming needs only a
working scraper.

Anki notes are drafted a ruling at a time by `rb gen-rules-anki`; nothing has
been reviewed yet, so `anki-review.json` doesn't exist until the first pass.
