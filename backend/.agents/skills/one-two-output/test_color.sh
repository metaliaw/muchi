#!/usr/bin/env bash
# Run the Color Rules against real Input, in your own Terminal.
#   bash test_color.sh
# Every Case Prints its Verdict, then Shows the Output it Judged.

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COLOR="python3 $HERE/color.py"
PASSED=0
FAILED=0

# CheckCase Runs one Case and Compares the Hues it Painted.
# Receives a Name, the Expected Hue Count and the Input on stdin.
check_case() {
    local name="$1" expected="$2" input="$3"
    local painted
    painted=$(printf '%s\n' "$input" | $COLOR --force | grep -o $'\033\[38;5;[0-9]*m' | sort -u | wc -l)

    if [ "$painted" -eq "$expected" ]; then
        echo "✅ $name — $painted Hues"
        PASSED=$((PASSED + 1))
    else
        echo "❌ $name — Expected $expected Hues, Got $painted"
        FAILED=$((FAILED + 1))
    fi
}

# CheckMarkdown Runs one Case on the Markdown Plane.
# Receives a Name, the Expected Quoted Lines, the Expected Mark Kinds
# and the Input. Counts Blockquote Lines and distinct Marks.
check_markdown() {
    local name="$1" quoted="$2" kinds="$3" input="$4"
    local out lines found=0

    out=$(printf '%s\n' "$input" | $COLOR --md)
    lines=$(printf '%s\n' "$out" | grep -c '^> ')

    # Count only the Marks the Script Added. A Mark the Page already
    # Spends Appears in both, so it Proves nothing about Grouping.
    for pattern in '\*\*' '(^|[^*])\*[^*]' '`'; do
        local before after
        before=$(printf '%s\n' "$input" | grep -oE "$pattern" | wc -l)
        after=$(printf '%s\n' "$out" | grep -oE "$pattern" | wc -l)
        [ "$after" -gt "$before" ] && found=$((found + 1))
    done

    if [ "$lines" -eq "$quoted" ] && [ "$found" -eq "$kinds" ]; then
        echo "✅ $name — $lines Quoted, $found Marks"
        PASSED=$((PASSED + 1))
    else
        echo "❌ $name — Expected $quoted Quoted and $kinds Marks, Got $lines and $found"
        FAILED=$((FAILED + 1))
    fi
}

echo "── Rules ──"

check_case "Three Groups take three Hues" 3 \
'de-la-case Carries one-two-refactor
one-two-refactor has no Slug
~/.claude/skills/de-la-case -> .agents/skills/one-two-refactor
one-two-output is fine'

check_case "The fourth Group Stays bare" 3 \
'alpha.go alpha.go
beta.go beta.go
gamma.go gamma.go
delta.go delta.go'

check_case "Background on every Line takes none" 0 \
'run-1 on host-a
run-2 on host-a
run-3 on host-a
run-4 on host-a
run-5 on host-a'

check_case "A short Block Keeps its shared Token" 1 \
'run-1 on host-a
run-2 on host-a
run-3 on host-a'

check_case "One Line has no Background" 1 \
'vendor.go Broke, vendor.go Broke again'

# The Prefix must not Cover every Line, or Rule five Drops it first.
check_case "One Line Set Shares one Hue" 1 \
'a6c4867 Add Color by Co-occurrence
.agents/skills/out/SKILL.md
.agents/skills/out/color.md
.agents/skills/out/color.py
3 files changed'

check_case "A Singleton takes no Hue" 0 \
'alpha.go Passed
beta.go Failed'

check_case "Prose is Grammar, never a Referent" 1 \
'✅ Build Closed in 18s
142 Files, 3 ⚠️ in vendor.go
Build still Pending, vendor.go untouched'

echo
echo "── Markdown ──"

check_markdown "A Run takes a Mark, never a Move" 0 1 \
'alpha.go Passed
alpha.go Closed
beta.go Waiting'

check_markdown "A Gap Changes nothing, the Mark Holds" 0 1 \
'alpha.go Passed
beta.go Waiting
alpha.go Closed'

check_markdown "Four Groups, three Marks, the fourth bare" 0 3 \
'api-gateway Answered on eu-west-1
api-gateway Answered on us-east-1
worker-pool Drained on eu-west-1
worker-pool Drained on us-east-1
billing-db Migrated
eu-west-1 Lagged behind us-east-1'

check_markdown "A Page that Bolds Groups with two" 0 2 \
'**Report**
a.go and x
b.go and y
c.go and z
a.go b.go c.go'

check_markdown "A Page that Spends all three Groups with none" 0 0 \
'**Report** on `main` with *stress*
a.go and x
b.go and y
a.go b.go'

check_markdown "No Group, no Mark and no Move" 0 0 \
'alpha.go Passed
beta.go Failed'

echo
echo "── Streams ──"

if [ -z "$(printf 'a.go a.go\n' | $COLOR)" ] || printf 'a.go a.go\n' | $COLOR | grep -q $'\033'; then
    echo "❌ A Pipe Ate an Escape"
    FAILED=$((FAILED + 1))
else
    echo "✅ A Pipe Gets clean Bytes"
    PASSED=$((PASSED + 1))
fi

if printf 'a.go a.go\n' | NO_COLOR=1 $COLOR --force | grep -q $'\033'; then
    echo "❌ NO_COLOR Lost to --force"
    FAILED=$((FAILED + 1))
else
    echo "✅ NO_COLOR Beats --force"
    PASSED=$((PASSED + 1))
fi

if [ -z "$(printf '' | $COLOR --force)" ]; then
    echo "✅ Empty Input Returns Empty"
    PASSED=$((PASSED + 1))
else
    echo "❌ Empty Input Invented Output"
    FAILED=$((FAILED + 1))
fi

echo
echo "── Look at it ──"
echo
printf '%s\n' \
'✅ Deploy Closed in 41s' \
'api-gateway Answered on eu-west-1, api-gateway Answered on us-east-1' \
'worker-pool Drained on eu-west-1, worker-pool Drained on us-east-1' \
'billing-db Migrated, 12 Tables Touched, shape_test.go:41 still Red' \
'cache-warm Skipped, no Upstream Change since v2.4.0' \
'⚠️ eu-west-1 Lagged 300ms behind us-east-1' | $COLOR --force

echo
echo "eu-west-1 and us-east-1 Share every Line, so they Share a Hue."
echo "api-gateway and worker-pool take the other two."
echo "Everything Named once Stays plain."
echo
if [ "$FAILED" -eq 0 ]; then
    echo "✅ $PASSED Passed, none Failed"
else
    echo "❌ $FAILED Failed, $PASSED Passed"
fi
exit $((FAILED > 0))
