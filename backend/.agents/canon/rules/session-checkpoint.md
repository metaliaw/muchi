# Session Checkpoint

> Provisional. Written from practice, not yet Weathered.

- A durable Change that lives only in the session is already lost.
  Leave its State outside the head before the turn ends.
- Checkpoint after an edit, a decision that changes the next step,
  or a validation that changes what is Known.
- Conversation alone Changes no durable State.
  Do not Write a checkpoint for talk.
- Replace `.handoff.md`; one repository Holds one current thread.
- Keep the Checkpoint small enough to rewrite often:
  Intent, Done, Open, State and Next.
- The Checkpoint Records the active Thread, never the whole conversation.
  Name unrelated dirty Paths only to exclude them.
- A Checkpoint is a Snapshot, not a session closing.
  The closing Handoff Adds Growth, parts, affected files and validation.
- Writing the checkpoint does not Trigger another checkpoint.
- Never Store secrets, tokens, terminal history or full conversation text.

This rule is *Provisional*.  
It came from one practice and has not yet Survived a second.
