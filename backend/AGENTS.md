# AGENTS.md

This Repository is a Go API that Follows the OneTwoThree Canon.
It does not Write the Canon; it Reads a reduced Copy of it.

## The Canon Arrives Vendored

The Source is one Address:

```text
https://github.com/cangrejometralleta/OneTwoThree.git   branch: main
```

It Lives in `.canon`, a sparse and blobless Clone
Reduced to the Paths that Govern. Every canonical Entrance here
is a relative symbolic Link into it:

| Link | Resolves to |
| --- | --- |
| `RULES.md` `VALUES.md` `PATTERNS.md` `.canonignore` | `.canon/` |
| `rules/` `values/` `patterns/` | `.canon/` |
| `.agents/skills` `.agents/agents` | `.canon/.agents/` |
| `.claude/` `.github/` | `.agents/` |

Never Edit a File under `.canon`. It is not Ours to Write.
Update it with OneTwoUpdate; Load it into a Client with OneTwoReload.

## The Cache is not Carried

This File and `CLAUDE.md` are the only canonical Paths the Repository Carries.
`.canon` and every Link in the Table above are Ignored, because
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
