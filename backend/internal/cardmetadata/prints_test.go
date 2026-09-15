package cardmetadata

import "testing"

func solRingPrints() []Print {
	return []Print{
		{Edition: "c19", EditionName: "Commander 2019", CollectorNumber: "221", Image: "https://images.test/c19-221.jpg"},
		{Edition: "c14", EditionName: "Commander 2014", CollectorNumber: "270", Image: "https://images.test/c14-270.jpg"},
		{Edition: "mkc", EditionName: "Murders at Karlov Manor Commander", CollectorNumber: "237", Image: "https://images.test/mkc-237.jpg"},
		{Edition: "znc", EditionName: "Zendikar Rising Commander", CollectorNumber: "72", Image: "https://images.test/znc-72.jpg"},
		{Edition: "pip", EditionName: "Fallout", CollectorNumber: "123", Image: "https://images.test/pip-123.jpg"},
		{Edition: "c21", CollectorNumber: "263", Image: "https://images.test/c21-263.jpg"},
		{Edition: "c21", CollectorNumber: "264", Image: "https://images.test/c21-264.jpg"},
		{Edition: "soc", CollectorNumber: "1", Image: "https://images.test/soc-1.jpg"},
	}
}

// TestImageForEveryStoreTitleShape Reads the Shapes real Stores Publish.
// Guessing the Shape Needs one Rule per Store; Asking which Word is an Edition
// Scryfall Listed for this Card Needs none, and never Invents one.
func TestImageForEveryStoreTitleShape(t *testing.T) {
	index := IndexPrints(solRingPrints())
	for _, test := range []struct{ name, title, want string }{
		{"code and hash number", "Sol Ring [C19] #221 ENG", "https://images.test/c19-221.jpg"},
		{"code and number in parentheses", "Sol Ring — Commander 2019 (C19 #221) · NM · EN", "https://images.test/c19-221.jpg"},
		{"code joined by a dash", "Sol Ring [C14-270] [Near Mint / Inglés]", "https://images.test/c14-270.jpg"},
		{"code spaced from its number", "Sol Ring [MKC - 237] [Moderately Played]", "https://images.test/mkc-237.jpg"},
		{"code among pipes", "Sol Ring | Inglés | NM | MKC", "https://images.test/mkc-237.jpg"},
		{"code buried in words", "Sol Ring [ZNC Normal Inglés NM Normal]", "https://images.test/znc-72.jpg"},
		{"edition named in words", "Sol Ring [Fallout]", "https://images.test/pip-123.jpg"},
		{"exact printing", "Sol Ring [C21] #263 ENG", "https://images.test/c21-263.jpg"},
		{"edition without a number answers with its first print", "Sol Ring [C21] ENG", "https://images.test/c21-263.jpg"},
		{"edition of a single print", "Sol Ring [SOC] ENG", "https://images.test/soc-1.jpg"},
		{"no edition at all", "Sol Ring — Near Mint", ""},
		{"condition only", "Sol Ring [Idioma: Español, Estado: NM]", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := index.ImageFor(test.title, "Sol Ring"); got != test.want {
				t.Fatalf("ImageFor(%q) = %q, want %q", test.title, got, test.want)
			}
		})
	}
}

// TestEmptyIndexSaysSo Lets a Caller Skip the Loop when nothing can Answer.
func TestEmptyIndexSaysSo(t *testing.T) {
	if !IndexPrints(nil).Empty() {
		t.Fatal("an index of nothing did not say it was empty")
	}
	if !IndexPrints([]Print{{Edition: "c21"}}).Empty() {
		t.Fatal("a print without a number or an image counts as usable")
	}
	if IndexPrints(solRingPrints()).Empty() {
		t.Fatal("a usable index said it was empty")
	}
}
