package offer

import (
	"regexp"
	"strconv"
)

// declaredUnits Matches the Way a Storefront Writes what it has Left, in the
// Sentence a Buyer Reads: "3 disponibles", "1 unidad", "quedan solo 4 unidades
// en stock". It is the same Number that Caps the Quantity Box on that Page.
var declaredUnits = regexp.MustCompile(`(?i)(\d{1,4})\s*(?:unidades?|disponibles?|en stock)`)

// CountDeclaredUnits Reads how many Units a Page Declares, or nil when it Stays
// quiet. Most Storefronts never Say a Number, and Guessing one would Turn a
// Silence into a Promise. The First Match Wins: it Sits next to the Quantity
// Box, before the Footer and its Unrelated Digits.
func CountDeclaredUnits(data []byte) *int {
	match := declaredUnits.FindSubmatch(data)
	if match == nil {
		return nil
	}
	units, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return nil
	}
	return &units
}
