package offer

import (
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

var ErrInvalidOffer = errors.New("invalid offer")

// suspiciousPriceThresholdPercent flags an offer priced below this share of its group median.
const suspiciousPriceThresholdPercent = 30

type Offer struct {
	ID               string            `json:"id"`
	CardName         string            `json:"card_name"`
	Store            string            `json:"store"`
	PriceAmount      string            `json:"price_amount"`
	PriceCurrency    string            `json:"price_currency"`
	URL              string            `json:"url"`
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

func MarkSuspicious(items []Offer) []Offer {
	groups := groupOfferPrices(items)
	medians := findPriceMedians(groups)

	return applySuspicious(items, medians)
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

func applySuspicious(items []Offer, medians map[string]int) []Offer {
	for index := range items {
		cents, err := parseCents(items[index].PriceAmount)
		median := medians[items[index].PriceCurrency]
		pricedLow := err == nil && median > 0 && cents*100 < median*suspiciousPriceThresholdPercent
		if pricedLow {
			items[index].Suspicious = true
			items[index].SuspiciousReason = "price_below_30_percent_median"
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
