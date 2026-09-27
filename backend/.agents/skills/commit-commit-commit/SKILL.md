---
name: commit-commit-commit
description: Analyze local git changes, explain the features they implement, separate them into coherent commits, commit each feature, and push after every commit succeeds. Use when the user asks to organize, explain, commit, and push local work.
---

# CommitCommitCommit

A Curator, not a commit message Generator.
It Reads the local Changes, finds the Features inside them,
explains the separation, then commits and pushes the work.

The Canon Lives in [Emoji](../../canon/rules/emoji.md),
[Structure](../../canon/rules/structure.md)
and [Channels](../../canon/rules/channels.md).

## What it Reads

Three Sources, all from git. Nothing else.

1. **State** — every tracked, staged and untracked Path:

   ```
   git status --short
   ```

2. **Changes** — staged and unstaged Diffs:

   ```
   git diff
   git diff --cached
   ```

3. **Lineage** — recent Messages and the upstream:

   ```
   git log -10 --oneline
   git status --branch --short
   ```

Read untracked Files before grouping them.
Never infer their Feature from a filename alone.

## What it Explains

Before staging anything, name each Feature and its paths.
Explain why each Group Belongs in one Commit.
Keep unrelated Changes separate, even when they share a file.

```
Commit 1 — Shorten skill names
- Renames the four shared skills and repairs their Links.

Commit 2 — Add the commit workflow
- Replaces the status-only Skill with an explicit commit and push flow.
```

Ask before proceeding when a Path could belong to more than one feature.
Do not stage a mixed Hunk merely because its file is already in a group.

## What it Does

1. Analyze every local Change before staging.
2. Separate changes by Feature, not by file type.
3. Explain the proposed Commits before creating them.
4. Run the narrowest available Validation for each feature.
5. Stage only that Feature, using patch staging for mixed files.
6. Commit with a message that says what the feature Changes.
7. Repeat until the intended local Changes are committed.
8. Push Once, after every commit and validation succeeds.

## Branches Keep the Flow

- Work on `main` when one Change can be finished and integrated at a time.
- Use a feature branch when work needs isolation or can proceed in parallel.
  Branch from `main` and return finished, validated work there.
- On a feature branch, use [Change Growth](../../canon/rules/change-growth.md).
  When the feature reaches three attributable writing turns or three touched
  files, remind both the agent and the user to pause and name its scope before
  adding more. If it has become multiple intents, recommend splitting the
  work; do not silently split or merge it.
- Use `develop` only when concurrent Features need a shared Integration before
  `main`; follow the repository's configured target instead of assuming one.
- [Branches Give Cooperation a Path](../../canon/patterns/branches-give-cooperation-a-path.md)
  describes direct integration and the optional shared integration branch.

## The Boundaries

- Never commit Secrets, generated credentials or ignored files.
- Never Rewrite, amend or squash existing commits unless asked.
- Never Include unrelated local changes to make the tree clean.
- Never Push when validation fails or a commit fails.
- Never force Push.
- Never create an Upstream without naming the branch and asking first.
- If nothing Changed, say so and stop.

## What it Returns

After the push, report the feature Commits and destination:

```text
✅ Pushed to origin/main
- a1b2c3d Shorten skill names
- d4e5f6a Add feature commit workflow
```

## Sources

[Emoji](../../canon/rules/emoji.md) —
one per Line, only to mark a state.
[Structure](../../canon/rules/structure.md) —
Group, Validate and Deliver.
[Channels](../../canon/rules/channels.md) —
plain Beats a compromise the reader has to decode.
