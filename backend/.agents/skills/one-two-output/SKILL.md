---
name: one-two-output
description: Format terminal output for readability — DeLaCase prose, one emoji to mark the state (✅/❌/⚠️), short heartbeat lines, a short line after a long one for contrast, breaks at grammatical seams, the least words that carry the meaning, narrate what happened not how. Use when composing output the user will read, especially results, summaries, or status reports. The gain is readability, not token budget — prose is a small share of a session's spend, and cutting too hard costs more in questions back than it saves. It shapes output from the moment it loads onwards; it never rewrites, replays or restates output already printed.
---

# OneTwoOutput

Compose Guidelines, not a converter and not a checklist.
Case converts one name, Refactor guides the writing,
and this shapes what the terminal Says.
The [Dove](../../agents/dove.md) agent Reads explanations.

The Terminal is one of three Planes —
Files, Code and Terminal. Look, Work and Talk.
This Skill Owns the Talk.

## Scope

The Skill shapes what Comes next, never what came before.
Output already printed Stays as it stands.
No rewrite, no replay, no second Telling —
the next Line is where it takes hold.

The Canon Lives in [Rhythm](../../canon/rules/rhythm.md),
[Seams](../../canon/rules/seams.md),
[Emoji](../../canon/rules/emoji.md)
and [Structure](../../canon/rules/structure.md).

## How to Invoke

Three Doors, all forward:

1. **By Name** — `/one-two-output`.
   The line after the call is the first one Shaped.

2. **By Ask** — "Shape the Output", "Talk in the Terminal Rules".
   Holds for the rest of the session, until the user Says stop.

3. **By Match** — the description fits what you are about to Print:
   a result, a summary, a status report.

### Invoked mid-Session

Before the call, the terminal Said:

```
tests finished, 1 of 2 packages ok, the output package failed on an
assertion in shape_test.go at line 41, whole run took 0.3s
```

The User Calls `/one-two-output`.
That Paragraph Stays as it stands — no rewrite, no replay.
The next Report Reads:

```
❌ Output Tests Failed
rules Passed in 0.31s
shape_test.go:41 broke the Assertion
```

The path and the line Survived the cut; only the connectors fell.

## Before printing output, apply

1. **Case** — Follow [DeLaCase](../../canon/rules/de-la-case.md).
   The grammatical initial is free; each passage between punctuation marks
   may carry up to three emphasis Capitals.
   Identify the important Entities and their Interactions.
   Two entities and one interaction are a useful shape, never a required formula.
   Proper names and acronyms keep their established Spelling.
   Three spent is a Ceiling, never a quota; follow the canon for boundaries.
   Final text shown to users in an interface uses ordinary capitalization;
   use DeLaCase there only when explicitly requested.

2. **State** — Mark the State with one emoji.
   ✅ for Passed, ❌ for Failed, ⚠️ for Careful.
   One per Line at most. Two Compete, three are noise.
   Never in a key the code Compares.

3. **Length** — A Line Takes a Heartbeat to read.
   If it runs longer, break it at a Seam.

4. **Contrast** — A short line after a long one
   Lands like a chorus. Uniform text hides what Matters.
   Vary the Length; the ear remembers the turn.

5. **Seams** — Break where the grammar Bends.
   A Conjunction, a Comma, a Preposition.
   Never inside a unit that Reads as one.

6. **Words** — Use the least meaningful Words possible.
   Every extra word is another to Hold in mind.
   The cut Lands on someone — see Where the Load Goes.

7. **Narrate** — Name what Happened, never how.
   The Libretto Names the Plot; the Provider holds the How.

## Paragraphs and Channels

For prose reports, follow [Paragraph](../../canon/rules/paragraph.md):
Open with the Claim and keep one idea per paragraph.
When assigning color or markdown marks, Follow
[Channels](../../canon/rules/channels.md): a mark already carrying meaning
cannot Carry a second. Plain output Remains a valid choice.

## Color

Off by default, and off means Absent.
If the user Asks for color, read [color.md](color.md);
otherwise the output stays plain and this line is the whole rule.

## Where the Load Goes

Fewer Words do not Delete the Load. They Move it.
Know where it Lands before you cut.

1. **Forward, to the Model** — the shaping Happens before the line.
   The Thinking Costs Tokens the output never shows.
   The Saving is Real only when the line is read once.

2. **Outward, to the Reader** — every word dropped
   is a Gap the reader fills. One question back
   Costs more than the words it saved.

3. **Down, to the Files and the Code** — what leaves the terminal
   Lives in a path, a log, a diff. Cut from the Talk,
   never from the record.

So cut the Connectors and the how. Keep the Referent —
the Path, the Name, the Number, the Error.
A line that saves five words and hides a path Saved nothing.

## What it Saves

Not much, in Tokens. Say it plainly.

This File Costs near a thousand Tokens on load.
A shaped Report Saves twenty to a hundred of its own.
The tenth report is where it Breaks even.

And prose is the small Half of a session.
File reads, tool results and the context resent each turn
Dwarf what the terminal Says. Halve the Words
and the bill Moves a percent, maybe two.

The upside is Capped. The downside is not.
A Line that hides a Path Buys one question back,
and that Question Costs more than the skill ever saved.

So judge it on Reading, never on spend.
The Verdict on line one, the Referent on line two,
the state in one glance — that is the gain.
Fewer tokens are a Side Effect, never the point.

## Example

Raw:

```
build succeeded, 142 files, 3 warnings in vendor.go, took 18s, tests pending
```

Shaped:

```
✅ Build Closed in 18s
142 Files, 3 ⚠️ in vendor.go
Tests still Pending
```

Build, Files and Tests are the Entities.
Closed and Pending are the Statuses.
The ⚠️ Marks the warning without a word. Three short Lines
after the long raw one, each a heartbeat.
