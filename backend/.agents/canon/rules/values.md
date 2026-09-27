# Values

> Provisional. Written from practice, not yet Weathered.

- A Number with a meaning Carries a Name.  
  The Index Counts; the Name Explains.
- `400` says where it Fell in a list someone else wrote.  
  `HTTP_BAD_REQUEST` says what Happened.
- Look for the Name before you write one.  
  A Vendor already Named it, almost always.

```java
// The literal Costs a reader one lookup, every time.
return new Fault(409, reason);

// The JDK already Named it.
return new Fault(HTTP_CONFLICT, reason);
```

- Take the Name from the narrowest source that owns it.  
  The standard library First, the framework second,  
  and your own constant only when neither knows.
- A Constant in the Core must not drag a Vendor in.  
  `java.net.HttpURLConnection` Costs nothing.  
  `org.springframework.http.HttpStatus` Costs the boundary.
- The exceptions are the Numbers that mean themselves.  
  Zero, one, the index in a Loop.  
  A weight table Keeps its digits and names the table.
- Two Literals with one Meaning are one missing Name.  
  Find the second occurrence and the name Writes itself.

The Test is Mechanical.  
Read the Literal alone, out of its line.  
If it cannot say what it means, it Wants a name.
