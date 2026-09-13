package prestashop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"golang.org/x/net/html"
)

const defaultMaxPages = 20

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

// Client Reads product listings rendered by a PrestaShop storefront.
type Client struct {
	Fetcher    SourceFetcher
	Domain     string
	Name       string
	SearchPath string
	Currency   string
	MaxPages   int
}

func (c Client) FindOffers(ctx context.Context, name string) ([]offer.Offer, error) {
	if c.Fetcher == nil || c.Domain == "" || strings.TrimSpace(name) == "" {
		return nil, errors.New("invalid PrestaShop client")
	}
	maxPages := c.MaxPages
	if maxPages < 1 {
		maxPages = defaultMaxPages
	}
	items := make([]offer.Offer, 0)
	for page := 1; page <= maxPages; page++ {
		data, err := c.Fetcher.FetchSource(ctx, c.Domain, c.searchURL(name, page))
		if err != nil {
			return nil, err
		}
		pageItems, hasNext, err := c.parsePage(data, name)
		if err != nil {
			return nil, fmt.Errorf("parse PrestaShop page %d: %w", page, err)
		}
		items = append(items, pageItems...)
		if !hasNext {
			return offer.DeduplicateOffers(items), nil
		}
	}
	return nil, fmt.Errorf("PrestaShop search exceeded %d pages", maxPages)
}

func (c Client) searchURL(name string, page int) string {
	path := c.SearchPath
	if path == "" {
		path = "/index.php"
	}
	query := url.Values{"controller": {"search"}, "s": {name}}
	if page > 1 {
		query.Set("page", strconv.Itoa(page))
	}
	return "https://" + c.Domain + path + "?" + query.Encode()
}

func (c Client) parsePage(data []byte, wanted string) ([]offer.Offer, bool, error) {
	items := make([]offer.Offer, 0)
	remaining := data
	for {
		start := bytes.Index(remaining, []byte(`<article class="product-miniature`))
		if start < 0 {
			break
		}
		remaining = remaining[start:]
		end := bytes.Index(remaining, []byte("</article>"))
		if end < 0 {
			return nil, false, errors.New("unterminated PrestaShop product")
		}
		fragment := remaining[:end+len("</article>")]
		remaining = remaining[end+len("</article>"):]
		document, err := html.Parse(bytes.NewReader(fragment))
		if err != nil {
			return nil, false, err
		}
		article := descendantWithClass(document, "product-miniature")
		item, ok := c.parseProduct(article, wanted)
		if ok {
			items = append(items, item)
		}
	}
	page := strings.ToLower(string(data))
	hasNext := strings.Contains(page, `rel="next"`) || strings.Contains(page, `rel='next'`)
	return items, hasNext, nil
}

func (c Client) parseProduct(node *html.Node, wanted string) (offer.Offer, bool) {
	id := attribute(node, "data-id-product")
	titleNode := descendantWithClass(node, "product-title")
	priceNode := descendantWithClass(node, "price")
	linkNode := firstElement(titleNode, "a")
	title := nodeText(linkNode)
	if id == "" || !matchesCard(title, wanted) || priceNode == nil || linkNode == nil {
		return offer.Offer{}, false
	}
	amount := digits(nodeText(priceNode))
	link := attribute(linkNode, "href")
	if amount == "" || link == "" {
		return offer.Offer{}, false
	}
	store := c.Name
	if store == "" {
		store = c.Domain
	}
	currency := c.Currency
	if currency == "" {
		currency = "CLP"
	}
	status := "available"
	text := strings.ToLower(nodeText(node))
	if descendantWithClass(node, "out_of_stock") != nil || strings.Contains(text, "en stock: 0") {
		status = "unavailable"
	}
	item := offer.Offer{ID: c.Domain + ":" + id, CardName: title, Store: store, PriceAmount: amount, PriceCurrency: currency, URL: link, Source: c.Domain, StockStatus: status}
	return item, offer.ValidateOffer(item) == nil
}

func matchesCard(title, name string) bool {
	title = offer.NormalizeCard(title)
	name = offer.NormalizeCard(name)
	if title == name {
		return true
	}
	for _, separator := range []string{" | ", " (", " [", " - ", " — "} {
		if strings.HasPrefix(title, name+separator) {
			return true
		}
	}
	return false
}

func hasNextPage(document *html.Node) bool {
	for node := range document.Descendants() {
		if node.Type == html.ElementNode && node.Data == "link" && attribute(node, "rel") == "next" && attribute(node, "href") != "" {
			return true
		}
	}
	return false
}

func descendantWithClass(node *html.Node, class string) *html.Node {
	if node == nil {
		return nil
	}
	for candidate := range node.Descendants() {
		if candidate.Type == html.ElementNode && hasClass(candidate, class) {
			return candidate
		}
	}
	return nil
}

func firstElement(node *html.Node, name string) *html.Node {
	if node == nil {
		return nil
	}
	for candidate := range node.Descendants() {
		if candidate.Type == html.ElementNode && candidate.Data == name {
			return candidate
		}
	}
	return nil
}

func attribute(node *html.Node, key string) string {
	if node == nil {
		return ""
	}
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

func hasClass(node *html.Node, name string) bool {
	for _, class := range strings.Fields(attribute(node, "class")) {
		if class == name {
			return true
		}
	}
	return false
}

func nodeText(node *html.Node) string {
	if node == nil {
		return ""
	}
	var text strings.Builder
	for candidate := range node.Descendants() {
		if candidate.Type == html.TextNode {
			text.WriteString(candidate.Data)
			text.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(text.String()), " ")
}

func digits(value string) string {
	var result strings.Builder
	for _, character := range value {
		if character >= '0' && character <= '9' {
			result.WriteRune(character)
		}
	}
	return result.String()
}