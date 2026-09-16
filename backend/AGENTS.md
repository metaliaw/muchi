# AGENTS.md

This Repository is a Go API that Follows the OneTwoThree Canon.
It does not Write the Canon; it Reads a reduced Copy of it.

## The Canon Arrives Vendored

The Source is one Address:

```text
https://github.com/cangrejometralleta/OneTwoThree.git   branch: main
```

It Lives in `.agents`, a flat Copy of the Paths that Govern.
Every canonical Entrance here is a relative symbolic Link into it:

| Link | Resolves to |
| --- | --- |
| `RULES.md` `VALUES.md` `PATTERNS.md` `.canonignore` | `.agents/` |
| `rules/` `values/` `patterns/` | `.agents/` |
| `.claude/` `.codex/` `.github/` | `.agents/` |
| `CLAUDE.md` | `AGENTS.md` |

`.agents` is the one Source, and every Client Enters through it.
Never Edit a File under `.agents`. It is not Ours to Write.
Update it with OneTwoUpdate; Load it into a Client with OneTwoReload.

### The `.canon` Clone is Gone, and this Note Says why

The Canon used to Arrive in `.canon`, a sparse and blobless Clone,
and `.agents` Linked into it. That Clone is Removed. The Copy now Sits
directly in `.agents`, taken on 2026-09-16 from this Commit:

```text
5f3e6ec7e20723449f4a03e079ba240af2f1af76   Separate choices from session sequencing
```

That Line is the whole Provenance. A flat Copy Forgets where it Came from,
so nothing but this Note can Say which Head it Answers to.
Read it as a Fork until someone Restores the Clone — see
[The Head is the Canon](rules/the-head-is-the-canon.md),
which Asks for the Head and Accepts no older Commit.

This is a deliberate Simplification, not the canonical Shape.
Restoring the Clone Means running OneTwoUpdate and letting it Rebuild
`.canon` and the Links; the flat Copy is then Redundant and Goes.

## The Cache is not Carried

This File and `CLAUDE.md` are the only canonical Paths the Repository Carries.
`.agents` and every Link in the Table above are Ignored, because
a Copy is a Cache and the Head is the Claim — see
[The Head is the Canon](rules/the-head-is-the-canon.md).

So a fresh Clone Arrives without them, and with no broken Link either.
Run OneTwoUpdate once; it Fetches the Cache and Recreates the Entrances.
Until then the Links in this File Resolve to nothing, and that is Correct.

## Where to Read

[Rules](RULES.md) Holds the How — Read it first.
It Governs how an Agent should Read and Write here,
including OneTwoThreeCase and the Cadence it Asks to be Read in.

[Values](VALUES.md) Holds the Why.
[Patterns](PATTERNS.md) Holds the Where.

Read the three Indexes. Read a Body the Turn its Rule is Invoked,
and not before.

[Canonignore](.canonignore) Lists every Path the Canon does not Govern.
Read them freely; Copy none of them.
