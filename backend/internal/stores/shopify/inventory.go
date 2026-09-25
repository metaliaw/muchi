package shopify

import (
	"regexp"
	"strconv"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// A Shopify Catalog Publishes whether a Variant Sells, never how Many Remain:
// `/products/<handle>.js` Says `available` and nothing else. The Storefront
// Page Does Say it when the Theme Shows a Counter —the same Number that Caps
// the Quantity Box a Buyer Sees— and that is the only public Place it Lives.

// countedVariant Matches the Theme Map that Names each Variant and its Count:
// `"inventories": {"41743898575021": {… "inventory_quantity": 4 …}}`.
var countedVariant = regexp.MustCompile(
	`"(\d+)"\s*:\s*\{[^{}]*?"inventory_quantity"\s*:\s*(-?\d{1,6})`)

// countedInline Matches the Theme that Writes the Count inside the Variant
// itself: `{"id":41743898575021, … "inventory_quantity":4}`.
var countedInline = regexp.MustCompile(
	`"id"\s*:\s*(\d+)[^{}]*?"inventory_quantity"\s*:\s*(-?\d{1,6})`)

// readVariantUnits Reads how many Units the Page Declares for one Variant, or
// nil when it Stays quiet. A Page that Never Says a Number Keeps its Silence:
// Guessing one would Turn it into a Promise.
func readVariantUnits(data []byte, variant string) *int {
	if units := readCountedVariant(data, variant); units != nil {
		return units
	}
	// A Theme that Writes no Map may still Write the Sentence: "quedan solo 4
	// unidades en stock". The Page was Asked for this Variant, so the Sentence
	// on it is about this Variant, the same one the Buyer Reads.
	return model.CountDeclaredUnits(data)
}

func readCountedVariant(data []byte, variant string) *int {
	for _, pattern := range []*regexp.Regexp{countedVariant, countedInline} {
		for _, match := range pattern.FindAllSubmatch(data, -1) {
			if string(match[1]) != variant {
				continue
			}
			units, err := strconv.Atoi(string(match[2]))
			if err != nil || units < 0 {
				continue
			}
			return &units
		}
	}
	return nil
}
