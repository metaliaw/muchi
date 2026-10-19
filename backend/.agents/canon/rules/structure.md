# Structure

- Three-line functions  
  are the ideal size Target.
- Three Lines means three Beats, not three newlines.  
  A beat is one Thought:  
  receive, transform and return.
- A language with explicit errors Spends newlines.  
  Count the Thoughts instead.
- One Unit Owns one Concern.  
  A Function Does one Thing.
- More Lines signal a missing Abstraction Layer.
- When the Body Earns more Lines,  
  group them into Three,  
  one blank line between each section.
- Three Sections Read like three Lines.  
  The Rhythm Survives.

```go
// ListStudentRecords Spends eleven lines on three beats.
// Receive, transform, return.
// The errors Cost lines, they never cost thoughts.
func (a SchoolAPI) ListStudentRecords(req Request) Response {
	page, err := ReadPageRequest(req)
	if err != nil {
		return BuildFailureReply(err)
	}

	students, err := a.Students.SelectStudentPage(page)
	if err != nil {
		return BuildFailureReply(err)
	}

	return Response{http.StatusOK, RenderStudentViews(students)}
}
```

Count the beats and you get Three.  
Count the newlines and you get Eleven.  
Only one of those numbers Means anything.
