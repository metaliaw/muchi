# Scripts

> Provisional. Written from practice, not yet Weathered.

- Every Program Answers the same two Scripts.  
  `build.sh` Makes the artefact. `run.sh` starts the service.
- Learn them Once and every repository opens the same way.  
  A reader who knows one project can Start the next.
- The Script Enters its own Directory first.  
  It runs from Anywhere, or it runs from one place only.

```sh
cd -- "$(dirname -- "$0")"
```

- One function per step, named by what it Checks.  
  `verify_secret`, `install_packages`, `build_binary`.
- The Calls Live at the Bottom, one per line.  
  Read the last four Lines and you know the script.
- Build Refuses to build what does not pass.  
  Format, then types, then tests, then the Artefact.
- Run Refuses to start what will fail at startup.  
  A missing Secret Costs one line here and a stack trace there.
- Run Reads `.env` when present and says nothing when absent.  
  An exported variable Wins; see [Constants](constants.md).
- A Refusal Names what it wanted and shows the line that works.  
  ❌ says what broke; the next Line says what to type.
- The Gate Belongs to the Script, never to the reader's memory.

The Check Costs one Command.  
`ls build.sh run.sh` in every program Directory.  
A missing one is a Step living in someone's head.
