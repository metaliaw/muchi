# Entrypoints

> Provisional. Written from practice, not yet Weathered.

- A program Declares its entry points, or it has none a reader can trust.  
  What you enter through must be Evident before you ask.
- Evident Means named in the root, at the top, in one list.  
  A newcomer reads that list and Runs the program.
- An entry point nobody listed is a Door someone remembers.  
  Memory is not a Door.
- See [Scripts](scripts.md) for what the two Doors do.  
  This rule says how the Doors are declared and multiplied.

## One Logic, one Shim per Platform

- A shim is the thin File a platform knows how to open,  
  and it does nothing but Call the logic that lives elsewhere.
- It Exists because the platform demands a `.cmd`, not a `.sh`.  
  The demand is about the Extension, never about the work.
- Name, call, exit code. A fourth line is the shim starting to Think,  
  and a shim that thinks is a Wrapper, which is a second program.
- The Logic Lives once, in the language it thinks best in.  
  Every other Platform Calls it. None of them translates it.
- A Script translated is a second script, and the second one lies.  
  Both start identical and Part at the first fix only one received.
- A Shim cannot Diverge, because a shim decides nothing.
- One Helper Finds the Interpreter, for every shim.  
  Five copies of that search go Stale on the first new path.
- The Pattern Asks for an Interpreter already installed over there.  
  Where you cannot assume one, rewrite, and test for the Divergence.

```cmd
@echo off
rem Deploys the Project. The logic lives in the .sh.
call "%~dp0run-bash.cmd" deploy.sh %*
exit /b %errorlevel%
```

## The Order of the Search

- What you choose is not an interpreter. It is a Toolbox.  
  Prefer the Interpreter that sees the tools the script names.
- First the one that Shares the system PATH.
- Then one from the PATH, launchers Discarded.
- Then the foreign environment, path translated and Said out loud.
- Then nothing: an Address to install from, and a non-zero code.
- An interpreter that runs but cannot find its tools is Worse than none.  
  It Fails further in.

## The Invariants

- The Shim Propagates the exit Code.  
  Without it, a Windows CI passes Always.
- The Shim Resolves against itself with `%~dp0`, never the current directory.
- The line Endings Live in `.gitattributes`, not in each machine's editor.

  ```
  *.cmd text eol=crlf
  *.sh  text eol=lf
  ```

- The `.cmd` stays ASCII. Another codepage Dirties the accents.
- A Shim is read from here and Proven only over there.  
  Until someone runs it on the platform, say it is Unverified.

The Check Costs two Commands.  
`rg -n 'build\.sh|run\.sh' README.md` Names the doors in the root.  
`wc -l *.cmd` Says no shim passed three lines plus `@echo off`.  
Every `.sh` with a shim beside it Holds the logic alone.
