---
name: one-two-update
description: "Fetch the latest OneTwoThree canon from main on its remote repository and connect the current project to a reduced copy of it, without hand-copying any file. Use when the manifesto rules may be stale, when a project first adopts OneTwoThree, or when the user asks to update, sync, pull or export a portable ZIP of selected agents and skills."
---

# OneTwoUpdate

A Fetcher of the Canon and an exporter of portable Snapshots.
It brings `main` from the remote OneTwoThree,
Keeps only what Governs, loads only the index,
and leaves every local file the project owns untouched.

The Canon Lives in [The Head is the Canon](../../../rules/the-head-is-the-canon.md),
[Vendor Integration](../../../rules/vendor-integration.md)
and [Canonignore](../../../rules/canonignore.md).
The Loading Lives in [OneTwoReload](../one-two-reload/SKILL.md).
ZIP export and installation Follow [Export a Snapshot](references/zip.md).

The Source is one Address:

```text
https://github.com/cangrejometralleta/OneTwoThree.git   branch: main
```

```mermaid
flowchart TD
    START["update or export"] --> MODE{"ZIP requested?"}
    MODE -- Yes --> ZIP["Export a snapshot · Verify archive · Stop"]
    MODE -- No --> HERE{"Inside OneTwoThree?"}
    HERE -- Yes --> PULL["Fast-Forward main · Stop"]
    HERE -- No --> LINK{"Link Exists?"}
    LINK -- Yes --> FOLLOW["Follow the Mechanism already Chosen"]
    LINK -- No --> ADOPT["sparse, blobless Clone · relative Link"]
    FOLLOW --> FETCH["Fetch main"]
    ADOPT --> FETCH
    FETCH --> DIVERGE{"Local Edits in the Canon?"}
    DIVERGE -- Yes --> STOP["Report the Divergence · Change nothing"]
    DIVERGE -- No --> APPLY["Fast-Forward to origin/main"]
    APPLY --> REDUCE["Prune to the governing Paths"]
    REDUCE --> VERIFY["Verify Links · Read the new Canonignore"]
    VERIFY --> REPORT["Report the Range · Name OneTwoReload"]
```

## When it Runs

Invoke when the user Asks to update, sync or pull the manifesto,
when a project first Adopts OneTwoThree,
when a rule read here disagrees with the rule named upstream,
or when the user requests a ZIP of selected agents and skills.

Do not invoke to load skills into a client; OneTwoReload Does that.
Do not invoke to open a session; YoYoYo Pulls the project's own branch.

## Choose the Distribution

A ZIP request goes directly to [Export a Snapshot](references/zip.md),
even inside the source Repository.
Use the checked-in exporter; never assemble or rewrite a package by Hand.
A receiving project with `.agents/distribution.json` keeps its snapshot
workflow until the user requests a different Mechanism.

The sections below Govern clone-based adoption and synchronization.
Generated ZIP files may adjust relative links; the upstream checkout Stays unchanged.

## Two Reductions, not one

`Memory` and `Context` are cut by different Means.
Do both, and never confuse one Report for the other.

**Memory** is what Lands on disk. Check out only the governing Paths:

```text
AGENTS.md  RULES.md  VALUES.md  PATTERNS.md  .canonignore
rules/     values/   patterns/
.agents/agents/<selected-agent-files>
.agents/skills/<selected-skill>/
```

**Context** is what Enters the prompt. Load only the Index:

```text
AGENTS.md  RULES.md  VALUES.md  PATTERNS.md
```

The Index Names every Rule with one line and one link.
A Body is read the turn its rule is invoked, and not before.
Three indexes cost about 7 KB; their bodies cost about 50 KB;
the whole repository Costs about 2.4 MB.

Never preload `rules/`, `values/` or `patterns/` wholesale.
The Index Exists so an agent can know a rule is there
without paying to Read it.

## Inside the Canon Repository

First Ask whether the current repository *is* OneTwoThree.
Compare the Origin address, not the directory name.

When it is, there is nothing to Vendor or reduce.
Fast-forward `main` from `origin`, Report the range, and Stop.
Never vendor the Canon into the repository that writes it.

## What it Reads

Read only what Names the connection:

1. **Origin** — the current repository's Remotes.
2. **Link** — an existing submodule, subtree, clone or symbolic link
   that already Points at OneTwoThree.
3. **State** — whether that link Carries local commits or dirty paths.
4. **Canonignore** — the incoming `.canonignore`, read after the Fetch.

`AGENTS.md` or local documentation may Name a different entrance.
Follow what the project Documents over what this skill assumes.

## Follow the Mechanism already Chosen

A project that already tracks the Canon has Answered this question.
Detect which answer it Gave, and use it:

| What you Find | What you Run |
| --- | --- |
| A Submodule pointing at OneTwoThree | `git submodule update --remote` |
| A Subtree | `git subtree pull --prefix <path> <url> main --squash` |
| A Clone under a Cache or Vendor Path | `git fetch origin main` then Fast-Forward |
| A symbolic Link to a Clone elsewhere | Update the Clone the Link Resolves to |
| `.agents/distribution.json` | Follow the ZIP replacement workflow; preserve local edits |
| Nothing | Adopt, below |

Re-apply the Path reduction after a fetch that widens it.
Never convert one Mechanism into another to make the update easier.
A project changes how it Tracks the canon on purpose, never as a side effect.

## Adopt, when Nothing Exists

Keep the distribution inside `.agents`, including the clone's Git metadata.
Use this Layout for a new installation:

```text
.agents/
├── canon/                  # Reduced upstream clone, including .git
├── agents/
│   ├── dove.md -> ../canon/.agents/agents/dove.md
│   └── dove.toml -> ../canon/.agents/agents/dove.toml
└── skills/
    └── <name> -> ../canon/.agents/skills/<name>
```

Keep existing project agents and skills Untouched.
Create individual links only where the destination is Free.
A Collision Needs Resolution before any replacement.

Select the requested agents and skills, then Close their dependencies:

- Include each referenced skill and agent, recursively, until the set Stops growing.
- Keep the selected skill's supporting files when its workflow Requires them.
- Include `one-two-update` and `one-two-reload` for distribution Maintenance.
- Keep all three indexes and their bodies so Dove can Read every value and pattern.
- Exclude stories, jokes, examples, development tools and local Settings.

A reference to an excluded resource does not Authorize copying it.
Report a required excluded resource as Unavailable for that workflow;
optional examples may Stay upstream.
Never claim a skill is self-contained when a required resource is Missing.

Create a shallow, blobless clone without an initial Checkout:

```sh
git clone --depth=1 --filter=blob:none --no-checkout --single-branch --branch main \
  https://github.com/cangrejometralleta/OneTwoThree.git .agents/canon
git -C .agents/canon sparse-checkout set --no-cone \
  '/AGENTS.md' '/RULES.md' '/VALUES.md' '/PATTERNS.md' '/.canonignore' \
  '/rules/' '/values/' '/patterns/' \
  '/.agents/skills/one-two-update/' '/.agents/skills/one-two-reload/'
```

Before checkout, add the selected agent files and skill directories with
`git sparse-checkout add --no-cone`, using anchored Patterns.
Inspect upstream files through Git to close dependencies before exposing Links.
Then run `git -C .agents/canon checkout main` and create the relative Links.
Non-cone mode Keeps unrelated root files off disk.
A shallow Clone Limits History; deepen only when an update needs ancestry.

Resolve a linked skill or agent to its physical File before following its
relative documentation links: `../../../rules/` then stays inside the clone.
The working repository remains the target Project for commands and edits.
Never rewrite upstream files to adapt their Paths.

Name the canonical entrance in the client's smallest supported loading
instruction, including the physical-path resolution Rule.
Only required discovery links or configuration may Live outside `.agents`.
Reuse the project's existing instruction file without replacing its Content.
Runtime outputs such as `.handoff.md` are not installation Files;
keep their established locations unless the user requests a separate Change.

An explicit request to install Authorizes adoption.
Otherwise ask before adding the Dependency.
For an existing installation, preserve its Mechanism unless relocation was
requested; validate the new entrance before removing an old managed Path.

## Never Overwrite what the Project Wrote

Fast-Forward only.

When the local canon Carries commits that `origin/main` does not,
stop and report the Divergence. Those commits are either
a Fork worth keeping or an Edit that belonged upstream,
and only the user can say which.

- Never force push, force pull, reset hard or Discard local commits.
- Never Merge or Rebase the canon into the project's own history.
- Never edit a File under the vendored canon to resolve a conflict.

## Read the new Canonignore

The Canon Arrives with its own Boundary; read it after every fetch.
Paths it lists are carried and not taught — `jokes/`, `stories/`,
`STORY.md`, `chaos/` among them.

The reduction already keeps them off the disk,
so a path that appears despite it is a Signal, not a convenience:
the Boundary Moved upstream, and the sparse set has to move with it.

Read them freely where they Exist. Copy none of them into the Project.
An update that teaches from an ignored path Made the canon worse,
not fresher.

## Verification

1. The fetched ref is `main` from the OneTwoThree address, not a Fork.
2. The Link Resolves from the project root.
3. `RULES.md`, `VALUES.md` and `PATTERNS.md` Exist at the linked root.
4. Every link inside those three indexes Resolves to a checked-out body.
5. Only governing paths and selected customizations Landed in the checkout.
   The clone, its metadata and every customization source Stay inside `.agents`.
   Each required skill dependency Resolves from the physical source file.
6. The range between the old and new commit is Nameable.

A Fetch that Moved no Ref is `Already Current`, not `Updated`.
An index link that resolves to nothing means the sparse Set is too narrow.

## What it Returns

```text
✅ Canon Updated — 63a3010..f3bdfa6 (4 Commits)

**Source** — cangrejometralleta/OneTwoThree, main.
**Entrance** — .agents/canon; selected agents and skills exposed by relative links.
**Changed** — 2 Rules, 1 Pattern.
**Reduced** — 58 KB on Disk, 7 KB Loaded.

**Next** — Reload the Client with OneTwoReload.
```

When the ref did not Move:

```text
✅ Already Current — main at f3bdfa6.
```

When the local canon Diverged:

```text
❌ Canon Diverged — 2 local Commits not on origin/main.
Nothing Changed. Name whether they are a Fork or an Edit to Send upstream.
```

## Bounds

- Never copy a canonical File into the project by hand.
- Never vendor the Canon into OneTwoThree itself.
- Never fetch from a Fork while claiming the canon.
- Never adopt a new Dependency without user authorization.
- Never Fast-forward past local Commits, and never discard them.
- Never preload a Body the index can name for free.
- Never widen the sparse Set to make one read easier; read it on demand.
- Never teach from a Path `.canonignore` lists.
- Never report `Updated` when no ref Moved.
- Never claim the client sees the new canon; OneTwoReload Proves that.
