package offer

import "testing"

func TestNormalizeOffers(t *testing.T) {
	items := []Offer{
		{Store: "one", URL: "https://one.test/card", VariantID: "a"},
		{Store: "one", URL: "https://one.test/card", VariantID: "a"},
	}
	if got := NormalizeCard("  Sol   RING "); got != "sol ring" {
		t.Fatalf("NormalizeCard() = %q", got)
	}
	if got := DeduplicateOffers(items); len(got) != 1 {
		t.Fatalf("DeduplicateOffers() returned %d items", len(got))
	}
}

func TestSelectOffers(t *testing.T) {
	items := []Offer{
		{ID: "cheap", PriceAmount: "1.00", PriceCurrency: "USD"},
		{ID: "middle", PriceAmount: "10.00", PriceCurrency: "USD"},
		{ID: "high", PriceAmount: "11.00", PriceCurrency: "USD"},
	}
	items = MarkSuspicious(items, 30)
	if !items[0].Suspicious || items[0].SuspiciousReason == "" {
		t.Fatal("cheap offer was not marked suspicious")
	}
	selected := SelectStockOffers(items, 5)
	if len(selected) != 2 || selected[0].ID != "middle" {
		t.Fatalf("SelectStockOffers() = %#v", selected)
	}
}

func TestRejectOffer(t *testing.T) {
	valid := Offer{Store: "one", Source: "catalog", PriceAmount: "10.00", URL: "https://one.test/card"}
	if err := ValidateOffer(valid); err != nil {
		t.Fatalf("ValidateOffer() error = %v", err)
	}
	valid.URL = "/relative"
	if err := ValidateOffer(valid); err == nil {
		t.Fatal("ValidateOffer() accepted relative URL")
	}
}

func TestMatchCardTitle(t *testing.T) {
	for _, test := range []struct {
		title string
		name  string
		want  bool
	}{
		{"Sol Ring", "Sol Ring", true},
		{"sol ring", "Sol Ring", true},
		{"Sol Ring (Commander 2019)", "Sol Ring", true},
		{"Kuriboh [LDS3-EN100]", "Kuriboh", true},
		{"Dark Magician | MAGO-EN001", "Dark Magician", true},
		{`MAMO-EN002 "Kuriboh - Multiply!" Ultra Rare`, "Kuriboh - Multiply!", true},
		{"RA05-EN083 “Dark Magician” (Stamp Artwork) Starlight Rare", "Dark Magician", true},
		{"MAMO-SP001 “Dark Magician, the Pharaoh’s Servant” Ultra Rare (Español)", "Dark Magician", false},
		{"LDS3-EN100 “Winged Kuriboh” Common Effect Monster", "Kuriboh", false},
		{"MZMU-EN050 “Darkuriboh” Super Rare", "Kuriboh", false},
		{"Sol Ring Token", "Sol Ring", false},
		{"Solar Ring", "Sol Ring", false},
		{"Sol Ring", "", false},
	} {
		if got := MatchesCard(test.title, test.name); got != test.want {
			t.Errorf("MatchesCard(%q, %q) = %t", test.title, test.name, got)
		}
	}
}

// TestMatchDashTail Separates a Collector Code from another Card's Name.
func TestMatchDashTail(t *testing.T) {
	for _, test := range []struct {
		title string
		name  string
		want  bool
	}{
		{"Mewtwo - 052", "Mewtwo", true},
		{"Mewtwo - SM214", "Mewtwo", true},
		{"Snorlax - SWSH068 (Prerelease)", "Snorlax", true},
		{"Gengar - 60/162 (XY BREAKthrough)", "Gengar", true},
		{"Sol Ring - 212 - uncommon", "Sol Ring", true},
		{"Kuriboh - Multiply!", "Kuriboh", false},
		{"Kuriboh - Multiply! (Starlight Rare) (Extended Art)", "Kuriboh", false},
		{"Kuriboh - ", "Kuriboh", false},
	} {
		if got := MatchesCard(test.title, test.name); got != test.want {
			t.Errorf("MatchesCard(%q, %q) = %t", test.title, test.name, got)
		}
	}
}

// TestReadMatchMode Defaults to the narrow Mode and Refuses an unknown one.
func TestReadMatchMode(t *testing.T) {
	for _, test := range []struct {
		value string
		want  MatchMode
		fails bool
	}{
		{"", MatchExact, false},
		{"exact", MatchExact, false},
		{"includes", MatchIncludes, false},
		{"contains", "", true},
		{"EXACT", "", true},
	} {
		got, err := ReadMatchMode(test.value)
		if (err != nil) != test.fails || got != test.want {
			t.Errorf("ReadMatchMode(%q) = %q, %v", test.value, got, err)
		}
	}
}

// TestQueryWidensWithMode Reads the same Titles under both Modes.
func TestQueryWidensWithMode(t *testing.T) {
	titles := []string{"Kuriboh", "Kuriboh (C)", "Winged Kuriboh", "Linkuriboh", "Kuriboh - Multiply!", "Token: Kuriboh", "Sol Ring"}
	exact := []string{}
	includes := []string{}
	for _, title := range titles {
		if (CardQuery{Name: "Kuriboh", Match: MatchExact}).AcceptsTitle(title) {
			exact = append(exact, title)
		}
		if (CardQuery{Name: "Kuriboh", Match: MatchIncludes}).AcceptsTitle(title) {
			includes = append(includes, title)
		}
	}
	if len(exact) != 2 || exact[0] != "Kuriboh" || exact[1] != "Kuriboh (C)" {
		t.Errorf("exact kept %v", exact)
	}
	if len(includes) != 6 || includes[5] != "Token: Kuriboh" {
		t.Errorf("includes kept %v", includes)
	}
}

// TestMissingModeStaysNarrow Covers a Query Built without naming a Mode.
func TestMissingModeStaysNarrow(t *testing.T) {
	if (CardQuery{Name: "Kuriboh"}).AcceptsTitle("Winged Kuriboh") {
		t.Error("an unnamed mode widened the match")
	}
}
