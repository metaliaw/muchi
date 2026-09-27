# Seams

- Break where the Grammar bends.  
  A Sentence Shows its own Joints: a conjunction, a comma, a preposition.
- Never break inside a Unit that reads as one.  
  An Article Holds its Noun.
- Symmetry is not the Cause.  
  A joint near the middle just Happens to land there.
- Among the legal joints, choose by Meaning.  
  A short Line Emphasises.
- Go has the same Joints:  
  && and ||, the comma in a list, the dot in a chain.
- Better than breaking a long expression, name its Parts.  
  A named Condition Documents while it breaks.

```go
// CheckOrderRecord Names each Condition, then reads them together.
// The break is not in the expression, it is in the Vocabulary.
func CheckOrderRecord(o Order) bool {
	identified := o.ID != ""
	assigned := o.MemberID != ""
	filled := len(o.Items) > 0

	return identified && assigned && filled
}
```

Three conditions, three names, one Return.  
The chain in [store_gorm.go](https://github.com/cangrejometralleta/OneTwoThree/blob/51b6d190b0709055177d9b589ca5a28ad066c2af/examples/school/go/store/store_gorm.go)  
Breaks at the dot for the same reason.
