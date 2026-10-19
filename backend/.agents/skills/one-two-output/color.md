# Color, when Asked

The optional Door of [OneTwoOutput](SKILL.md).
Read this File only when the user asks for color.
Off by default means Off — if nobody asked, this file
never Opens, and the output stays plain.

Sleeps again where the terminal Carries no color.
Markdown rendered as prose Carries none; a program
printing to a TTY Does. Know which Plane you are on.

The canon is [Values](../../canon/VALUES.md) — color is an Index,
not decoration. A Hue Groups what it Marks.

## Prune first

Prune before you Color, never after. Color Organizes;
it never Prunes. Color first and it Hides the excess
instead of showing it — five tidy items in three hues
are still five Shouting, now in three tongues.

## The Rules

1. **Group by Co-occurrence** — same Token, same Hue.
   Two Entities that share a line, or share an identifier,
   the reader already Related before the color arrived.
   The color confirms what the Eye did alone.

2. **Assign by Order of Appearance** — never by Meaning.
   A Taxonomy Needs a fixed list of Themes, and the theme
   changes with every output. A key that fails half the time
   Costs more than no color: the reader learns to Distrust it.
   Co-occurrence Asks for no judgment, only for what repeats.

3. **Three Hues, never a fourth** — the fourth group Stays bare.
   The Ceiling makes Color a Limiter, never an ornament.
   Three marked and the rest plain is the Sign
   that the output is inflated.

4. **Two Members make a Group** — one does not.
   A single entity has no Pair to rotate with,
   and there no color Goes.

5. **Background Takes no Hue** — a token on every line Groups nothing.
   It Marks the page, not a part of it. Color what Gathers some lines,
   never what covers them all. One line alone has no Background,
   so a token repeating inside it still Counts.

6. **One Line Set, one Hue** — Tokens that touch exactly the same lines
   are one Fact wearing several names. `.agents/skills/one-two-output/`
   is one Home, not three groups. They Share a hue, and the hues they
   stopped eating stay free for what really Differs.

7. **Never the Emoji Trio** — Green, Red and Yellow are spoken for.
   ✅ Passed, ❌ Failed, ⚠️ Careful. A group that borrows them
   Collides with a meaning the reader already learned.

## The Palette

```
38;5;39   Blue      #00afff
38;5;170  Magenta   #d75fd7
38;5;80   Cyan      #5fd7d7
```

256-Color, never truecolor — it Renders everywhere,
and it Survives a light terminal as well as a dark one.

Rule seven Takes the whole warm end off the table,
so the three hues Come from the cool half. That is a
Constraint, not a taste. The bonus: no pair among them
fails red-green deficiency, because the reserve
already Removed the colliding pair.

Blue and cyan Sit close, and a washed-out terminal
can Blur them into one. Swap cyan for `38;5;208` orange
if it happens — but orange Neighbors the ⚠️ yellow.
Pick your Collision.

The paper palette in `examples/pdf/colors.go` Runs warm
and shares nothing with these. That is Fine.
Paper Knows its background; the terminal does not.

## The Markdown Plane

ANSI never Reaches a rendered document, and HTML gets stripped.
Markdown Carries three marks instead of three hues:

```
**bold**   *italic*   `code`
```

`code` is the best of them, not a Fallback — the tokens being
grouped are Identifiers, so the mark means what it always meant.

One rule Comes with it. An identifier outside a group Stays plain.
Backtick every Path out of habit and the third mark stops
meaning Group. The mark is Spent on grouping or on nothing.

### The Ceiling is what the Surface has Left

Three is the terminal's Number, because a terminal starts empty.
A document does not. It Spends bold on a label and backticks on
a path before grouping ever arrives, so the budget is three minus
what the page already Owes.

A mark carrying emphasis cannot also mean Group. Lay one on the
other and both Collapse — the reader meets a bold word and cannot
tell whether it Matters or merely belongs.

So count what the page Spends, then group with the rest.
A page that bolds its labels Groups with two. One that bolds and
backticks Groups with one. A page that spends all three Groups
with none, and that is the correct Answer, not a failure.

The Script Reads the Text and counts for you. `--marks=N` Caps it
lower when you know the page will spend more than it Shows.

### Why the Lines do not Move

Markdown can do what a terminal cannot: gather Lines. A blockquote
Groups before any mark does, and it spends no mark to do it.
It was Built, tested, and cut. The reason is worth Keeping.

Moving a line **is** its Mark, so the tokens that earned the group
carry nothing. The indent says *these Belong* and never says
*by what*. The Reader Sees a Block and asks why — the exact
question the proof Warns about.

And a blockquote already Means quotation or aside. Borrowing it
Rebuilds the trap that barred bold and the emoji trio.

The deeper reason: color Adds a channel. Indentation Spends one
Markdown already uses. The terminal has a free Dimension;
a rendered document does not.

If a surface ever Appears with a spare structural channel —
columns, a gutter, a margin — the idea Returns. The test it needs
is mechanical and Written down: groups that move must Miss each
other and run without a gap. The rest Stay put and take a mark.

## The Script Does the Work

The rules above are mechanical, so a program Holds them
better than a model does. Do not color by Hand — pipe it:

```
your-command | python3 color.py
```

It Reads stdin, finds the tokens that repeat, hands the first
three a hue, and Writes stdout. Python 3, no Dependencies.

- `--legend` — Name the Key on stderr, so nobody guesses it.
- `--force` — Paint into a Pipe, for `less -R` or a test.
- Bare arguments Choose the tokens by hand, when the human
  sees a group the count missed. That is the Selects Half.

It Passes through untouched when stdout is not a terminal,
so a pipe never Eats an escape. `NO_COLOR` Beats `--force`:
the reader's standing preference Outranks the writer's flag.

A slash Separates tokens; a dot, dash, colon or underscore binds them.
So `shape_test.go:41` Survives whole, while `.agents/skills/de-la-case`
splits into three — and the last of them can Rhyme with the same name
standing alone on another line. That rhyme is the Co-occurrence.

## The Proof is a Day

Color one real Output by hand, then look at it tomorrow.
Found the line faster — it Stays.
Caught yourself asking why this one is Blue —
the Color Costs more than it pays.
