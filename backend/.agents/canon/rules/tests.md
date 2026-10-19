# Tests

> Provisional. Written from practice, not yet Weathered.

- Spell the Expectation; never read it from the code under test.  
  A Test that Computes what it Checks  
  Agrees with itself and proves nothing.

```go
// Both sides Move together. The mutation passes.
if reply.Status != faults.ReadFaultStatus(want) {

// The table Says the number. The mutation fails.
if reply.Status != status {
```

- Prove it by Mutation.  
  Break the Declaration and run the test.  
  A Test that still Passes was never Watching.
- A test Name is a Use Case, not a method name.  
  *someone Enrols below the Enrolment Age*  
  Beats *testAddStudentInvalidAge*.
- Arrive the Way a caller arrives.  
  Cross the Route, not the private helper.
- One Fake Stands in for every Vendor.  
  A test that needs a port has Found a missing provider.
- Keep one real Test per guarantee only a real thing can give.  
  A unique index is One. A clock is another.
- The Fake obeys the same Rules as the Store.  
  A fake that accepts what the store refuses Hides the case.

Count the Declarations, then count the tests that reach them.  
The two Numbers Match, or the gap has a name.
