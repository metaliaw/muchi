---
name: one-two-refactor
description: Apply the OneTwoThree manifesto's code conventions when writing or refactoring code and its configuration — structure, naming, seams, comments, named values, Provider and Layer boundaries, wire/business/storage Shapes, controlled Failures, constants, tests, build scripts, and shared vendor integration. Use whenever generating or refactoring code in a project that follows Rules from cangrejometralleta/OneTwoThree, or when the user asks for "the manifesto rules". To hear the pattern under a design or a change, send Dove instead — it reads the explanation and names the shape it keeps making.
---

# OneTwoRefactor

A portable summary of [Rules](../../canon/RULES.md).
The Canon Lives in [rules/](../../canon/rules) — each section its own file.
If a Rules file exists in the current repo, it is Canonical — this
skill is the checklist, not a replacement.

This Skill Rides along while you write.
To hear the Pattern under a design or a change, send the
[Dove](../../agents/dove.md) agent —
it reads the explanation and names the shape it keeps making.

## Before returning code, check

Three Groups. The Line, the Boundary, the Project.

### The Line

1. **Structure** — Aim for three-beat Functions: Receive, Transform,
   Return. A beat is one Thought, not one newline — explicit error
   checks don't count against the three. More beats Signal a missing
   abstraction; extract a helper instead of padding one function.

2. **Naming** — `Verb + Noun + context`, three words at most. A name
   past three words means the responsibility is Unclear. A variable
   living inside three lines can Drop to one word — the scope says the
   rest. A Construct Names the Responsibility, never the vendor:
   `store`, not `storegorm`. A Filename may Name the Guest.

3. **Seams** — Break long lines at a real grammatical Joint: `&&`,
   `||`, a comma in a list, a dot in a chain. Never break inside a
   Unit that reads as one. If a boolean expression needs a break, name
   its Parts as local variables instead of splitting mid-expression.

4. **Comments** — Stop at the Claim. A second clause must Add a
   constraint, never repeat the first in other words. Test by
   Deletion: cut the tail and read the head alone. At most one Emoji,
   only in output or comments, never in an identifier.

5. **Values** — A Number with a meaning Carries a Name. Read the
   Literal alone, out of its line; if it cannot say what it means, it
   wants a name. Look for an existing Constant first — the standard
   library, then the framework, then your own. A constant in the core
   must not Drag a vendor in.

### The Boundary

6. **Providers** — Any Interface to something outside the core is
   named after the business need it fills, never the vendor behind it:
   `StudentStore`, not `GormRepository`. A port must never Leak a
   vendor type, error, or import outside its own file.

7. **Layers** — The Core names no Vendor and no Socket. Let the
   compiler hold the Boundary: a package the core cannot import beats
   a rule the core agrees to follow. The shapes the layers speak in
   Import nothing. Two places Hold every vendor: the store and the
   server.

8. **Shapes** — The Entity is never the DTO. Three shapes Carry one
   record: the wire (untrusted, weak types), the business (trusted,
   named types), the storage. Bind the wire Shape, never the entity.
   The Identity comes from the Path or the Store, never from the body.

9. **Failures** — A failure the program expected Carries its own
   answer. Declare the Answer beside the reason, once, and name the
   failure by the case: `ErrRutTaken`, not `Conflict`. A Failure
   carrying no answer was never controlled, and answers five hundred.
   One Function turns a Failure into a Number, and the program holds
   exactly one.

10. **Script / Handler** — Entry points Speak business language only —
    no driver, query, or socket names. A handler Answers with a value
    or it fails; it builds no reply and names no status. The route
    Declares the happy status. Read it aloud; if it stops sounding
    like a sentence, an abstraction is Missing.

### The Project

11. **Constants** — Sort the Value before you place it. Six kinds in
    three Pairs: what the code means (global and algorithmic constants),
    what the deployment chooses (environment configuration and deployment
    topology), what comes from outside (domain data and secrets).
    Global constants Live in `constants/` and never enter the override
    chain; algorithmic constants stay beside their logic. Validate at
    Startup, select the environment explicitly, apply defaults →
    environment file → declared variables, and inject secrets separately.

12. **The Declared Surface** — `.env.example` is the Contract and it is
    committed: required variables live, optional ones commented beside
    the default they replace. `.env` is Local and ignored. Precedence
    Runs one way — the argument, then the environment, then the default —
    so an exported variable wins over a file.

13. **Tests** — Spell the Expectation; never read it from the code
    under test. A test that computes what it checks Agrees with itself
    and proves nothing. Prove it by Mutation: break the declaration
    and watch the test fail. A test name is a Use Case, not a method
    name. Arrive the Way a caller arrives.

14. **Scripts** — Every Program Answers `build.sh` and `run.sh`. Build
    Refuses to build what does not pass; run refuses to start what
    will fail at startup; run reads `.env` when present. One function
    per step, named by what it Checks, with the calls at the bottom
    one per line.

15. **Vendor Integration** — Keep shared Instructions in one source.
    Use relative Links for supported vendor entrances and small adapters
    for required formats. Generate unsupported Copies from that source;
    never maintain them by hand.

**Anti-patterns to flag** — more than three responsibilities in one
unit; a name with no verb; a function with no clear return.

## Example

```go
// BuildOrderReceipt Reads as three sections: total, lines, result.
func BuildOrderReceipt(id string, items []Item, percent int) string {
	total := SumItemPrices(items)
	total = ApplyMemberRate(total, percent)

	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, FormatItemLine(it))
	}

	lines = append(lines, ReportOrderState(id, total, nil))
	return strings.Join(lines, "\n")
}
```

Three names, three beats, one return — `SumItemPrices` and
`ApplyMemberRate` are Providers of a calculation, `ReportOrderState`
is the script's own narration.

The boundary checks Read the same way. A Handler Answers or Fails,
and the route says what a success costs:

```go
// ShowStudentRecord Tells the story of one student.
func (a SchoolAPI) ShowStudentRecord(req transport.Request) (any, error) {
	id, err := app.ReadPathNumber(req)
	if err != nil {
		return nil, err
	}

	student, err := a.Students.SelectStudentRow(school.StudentID(id))
	if err != nil {
		return nil, err
	}

	return school.RenderStudentView(student), nil
}

{Method: "GET", Pattern: "/students/{id}", Handle: a.Guarded(http.StatusOK, a.ShowStudentRecord)},
```

No Status inside the handler, no driver, no reply built by hand.
The Fault Carries its own Answer, and one function reads it.

## Sources

Canon Lives in [rules/](../../canon/rules). A rule marked *(Provisional)*
Came from one practice and has not yet survived a second.

**The Line** —
[Structure](../../canon/rules/structure.md) ·
[Naming](../../canon/rules/naming.md) ·
[Seams](../../canon/rules/seams.md) ·
[Comments](../../canon/rules/comments.md) ·
[Values](../../canon/rules/values.md) *(Provisional)* ·
[Emoji](../../canon/rules/emoji.md)

**The Boundary** —
[Providers](../../canon/rules/providers.md) ·
[Layers](../../canon/rules/layers.md) *(Provisional)* ·
[Shapes](../../canon/rules/shapes.md) ·
[Failures](../../canon/rules/failures.md) *(Provisional)* ·
[Script](../../canon/rules/script.md) ·
[Anti-Patterns](../../canon/rules/anti-patterns.md)

**The Project** —
[Constants](../../canon/rules/constants.md) ·
[Tests](../../canon/rules/tests.md) *(Provisional)* ·
[Scripts](../../canon/rules/scripts.md) *(Provisional)* ·
[Entrypoints](../../canon/rules/entrypoints.md) *(Provisional)* ·
[Vendor Integration](../../canon/rules/vendor-integration.md) ·
[Canonignore](../../canon/rules/canonignore.md) *(Provisional)*

A worked Example Lives in [examples/school](https://github.com/cangrejometralleta/OneTwoThree/blob/93d28555f49e1f2968565b358f320e113bb50a5d/examples/school),
the same service in Go, TypeScript and Java, with
[the Before](https://github.com/cangrejometralleta/OneTwoThree/blob/93d28555f49e1f2968565b358f320e113bb50a5d/examples/school/BEFORE.md) reading the original.
