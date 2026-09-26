# Canonignore

> Provisional. Written from practice, not yet Weathered.

- A Repository Carries more than it teaches.  
  `.canonignore` Says which is which.
- `.gitignore` says what the repository does not Carry.  
  `.canonignore` says what it Carries and does not teach.
- Read every Path in it. Copy none.  
  Ignoring is not hiding; it is withholding Authority.

```text
.gitignore      absent from the Repository
.canonignore    present, and not to be Imitated
```

- Same syntax, so no one Learns a second one.  
  Globs, `#` for a comment, `!` to take one Back.
- Three kinds Belong here, and a fourth belongs nowhere.  
  The material read but never copied, the undistilled note,
  the generated Artefact.  
  Canon that embarrasses you does not go here; it goes Away.
- Name the Reason above the pattern.  
  A Path with no Reason Rots into a path no one dares delete.
- Take back the File that explains the rest.  
  `jokes/**` Ignores every joke and keeps the README that frames them.
- The Entry Earns its Place by being read wrongly once.  
  Add a Path the day an agent copies from it.

```sh
git ls-files --cached --ignored --exclude-from=.canonignore
git ls-files --others --ignored --exclude-from=.canonignore
```

git Reads the file without being told it is new.  
The first Command Lists the Committed; the second, the merely present.  
A Path in neither Answer is Canon, and governs.
