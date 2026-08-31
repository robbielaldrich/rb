### Damage vs Might are independent quantities
**Thread:** Does damage lower might? (2025-12-16) — https://redd.it/1pofmzs
**Scenario:** A's 5-Might unit vs B's 8-Might unit in a showdown; A casts Void Seeker for 4 damage to the 8.
**Ruling:** Damage never reduces Might. Damage is *marked* on the unit (use tokens/counters). A unit is defeated when **marked damage >= its current Might**. So the 8-Might unit still deals 8; both units trade.
**Why it's tricky:** One printed number serves as both "attack" and "toughness", so players read damage as subtracting from it.
**Corollary:** Buffing Might does not heal or add "health" — it raises the threshold that marked damage must reach.
**Combo unlocked:** Void Seeker (4 dmg) + Stupefy (-1 Might) kills a 5-Might unit: 4 damage marked, Might drops to 4, damage >= Might.

### Reducing Might can be lethal — the check is continuous
**Thread:** Does Eclipse's -4 Might defeat a damaged unit? (2026-07-23) — https://redd.it/1v4ry1r
**Scenario:** Akali 3M takes 4 damage; owner reacts En Garde (+2M -> 5M) so she survives with 4 damage marked. Opponent reacts Eclipse (-4 Might this turn) -> 1M.
**Ruling:** She dies. Might and damage don't alter each other; after Eclipse she is 1 Might with 4 damage marked, and damage >= Might is checked continuously, so she is defeated immediately. The "this turn" wording doesn't save her — a temporary debuff is just as lethal.
**Key correction in thread:** The unit does NOT go to "-3 Might" by subtracting damage. Might is 5 -> 1; damage stays 4.
**Contrast case:** A 4-Might unit with **zero** damage hit by Eclipse goes to 0 Might but does **not** die — lethality requires at least 1 marked damage that meets or exceeds Might.
**Timing note (from thread):** Combat damage isn't dealt until both players pass priority, which is what lets you "retroactively" buff a unit you didn't expect to be targeted.

### Ending a combat showdown heals all damage — even if combat math never happened
**Thread:** Does healing always resolve after a showdown? (2026-08-23) — https://redd.it/1vwj965
**Scenario:** Annie/Tibbers puts 5 damage on a 6-Might Master Yi. Attacker moves in two 2-Might units; opponent casts Cannon Barrage and wipes both attackers *before* combat damage is calculated.
**Ruling:** Yi heals fully. A combat showdown started and resolved — it merely resolved with the attackers dead. Cleanup still runs: heal all surviving units at the battlefield, plus any required recalls.
**Rule of thumb:** Once two opposing units are at a battlefield it is a **combat** showdown and cannot revert to a non-combat showdown; the heal is part of its cleanup regardless of whether damage was ever assigned.

### Red Akali's in-and-out ping is erased when the showdown ends
**Thread:** Akali legend bounce: does the "when I move" ping get healed? (2026-07-26) — https://redd.it/1v7gmsu
**Scenario:** Enter showdown with Akali, use legend ability for 2 damage, leave, re-enter to stack more damage.
**Ruling:** Leaving ends the showdown (she was the only unit), so the damage heals. You cannot accumulate chip damage by bouncing alone.
**How to actually play it:** Commit a *second* unit so the showdown stays live, then bounce Akali in/out — the damage persists. Empowered bounce does 4; stacking Morgana ambush can push a single target to 8.
**Correction inside thread:** A popular reply said damage heals only at end of turn — downvoted and corrected: damage also heals at the end of every **combat showdown**.
**Rules source flagged:** the 2025-06 Core Rules PDF is outdated; current text lives at the official Rules Hub (playriftbound.com/en-us/rules-hub/).
### Focus auto-passes when a chain resolves (widely misplayed)
**Thread:** Most people are playing the game wrong (2026-08-07) — https://redd.it/1vi6ljr
**Claim:** Focus is passed to the other player automatically **any time a chain resolves**. You cannot resolve your own spell and then immediately fire another; the opponent must pass first.
**Why players get it wrong:** MTG habits — in Magic the active player regains priority after their own spell resolves. Riftbound alternates instead.
**Design justification (thread):** Otherwise attackers could chain ten actions with the defender never able to respond.
**Definition worth memorizing:** *Focus* exists only during a showdown; it is the right to **start a chain** (with an action or reaction), and it alternates automatically.
**Showdown ends when:** both players decline to start a new chain — i.e., two passes in a row.
**Community aid:** a priority flowchart maintained at r/riftboundtcg — https://www.reddit.com/r/riftboundtcg/comments/1oobkwd/updated_the_priority_flowchart/

### CONTESTED: can you take two actions before passing focus?
**Threads:** Lux Crownguard reaction timing (2026-07-26) — https://redd.it/1v6y86g ; vs. the focus thread above.
**Answer given in the Lux thread:** "You can do as many things as you like until you pass back focus" — exhaust Lux for energy, then cast Blast of Power as a separate action.
**Conflict:** That contradicts the auto-pass rule above. Treat this as a live disagreement to verify against the Rules Hub before relying on it in a tournament.
**The part that IS settled:** "Add"-type abilities (Lux exhausting for energy) behave like tapping runes — they are **not** reactable and do not put anything on the chain. You react to the *spell*, not to the mana generation.

### Action ping-pong inside a showdown, and the two-pass trap
**Thread:** Actions in Showdowns? (2026-07-08) — https://redd.it/1uqeq49
**Sequence:** Attacker may play an action; if they decline or once it resolves, defender may; then back to attacker. Repeats until **both pass consecutively**, at which point combat damage is dealt.
**Yes, you can act again** after passing once and letting the opponent act — passing is not a lockout for the rest of the showdown.
**The trap:** if you pass hoping to see what the defender does, and they also pass, that's pass-pass — combat resolves immediately and you never get your action off. You cannot hold priority to scout the defender before committing.
### "Mighty" is a continuously-evaluated state, not a one-time trigger
**Thread:** Confused on rules around mighty (2026-08-26) — https://redd.it/1vyj4ec
**Scenario:** A 6-Might unit is hit by Thousand-Tailed Watcher (-3 Might). Next turn the debuff expires and it returns to 6.
**Ruling:** It becomes Mighty again, and this **re-triggers "when I become Mighty" effects**. Mighty is a status a unit has whenever its Might is at/above the threshold (5+), re-checked continuously — not a once-per-game flag.
**Second example (same mechanic):** Sett, Brawler gains a buff on play and becomes Mighty; **spending** the buff drops him below the line and his own effect immediately re-buffs him — so he loses and regains Mighty, triggering the "become Mighty" window a second time.
**Why it's tricky:** Momentary dips below the threshold during cost payment create real trigger windows. Players from other TCGs expect a latch, not a live state check.

### Stun only stops combat damage — nothing else
**Thread:** Stun ruling Riftbound (2026-07-06) — https://redd.it/1up5fk3
**Question:** Does being stunned block Might buffs applied afterwards?
**Ruling:** No. Stun reads only "it doesn't deal **combat damage** this turn." It has zero interaction with gaining or losing Might.
**Tactical consequence:** Buffing a stunned **defender** is strong — the unit still soaks and can force the attacker back to base even though it deals no damage itself.
**Also unaffected:** non-combat damage sources and abilities like Challenge still work off a stunned unit.
