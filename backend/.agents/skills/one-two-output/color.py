#!/usr/bin/env python3
"""Color shaped Output by Co-occurrence. The Rules live in color.md.

A Token that Repeats is a Group of its Occurrences. The first three
such Groups, by Order of Appearance, take the three Hues. The fourth
Stays bare, because the Ceiling makes Color a Limiter.

Reads stdin, Writes stdout. Passes Through untouched when stdout is
not a Terminal or NO_COLOR is Set, so a Pipe never Eats an Escape.
--md Marks for a rendered Document, using only the Marks the Page has
not Spent; --marks=N Caps that further. --force Paints into a Pipe;
--legend Names the Key on stderr. Bare
Arguments Choose the Tokens by Hand, when the Human Sees a Group the
Count missed. NO_COLOR Beats --force: the Reader's standing Preference
Outranks the Writer's Flag.
"""

import os
import re
import sys

# Blue, Magenta, Cyan. Green, Red and Yellow stay Reserved
# for the Emoji Trio — see color.md, Rule seven.
HUES = ("\033[38;5;39m", "\033[38;5;170m", "\033[38;5;80m")
RESET = "\033[0m"

# Markdown Carries three Marks where the Terminal Carries three Hues.
MARKS = (("**", "**"), ("*", "*"), ("`", "`"))

HUE_CEILING = 3
GROUP_FLOOR = 2
# A Slash Separates Tokens; a Dot, Dash, Colon or Underscore Binds them,
# so shape_test.go:41 and v0.1.0-rc.3 Survive whole while a Path Splits.
TOKEN_PATTERN = re.compile(r"[A-Za-z_][A-Za-z0-9_.:-]*[A-Za-z0-9_]|[A-Za-z_]")
PATH_MARKS = "/."
BLANK_LINE = re.compile(r"\n[ \t]*\n")


def name_referent(token, is_path_bound):
    """A Referent Carries a Mark, Sits in a Path, or Runs long and lowercase.

    Connectors never Qualify, so the Color Lands on what the Reader
    would Look up, never on the Grammar between.
    """
    if any(mark in token for mark in "._:-") or any(c.isdigit() for c in token):
        return True
    if is_path_bound:
        return True
    return len(token) >= 8 and token.islower()


def bind_to_path(text, match):
    """A Token Sits in a Path when a Slash Leads it, or a Dot Opens it."""
    before = text[match.start() - 1] if match.start() else ""
    after = text[match.end()] if match.end() < len(text) else ""
    return before in PATH_MARKS or after == "/"


def find_repeated_tokens(text):
    """Collect the Referents that Appear twice or more, in first-seen Order.

    A Token on every Line Groups nothing — it is Background, not a Group,
    so it Falls out even when it Repeats. Color Marks what Gathers some
    Lines, never what Covers them all.
    """
    counts = {}
    lines_seen = {}
    order = []
    lines = [line for line in text.splitlines() if line.strip()]

    for index, line in enumerate(lines):
        for match in TOKEN_PATTERN.finditer(line):
            token = match.group(0)
            if not name_referent(token, bind_to_path(line, match)):
                continue
            if token not in counts:
                order.append(token)
                lines_seen[token] = set()
            counts[token] = counts.get(token, 0) + 1
            lines_seen[token].add(index)

    kept = [
        token
        for token in order
        if counts[token] >= GROUP_FLOOR
        and not covers_every_line(lines_seen[token], len(lines))
    ]
    return kept, lines_seen


def covers_every_line(seen, total):
    """Background Needs a Page to Spread across.

    In a Block no taller than the Ceiling, a Token on every Line is not
    Background — it is the Co-occurrence itself. Only when the Block
    Outgrows the Ceiling does Total Coverage Stop Telling anything.
    """
    return total > HUE_CEILING and len(seen) == total


def assign_hues(tokens, lines_seen):
    """Hand the first three Groups a Hue each. The rest Stay bare.

    Tokens that Touch exactly the same Lines are one Group, not three.
    A shared Prefix Reads as one Fact, so it takes one Hue — and the
    Hues it did not Eat Stay free for the Tokens that really Differ.
    """
    return dict(wrap_by_signature(tokens, lines_seen, HUES))


def group_by_signature(tokens, lines_seen):
    """Gather the Tokens that Touch exactly the same Lines, in first-seen Order."""
    signatures = []
    members = {}
    for token in tokens:
        signature = frozenset(lines_seen.get(token, ()))
        if signature not in members:
            signatures.append(signature)
            members[signature] = []
        members[signature].append(token)

    return [(signature, members[signature]) for signature in signatures]


def wrap_by_signature(tokens, lines_seen, wrappers):
    """Hand each Group the next Wrapper. Past the Ceiling, none."""
    paired = []
    for index, (_, members) in enumerate(group_by_signature(tokens, lines_seen)):
        if index >= len(wrappers):
            break
        paired.extend((token, wrappers[index]) for token in members)

    return paired


def paint_text(text, hues):
    """Wrap every whole-Token Occurrence in its Hue."""
    if not hues:
        return text

    pattern = re.compile(
        r"(?<![A-Za-z0-9_])(" + "|".join(re.escape(t) for t in hues) + r")(?![A-Za-z0-9_])"
    )
    return pattern.sub(lambda m: hues[m.group(1)] + m.group(1) + RESET, text)


def paint_marks(text, marks):
    """Wrap every whole-Token Occurrence in its Markdown Mark."""
    if not marks:
        return text

    pattern = re.compile(
        r"(?<![A-Za-z0-9_])(" + "|".join(re.escape(t) for t in marks) + r")(?![A-Za-z0-9_])"
    )
    return pattern.sub(lambda m: marks[m.group(1)][0] + m.group(1) + marks[m.group(1)][1], text)


def split_paragraphs(text):
    """A blank Line Ends a Group. The Eye Relates within a Paragraph, not across.

    Each Block Counts its own Repetitions and Spends its own Budget,
    so a Header Paragraph Spending Bold Leaves Bold free below it.
    """
    return BLANK_LINE.split(text)


def render_paragraphs(text, render_one):
    """Run one Renderer over each Paragraph and Join them back."""
    return "\n\n".join(render_one(block) for block in split_paragraphs(text))


def find_free_marks(text, cap):
    """Return the Marks the Document has not Spent, up to the Cap.

    A Mark already Carrying Emphasis cannot also Mean Group. The
    Ceiling is what the Surface has Left, never what it Has — so a
    Page that Bolds its Labels Groups with one Mark fewer.
    """
    spent = {
        "**": "**" in text,
        "*": bool(re.search(r"(?<![*\\])\*(?!\*)", text)),
        "`": "`" in text,
    }
    free = [pair for pair in MARKS if not spent[pair[0]]]

    return free[:cap] if cap else free


def render_markdown(text, cap=0):
    """Mark each Paragraph on its own Budget, then Join them back."""
    return render_paragraphs(text, lambda block: mark_paragraph(block, cap))


def mark_paragraph(text, cap=0):
    """Mark the Groups for a Rendered Document, where no Escape Survives.

    Receives the raw Text, Wraps each Group in a Mark the Page has not
    Spent, and Returns the Document. Nothing Moves — see color.md,
    Why the Lines do not Move.
    """
    found, lines_seen = find_repeated_tokens(text)
    marks = dict(wrap_by_signature(found, lines_seen, find_free_marks(text, cap)))

    return paint_marks(text, marks)

def collect_hues(text):
    """Gather every Paragraph's Key into one, for the Legend."""
    hues = {}
    for block in split_paragraphs(text):
        found, lines_seen = find_repeated_tokens(block)
        hues.update(assign_hues(found, lines_seen))

    return hues


def paint_paragraph(text):
    """Hue one Paragraph on its own Count."""
    found, lines_seen = find_repeated_tokens(text)

    return paint_text(text, assign_hues(found, lines_seen))


def write_legend(hues, stream):
    """Name which Token took which Hue, so the Key is never Guessed."""
    for token, hue in hues.items():
        stream.write(f"{hue}{token}{RESET}\n")


def main():
    wants_legend = "--legend" in sys.argv[1:]
    wants_markdown = "--md" in sys.argv[1:]
    cap = next((int(a.split("=")[1]) for a in sys.argv[1:] if a.startswith("--marks=")), 0)
    forces_color = "--force" in sys.argv[1:]
    chosen = [a for a in sys.argv[1:] if not a.startswith("-")]
    text = sys.stdin.read()

    if wants_markdown:
        sys.stdout.write(render_markdown(text, cap))
        return 0

    if os.environ.get("NO_COLOR") or not (forces_color or sys.stdout.isatty()):
        sys.stdout.write(text)
        return 0

    if chosen:
        # A Human Naming the Tokens Needs no Inference: one Hue each.
        hues = dict(zip(chosen[:HUE_CEILING], HUES))
    else:
        found, lines_seen = find_repeated_tokens(text)
        hues = assign_hues(found, lines_seen)
    sys.stdout.write(paint_text(text, hues))

    if wants_legend:
        write_legend(hues, sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
