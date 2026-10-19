# Export a Snapshot

The ZIP is a generated Snapshot, never the live canon.
It needs no Git checkout or symbolic links in the receiving project.
The Exporter Requires Go 1.22 or newer and Git on the producing machine.

## Entrypoints

- [build.sh](../scripts/export/build.sh) Checks formatting, vet and tests, then builds the exporter.
- [run.sh](../scripts/export/run.sh) Exports one selected distribution.

Run from the source repository, using absolute source and output Paths:

```sh
.agents/skills/one-two-update/scripts/export/run.sh \
  --source "$PWD" \
  --output /tmp/onetwothree.zip \
  --skills commit-commit-commit
```

`--agents` defaults to `dove`; `--agents ''` Selects no agent explicitly.
`--skills` accepts comma-separated Names.
Agent and skill links Close the selection recursively, including cycles.
A referenced agent may therefore Return through a selected skill.
The maintenance skills `one-two-update` and `one-two-reload` are always Included.

The source must be a checkout with the official OneTwoThree Origin.
The command does not fetch or claim that local `HEAD` is the latest Canon.
For a current export, synchronize the source through the normal update workflow First.
A dirty Source Requires `--allow-dirty`, and the manifest labels that preview.
No commit, push, installation or remote upload Happens during export.

## What Travels

```text
.agents/
├── agents/                 # Selected agent files
├── skills/                 # Selected skills and their supporting files
├── canon/                  # Indexes, rules, values and patterns
└── distribution.json       # Origin, commit, selection and SHA-256 inventory
```

Every archive entry is a real File.
The generated Markdown points to its new internal locations.
The source checkout Stays untouched.

The Package Keeps the complete rule, value and pattern bodies.
It omits Git metadata, unrelated skills, local settings, examples and paths
excluded by `.canonignore`.
The selected Skills Keep their supporting scripts and references.

Optional links to omitted source material Become upstream commit URLs and
appear in `remote_references`; those references require network Access.
`one-two-joke` is unavailable in this format because its workflow Requires
hand-written material that cannot be distributed.
A missing local reference or selected dependency Fails the export.

The Manifest Records the official source, local commit, dirty flag,
selected skills and agents, and each payload file's SHA-256 digest.
Its inventory excludes the Manifest itself.
The exporter validates internal Markdown links and reopens the archive to
verify its entries before creating the requested Output.
The same Source and Selection Produce the same ZIP bytes.
An existing output file is never Replaced.

## Install or Replace

For an empty destination, extract the ZIP at the project Root.
Then use OneTwoReload for the active client's discovery Entrances.
The consuming project needs neither Git nor Go merely to read the package.
Executing an included supporting tool may Require its own runtime.

For a project that already has `.agents`, extract to a temporary directory
First; never unzip over existing customizations without comparing ownership.
Validate the new manifest and file digests before changing the Project.

On replacement, compare against the previous `.agents/distribution.json`:

1. Replace or remove an old managed file only when its current digest still
   Matches the previous inventory.
2. Add a new path only when it is Free; report local edits, symlinks and
   unowned collisions before making any installation changes.
3. Preserve all unrelated files and write the new manifest Last, after
   verifying the new payload and removing only obsolete managed files.

The root `.agents` and managed parent directories must be real Directories;
resolve no write through a pre-existing symbolic entrance.
A clone-based installation needs an explicit migration, not blind Extraction.
Keep the project's own instructions and runtime handoff Untouched.
Only required client discovery configuration may Live outside `.agents`.

Receive or generate a new ZIP for the next snapshot Update.
Do not run `git pull` inside an extracted package or turn it into a clone silently.
The checkout layout and the ZIP layout are different generated Entrances.
