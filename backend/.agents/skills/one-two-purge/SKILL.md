---
name: one-two-purge
description: "Find an exact sensitive value across files, Git history and local shell histories, then remove it through an explicit detect, confirm and purge workflow. Use when a secret, token, credential or private value may have entered repository or terminal history."
---

# OneTwoPurge

A Remover of sensitive history, not a promise that exposure never happened.
It Detects without echoing, names the affected surfaces,
confirms the destructive boundary, then purges and Verifies.

The first action after a credential exposure is Revocation or Rotation.
History Rewriting Reduces Distribution; it does not make a leaked secret safe.

## The Value

Never ask the user to paste a sensitive Value into chat.
Never place it in a command Argument, generated script, log or report.

Ask the user to enter it directly into an interactive terminal with Echo
disabled and keep it in a temporary environment Variable such as
`PURGE_VALUE`. The Agent Uses only the variable Name.

- Never print, expand, inspect or persist the Variable.
- Never enable shell Tracing while it exists.
- Reject an empty Value before every search or mutation.
- Clear it from the Environment when verification finishes.
- If the client cannot accept secret terminal Input without exposing it to the
  model, stop and give the user a local Command to run themselves.

## Detect

Read three Surfaces. Detection Changes nothing.

1. **Files** — tracked, untracked, hidden and ignored Files in scope.
2. **Git** — reachable Commits, Branches, Tags and local Reflogs.
3. **Shell** — known local history Files for Bash, Zsh, Fish and the active
   shell, plus project-local session Logs explicitly named by the user.

Use fixed-string matching by Default. Treat the Value as data, never a regular
expression. Exclude binary payload output and report only redacted Paths,
commit IDs, ref names and Counts.

Do not search unrelated home Directories merely because they are accessible.
Name every Path outside the current repository and ask before reading it.

Report Detection separately:

```text
Files: 2 Matches in 2 Paths
Git: 4 Commits across 1 Branch and 1 Tag
Shell: 3 Entries in Fish History
```

Never include matching lines or the sensitive Value.

## Confirm

Detection does not Authorize Mutation.

Before purging, Explain:

- the exact Files, refs and history stores that will change;
- whether Git commit IDs will Change;
- whether Tags or Branches require replacement;
- whether a remote Force Push will be needed;
- that existing Clones, forks, caches and logs may retain the old value;
- which Backups will be created and when they will be removed.

Require explicit Confirmation for each destructive boundary:

1. Rewrite repository History.
2. Rewrite each shell history File.
3. Expire Reflogs and prune unreachable Git objects.
4. Force Push rewritten branches or tags.

Never combine those Approvals. Never infer Consent from the original request.

## Purge

Purge only the Surfaces the user confirmed.

### Files

Replace or remove the Value from the current working tree first.
Preserve file Structure and unrelated content.
Do not commit unless the user separately Requests a commit.

### Git

Prefer `git-filter-repo` when installed and supported by the Repository.
Read its installed Help before constructing the rewrite.
Use a replacement File or Callback that reads the value without placing it in
the command line, process list or persistent project files.

Before rewriting:

- require a clean or fully understood working Tree;
- record the current Branches, Tags, Remotes and `HEAD` without secrets;
- create a local recovery Bundle outside the repository with restrictive
  Permissions;
- name the Bundle path to the user;
- refuse to overwrite an existing Bundle.

Do not use `filter-branch` when `git-filter-repo` is available.
Do not delete original refs, expire reflogs or run garbage collection until the
rewritten history passes Verification and the user confirms final pruning.

Never Force Push automatically. Show the affected remote Refs and ask for a
separate Approval. Use `--force-with-lease`, never unconditional `--force`,
when the remote State permits it.

### Shell

Stop or account for active Shells that may rewrite history on exit.
Create a permission-restricted Backup beside neither the repository nor its
tracked Files. Parse the native history Format; do not treat structured Fish or
multiline history as plain independent lines when that would corrupt entries.

Write a replacement File atomically, preserve Permissions and ownership,
then ask the user to restart or reload affected shell sessions.
Never clear an entire History when exact entry removal is possible.

## Verify

Repeat the same fixed-string Detection against every purged surface.

1. Current Files contain zero matches.
2. Rewritten reachable Git History contains zero matches.
3. Confirmed shell history Stores contain zero matches.
4. Refs and repository integrity checks pass.
5. The sensitive Variable is cleared.

A zero Match in rewritten Git does not prove remote caches or existing clones
forgot the Value. Say so.

Only after Verification may the user separately approve deleting recovery
refs, expiring reflogs, pruning objects and removing backups.

## Boundaries

- Never reveal a sensitive Value back to the user.
- Never send it to a network Service or external Scanner.
- Never mutate global credential Stores without an explicit request.
- Never rewrite signed Commits or tags without naming that signatures Break.
- Never rewrite a shared Branch without naming the coordination required.
- Never remove audit evidence required by Law, policy or an active incident.
- Stop when repository ownership, remote authority or History format is unclear.

## What it Returns

Report States, counts and remaining actions only:

```text
✅ Purge Verified
- Files: 0 Matches
- Git: 0 Matches in rewritten reachable History
- Shell: 0 Matches in 1 confirmed History Store
- Credential: Rotation still Required
- Remote: Force Push not Performed
- Recovery Bundle: Retained until final Approval
```

The Report never Contains the Value or a recoverable fragment of it.
