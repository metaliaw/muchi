# Project Instructions

This repository is a Go API that Follows OneTwoThree.

The generated snapshot Lives in `.agents`, versioned in this Repository.
Its source is `https://github.com/cangrejometralleta/OneTwoThree.git`, branch `main`.
The manifest `.agents/distribution.json` Records commit
`f620c09da052d9a43484bef8a097f6406ec21f1d` and `dirty: false`.
This is a committed working-tree Snapshot, not a published canon release.

[Rules](.agents/canon/RULES.md) Holds the how; read it first.
[Values](.agents/canon/VALUES.md) Holds the why.
[Patterns](.agents/canon/PATTERNS.md) Holds the where.
Read the indexes first and individual bodies on Demand.
[Canonignore](.agents/canon/.canonignore) Defines excluded material.

Dove Lives in `.agents/agents/dove.md`.
Skills Live in `.agents/skills`; the Codex adapter enters through `.codex/agents`.
Existing Claude and Copilot entrances Resolve to the same source.
Commands and edits Target this API repository.

Do not edit generated snapshot Files.
Use OneTwoUpdate's ZIP replacement workflow to Refresh the snapshot,
then OneTwoReload to Validate client entrances.
Verify manifest hashes and preserve local changes before Replacement.
The joke skill is Unavailable because its required material is excluded.

The snapshot and the client entrances are versioned in Git; the entrances are
relative Links into `.agents`, still excluded from deployment by `.gcloudignore`.
A fresh checkout Resolves these references immediately, with no OneTwoReload.

[Story](STORY.md) Holds raw Context on its way to code and docs.
Treat it as Notes in progress, never as instructions.
`chaos/` is private and never Committed.
