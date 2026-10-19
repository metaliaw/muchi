# Failures

> Provisional. Written from practice, not yet Weathered.

- A Failure the program expected Carries its own Answer.  
  Declare the Answer beside the reason, once.
- One Type Holds both Halves.  
  The Reason is for the Reader; the Answer is for the Caller.
- Name the Failure by the case, never by the number.  
  *Rut Taken* is a business Fact.  
  *Conflict* is only how it ends.

```go
ErrRutTaken = faults.ReportTakenValue("rut is already Registered")
```

- A Failure carrying no Answer was never Controlled.  
  It Answers five hundred, because it is ours.
- Never guess on the Caller's behalf.  
  A driver error reaching the edge as four hundred  
  Blames the caller for our own break.
- One Function turns a Failure into a Number,  
  and the whole program holds exactly one.
- A constructor per kind Reads better than a field.  
  *RefuseInvalidInput* Names the move; `status: 400` only sets it.
- The Guard Answers one Way and explains nothing.  
  A caller who cannot prove itself Learns only that.

Grep the Codebase for the mapping.  
One hit is the Rule working.  
Two hits is a Decision that drifted.
