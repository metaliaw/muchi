# The Path that Travels

- An absolute Path Names a Machine.
  A relative one Names a Relationship.
  Only the second Survives a Clone.
- `/home/someone/src/project` Works for one Person, once.
  `../project` Works for anyone who Keeps the two Checkouts side by side.
- The Layout between two things Outlives the Place they Sit in.
  Write down the Layout, never the Place.
- A symbolic Link is the same Idea Written into the Filesystem.
  It Points by Relationship, so it Travels with the tree it Lives in.

```text
CLAUDE.md      -> AGENTS.md
.claude/agents -> ../.agents/agents
.claude/skills -> ../.agents/skills
```

- Every Target above Starts from the Link, never from a Root.
  Move the whole Repository and nothing Breaks.
  Move one Half alone and the Link Says so, loudly.
- A relative Link Breaks only when the Pair Moves apart.
  An absolute one Breaks when anything Moves at all.
- The same Instinct Writes a Pair of Repositories into a Memory.
  `../../owner/other-repo` Holds on any Machine that Keeps the Layout.
  [Paired Repositories](../rules/paired-repositories.md) Makes it a Rule.
- One Source, many Entrances is this Pattern in a Tree.
  The Link Shares the Instructions; it never Copies them.
  [Vendor Integration](../rules/vendor-integration.md) Holds the Boundary.
- A Link Shares Files, it does not Convert them.
  Where a Tool cannot Follow one, Generate the File from the Source
  and Mark it Generated, never Maintain a second Copy by hand.
- A Path that Travels is Verified, not Trusted.
  Check the Target Exists before the Work Depends on it.
- Portable does not mean Universal.
  A Link Needs a Filesystem that Keeps it, and a Checkout that Honours it.
- Convention over Configuration, and Simplicity over Coverage.
  A Layout every Checkout Keeps Beats a setting that Bends to each one.
  Declining a Case is allowed: say it is not Supported, and Stop.
  A Promise to cover every Machine is the Configuration this Pattern Avoids.
