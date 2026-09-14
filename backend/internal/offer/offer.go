package offer

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

var ErrInvalidOffer = errors.New("invalid offer")

type Offer struct {
	ID               string            `json:"id"`
	CardName         string            `json:"card_name"`
	Store            string            `json:"store"`
	PriceAmount      string            `json:"price_amount"`
	PriceCurrency    string            `json:"price_currency"`
	URL              string            `json:"url"`
	Image            string            `json:"image,omitempty"`
	VariantID        string            `json:"variant_id,omitempty"`
	Language         string            `json:"language,omitempty"`
	Condition        string            `json:"condition,omitempty"`
	Finish           string            `json:"finish,omitempty"`
	Source           string            `json:"source"`
	StockStatus      string            `json:"stock_status"`
	Suspicious       bool              `json:"suspicious"`
	SuspiciousReason string            `json:"suspicious_reason,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func NormalizeCard(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// editionBrackets Open a Tail that cannot be Part of a Card Name.
var editionBrackets = []string{" | ", " (", " ["}

// editionDashes Open a Tail that Must Look like a Code, because a Dash also
// Lives inside Card Names such as "Kuriboh - Multiply!".
var editionDashes = []string{" - ", " \u2013 ", " \u2014 "}

// quotePairs Wrap a Card Name a Store Leads with something else, such as a Set Code.
var quotePairs = [][2]string{{"\u201c", "\u201d"}, {`"`, `"`}, {"\u00ab", "\u00bb"}}

// MatchMode Says how far a Title may Stray from the Card Name.
type MatchMode string

const (
	// MatchExact Keeps the Card and its Printings: `Kuriboh`, `Kuriboh (C)`.
	MatchExact MatchMode = "exact"
	// MatchIncludes Keeps any Title Carrying the Name: `Winged Kuriboh`, `Linkuriboh`.
	MatchIncludes MatchMode = "includes"
)

// ErrInvalidMatch Answers a Match Mode the Catalog does not Offer.
var ErrInvalidMatch = errors.New("invalid match mode")

// CardQuery Names the Card to Look for and how Wide its Title may be.
type CardQuery struct {
	Name  string
	Match MatchMode
}

// AcceptsTitle Answers whether a Store Title Belongs to this Query.
func (q CardQuery) AcceptsTitle(title string) bool {
	if q.Match == MatchIncludes {
		return ContainsCard(title, q.Name)
	}
	return MatchesCard(title, q.Name)
}

// ReadMatchMode Reads the Mode a Caller Asked for, Defaulting to the narrow one.
func ReadMatchMode(value string) (MatchMode, error) {
	switch MatchMode(value) {
	case "", MatchExact:
		return MatchExact, nil
	case MatchIncludes:
		return MatchIncludes, nil
	}
	return "", ErrInvalidMatch
}

// ContainsCard Accepts a Title that Carries the Card Name anywhere inside it.
func ContainsCard(title, name string) bool {
	name = NormalizeCard(name)
	return name != "" && strings.Contains(NormalizeCard(title), name)
}

// MatchesCard Accepts a Title that Names the Card, Wherever the Store Puts it.
func MatchesCard(title, name string) bool {
	title = NormalizeCard(title)
	name = NormalizeCard(name)
	if name == "" {
		return false
	}
	if title == name {
		return true
	}
	for _, bracket := range editionBrackets {
		if strings.HasPrefix(title, name+bracket) {
			return true
		}
	}
	return followsCode(title, name) || quotesCardName(title, name)
}

// followsCode Accepts a Dash Tail that Opens with a Collector Code, never with a Word.
// `Mewtwo - SM214` is the same Card; `Kuriboh - Multiply!` is another one.
func followsCode(title, name string) bool {
	for _, dash := range editionDashes {
		tail, found := strings.CutPrefix(title, name+dash)
		if !found {
			continue
		}
		fields := strings.Fields(tail)
		if len(fields) > 0 && strings.ContainsAny(fields[0], "0123456789") {
			return true
		}
	}
	return false
}

// quotesCardName Reads the Quoted Form, where the Card Name Sits inside the Title.
func quotesCardName(title, name string) bool {
	for _, pair := range quotePairs {
		if strings.Contains(title, pair[0]+name+pair[1]) {
			return true
		}
	}
	return false
}

func ValidateOffer(item Offer) error {
	link, err := url.ParseRequestURI(item.URL)
	valid := item.Store != "" && item.Source != "" && item.PriceAmount != "" && err == nil
	if !valid || link.Scheme == "" || link.Host == "" {
		return ErrInvalidOffer
	}
	return nil
}

func DeduplicateOffers(items []Offer) []Offer {
	seen := make(map[string]struct{}, len(items))
	result := make([]Offer, 0, len(items))
	for _, item := range items {
		key := strings.Join([]string{item.Store, item.URL, item.VariantID}, "|")
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func SelectStockOffers(items []Offer, limit int) []Offer {
	result := append([]Offer(nil), items...)
	sort.SliceStable(result, func(i, j int) bool {
		return compareDecimal(result[i].PriceAmount, result[j].PriceAmount) < 0
	})
	result = filterTrusted(result)
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

// MarkSuspicious Flags an Offer priced below percent of its Currency Median.
func MarkSuspicious(items []Offer, percent int) []Offer {
	groups := groupOfferPrices(items)
	medians := findPriceMedians(groups)

	return applySuspicious(items, medians, percent)
}

func groupOfferPrices(items []Offer) map[string][]int {
	groups := make(map[string][]int)
	for _, item := range items {
		if cents, err := parseCents(item.PriceAmount); err == nil {
			groups[item.PriceCurrency] = append(groups[item.PriceCurrency], cents)
		}
	}
	return groups
}

func findPriceMedians(groups map[string][]int) map[string]int {
	medians := make(map[string]int, len(groups))
	for currency, prices := range groups {
		sort.Ints(prices)
		medians[currency] = prices[len(prices)/2]
	}
	return medians
}

func applySuspicious(items []Offer, medians map[string]int, percent int) []Offer {
	reason := fmt.Sprintf("price_below_%d_percent_median", percent)
	for index := range items {
		cents, err := parseCents(items[index].PriceAmount)
		median := medians[items[index].PriceCurrency]
		pricedLow := err == nil && median > 0 && cents*100 < median*percent
		if pricedLow {
			items[index].Suspicious = true
			items[index].SuspiciousReason = reason
		}
	}
	return items
}

func filterTrusted(items []Offer) []Offer {
	result := make([]Offer, 0, len(items))
	for _, item := range items {
		if !item.Suspicious {
			result = append(result, item)
		}
	}
	return result
}

func compareDecimal(left, right string) int {
	left = normalizeDecimal(left)
	right = normalizeDecimal(right)
	if len(left) != len(right) {
		if len(left) < len(right) {
			return -1
		}
		return 1
	}
	return strings.Compare(left, right)
}

func normalizeDecimal(value string) string {
	parts := strings.SplitN(value, ".", 2)
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	fraction := "00"
	if len(parts) == 2 {
		fraction = (parts[1] + "00")[:2]
	}
	return whole + fraction
}

func parseCents(value string) (int, error) {
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.Atoi(parts[0])
	if err != nil || whole < 0 {
		return 0, ErrInvalidOffer
	}
	fraction := "00"
	if len(parts) == 2 {
		fraction = (parts[1] + "00")[:2]
	}
	cents, err := strconv.Atoi(fraction)
	if err != nil {
		return 0, ErrInvalidOffer
	}
	return whole*100 + cents, nil
}
