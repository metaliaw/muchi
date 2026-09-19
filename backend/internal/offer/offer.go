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
	ID            string `json:"id"`
	CardName      string `json:"card_name"`
	Store         string `json:"store"`
	PriceAmount   string `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	URL           string `json:"url"`
	Image         string `json:"image,omitempty"`
	VariantID     string `json:"variant_id,omitempty"`
	Language      string `json:"language,omitempty"`
	Condition     string `json:"condition,omitempty"`
	Finish        string `json:"finish,omitempty"`
	Edition       string `json:"edition,omitempty"`
	// Kind Says whether the Offer is a Single Card or a Sealed Product. A Pack
	// and a Display Share a Set and a Name, and only this Tells them apart from
	// the Card inside them.
	Kind      ProductKind `json:"kind,omitempty"`
	Locations []string    `json:"locations,omitempty"`
	// CardKey Names the Card the Offer is for, with the Printing Dropped.
	// The Caller Groups by it; the Title Stays for Reading.
	CardKey     string `json:"card_key,omitempty"`
	Source      string `json:"source"`
	StockStatus string `json:"stock_status"`
	// StockQuantity Counts the Units the Store Declares. Absent Means the Store
	// Never Said; Zero Means it Said None. A Reader Tells them Apart.
	StockQuantity    *int              `json:"stock_quantity,omitempty"`
	Suspicious       bool              `json:"suspicious"`
	SuspiciousReason string            `json:"suspicious_reason,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// StockReading Carries what one Store Answered about one Offer.
type StockReading struct {
	Status   string `json:"stock_status"`
	Quantity *int   `json:"stock_quantity,omitempty"`
}

// ReadStock Names a Status no Store Counted.
func ReadStock(status string) StockReading { return StockReading{Status: status} }

// CountStock Names a Status the Store Backed with a Number.
func CountStock(status string, quantity int) StockReading {
	return StockReading{Status: status, Quantity: &quantity}
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

// ProductKind Says what the Caller is Shopping for.
type ProductKind string

const (
	// KindSingle Keeps the Cards a Store Sells one by one.
	KindSingle ProductKind = "single"
	// KindSealed Keeps the Boxes a Store Sells Unopened: Booster Box, Elite
	// Trainer Box, Bundle, Display.
	KindSealed ProductKind = "sealed"
)

// ErrInvalidKind Answers a Product Kind the Catalog does not Offer.
var ErrInvalidKind = errors.New("invalid product kind")

// ReadProductKind Reads the Kind a Caller Asked for, Defaulting to the Card.
func ReadProductKind(value string) (ProductKind, error) {
	switch ProductKind(value) {
	case "", KindSingle:
		return KindSingle, nil
	case KindSealed:
		return KindSealed, nil
	}
	return "", ErrInvalidKind
}

// ErrInvalidMatch Answers a Match Mode the Catalog does not Offer.
var ErrInvalidMatch = errors.New("invalid match mode")

// CardQuery Names the Card to Look for and how Wide its Title may be.
type CardQuery struct {
	Name  string
	Match MatchMode
	Kind  ProductKind
}

// Sealed Answers whether this Query Asks for Unopened Product.
func (q CardQuery) Sealed() bool { return q.Kind == KindSealed }

// AcceptsTitle Answers whether a Store Title Belongs to this Query.
// A Sealed Title Never Follows the Grammar of a Printing: a Store Writes
// `Aetherdrift: "Collector Booster Pack"`, never `Collector Booster Pack (ADF)`.
// So a Sealed Question Reads the Name wherever the Title Puts it.
func (q CardQuery) AcceptsTitle(title string) bool {
	if q.Match == MatchIncludes || q.Sealed() {
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

// ReadCardKey Reduces a Store Title to the Card it Names, Dropping the Printing.
// `Winged Kuriboh`, `LDS3-EN100 \u201cWinged Kuriboh\u201d Common` and
// `Winged Kuriboh (PUR)` Answer the same Thing, so three Sources Selling one
// Card Stop Looking like three Cards.
func ReadCardKey(title string) string {
	title = NormalizeCard(plainDashes(title))
	if quoted := readQuotedName(title); quoted != "" {
		return quoted
	}
	return trimPrintingTail(title)
}

// plainDashes Levels the three Dashes a Store may Type, because `Kuriboh -
// Multiply!` and `Kuriboh \u2013 Multiply!` are one Card Written twice.
func plainDashes(title string) string {
	return strings.NewReplacer("\u2013", "-", "\u2014", "-").Replace(title)
}

// readQuotedName Returns the Name a Store Wrapped in Quotes, or an empty String.
func readQuotedName(title string) string {
	for _, pair := range quotePairs {
		opening := strings.Index(title, pair[0])
		if opening < 0 {
			continue
		}
		rest := title[opening+len(pair[0]):]
		if closing := strings.Index(rest, pair[1]); closing > 0 {
			return rest[:closing]
		}
	}
	return ""
}

// trimPrintingTail Cuts the Edition a Store Appends, and only when it is one.
func trimPrintingTail(title string) string {
	shortest := title
	for _, bracket := range editionBrackets {
		if cut := strings.Index(title, bracket); cut > 0 && cut < len(shortest) {
			shortest = title[:cut]
		}
	}
	for _, dash := range editionDashes {
		cut := strings.Index(title, dash)
		if cut <= 0 || cut >= len(shortest) {
			continue
		}
		fields := strings.Fields(title[cut+len(dash):])
		if len(fields) > 0 && strings.ContainsAny(fields[0], "0123456789") {
			shortest = title[:cut]
		}
	}
	return shortest
}

// PriceGroupOf Names the Set of Offers whose Prices Compare with this one.
// Under MatchExact every Offer is the same Card and its Printings compete.
// Under MatchIncludes each Title is another Card, and a Starlight Rare of one
// must not Judge a Common of another.
func (q CardQuery) PriceGroupOf(item Offer) string {
	if q.Match == MatchIncludes || q.Sealed() {
		return item.CardKey
	}
	return NormalizeCard(q.Name)
}

// NameCards Tells every Offer which Card or known Printing it is for, so nobody
// has to Read the Title again to Find out.
func NameCards(items []Offer) []Offer {
	for index := range items {
		items[index].CardKey = readOfferKey(items[index])
	}
	return items
}

func readOfferKey(item Offer) string {
	if item.Kind == KindSealed {
		return ReadSealedKey(item)
	}
	card := ReadCardKey(item.CardName)
	if item.Metadata["game"] != "pokemon" {
		return card
	}
	if function := item.Metadata["functional_key"]; function != "" {
		return strings.Join([]string{card, function}, "|")
	}
	if product := item.Metadata["product_id"]; product != "" {
		return strings.Join([]string{card, "printing", product}, "|")
	}
	return card
}

// ReadSealedKey Names the Sealed Product an Offer is for, Keeping the whole
// Title. A Booster Pack and a Booster Display of one Set Share every Word but
// one and Differ by Ten Times the Price: Trimming a Tail here would Make the
// Pack Look like a Display Priced Suspiciously low.
func ReadSealedKey(item Offer) string {
	name := NormalizeCard(plainDashes(item.CardName))
	if item.Edition == "" {
		return name
	}
	return strings.Join([]string{NormalizeCard(item.Edition), name}, "|")
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

// MarkSuspicious Flags an Offer priced below percent of the Median of its Peers.
// Peers are Offers of the same Card in the same Currency: a Price only Judges
// another Price when both Name the same Thing.
func MarkSuspicious(items []Offer, percent int, query CardQuery) []Offer {
	groups := groupOfferPrices(items, query)
	medians := findPriceMedians(groups)

	return applySuspicious(items, medians, percent, query)
}

// priceGroup Keys a Median by what makes two Prices comparable.
type priceGroup struct {
	card     string
	currency string
}

func groupOfferPrices(items []Offer, query CardQuery) map[priceGroup][]int {
	groups := make(map[priceGroup][]int)
	for _, item := range items {
		cents, err := parseCents(item.PriceAmount)
		if err != nil {
			continue
		}
		peers := priceGroup{query.PriceGroupOf(item), item.PriceCurrency}
		groups[peers] = append(groups[peers], cents)
	}
	return groups
}

func findPriceMedians(groups map[priceGroup][]int) map[priceGroup]int {
	medians := make(map[priceGroup]int, len(groups))
	for group, prices := range groups {
		sort.Ints(prices)
		medians[group] = prices[len(prices)/2]
	}
	return medians
}

func applySuspicious(items []Offer, medians map[priceGroup]int, percent int,
	query CardQuery) []Offer {
	reason := fmt.Sprintf("price_below_%d_percent_median", percent)
	for index := range items {
		cents, err := parseCents(items[index].PriceAmount)
		peers := priceGroup{query.PriceGroupOf(items[index]), items[index].PriceCurrency}
		median := medians[peers]
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
