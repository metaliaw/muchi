# Code

Go, because the rules above Read better when they run.

```go
import (
	"fmt"
	"strings"
)

// Item is one Line of an order.
type Item struct {
	Name  string
	Price int
	Qty   int
}

// SumItemPrices Adds every line into a total.
func SumItemPrices(items []Item) int {
	total := 0
	for _, it := range items {
		total += it.Price * it.Qty
	}
	return total
}

// FormatItemLine Renders one item for the receipt.
func FormatItemLine(it Item) string {
	return fmt.Sprintf("%-12s x%d %6d", it.Name, it.Qty, it.Price*it.Qty)
}

// ApplyMemberRate Lowers a total by a percentage.
func ApplyMemberRate(total, percent int) int {
	return total - total*percent/100
}

// ReportOrderState says how it Went, at a glance.
func ReportOrderState(id string, total int, err error) string {
	if err != nil {
		return fmt.Sprintf("❌ Order %s Failed: %v", id, err)
	}
	return fmt.Sprintf("✅ Order %s Closed at %d", id, total)
}

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

- Every Name Follows **Verb + Noun + context**.
- Every Function Owns one Concern and returns it.
- The Emoji Lives in Output, never in a name.
- Comments Follow DeLaCase too.
- BuildOrderReceipt Spends eight lines  
  on three beats.
- Every name here Counts five syllables.

Two whole Services Live in [examples/school](../examples/school),
one in Go and one in TypeScript.
Six Frameworks Serve them and return identical answers.

[The Before](../examples/school/BEFORE.md) Reads the original beside them.
Every rule there was Broken, and each break names the rule it earned.

- A Handler there names no Driver and no Query.
- Read one out loud and it is still a Sentence.
- Two Files Hold every vendor Import.
- A Handler there spends Lines on Errors  
  and still counts three beats.
- The Code Obeyed this Rule  
  before the rule was written down.

*Talk is cheap. Show me the code.*
Torvalds Answered a proposal that shipped no patch.
A Manifesto Runs the same Risk, and this is the answer to it:
every rule that governs code Runs in [examples](../examples).
[Show me the Code](../patterns/show-me-the-code.md) Says why that matters.
