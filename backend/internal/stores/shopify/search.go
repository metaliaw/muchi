package shopify

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func (c Client) findProducts(ctx context.Context, name string) ([]string, error) {
	query := url.Values{"q": {`"` + strings.ReplaceAll(name, `"`, "") + `"`}, "type": {"product"}, "options[prefix]": {"none"}, "filter.v.availability": {"1"}}
	products := make([]string, 0)
	seen := map[string]bool{}
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		data, err := c.fetchPage(ctx, "/search?"+query.Encode())
		if err != nil {
			return nil, err
		}
		links, next, err := readSearch(data, c.Domain, page)
		if err != nil {
			return nil, err
		}
		before := len(products)
		for _, link := range links {
			if !seen[link] {
				products = append(products, link)
				seen[link] = true
			}
		}
		if page > 1 && len(products) == before {
			return nil, errors.New("Shopify pagination repeated products")
		}
		if !next {
			return products, nil
		}
	}
}

func readSearch(data []byte, domain string, page int) ([]string, bool, error) {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, false, err
	}
	products := make([]string, 0)
	valid, next := false, false
	for node := range document.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		attrs := readAttributes(node)
		if node.Data == "input" && attrs["name"] == "q" {
			valid = true
		}
		if node.Data != "a" {
			continue
		}
		link, err := url.Parse(attrs["href"])
		if err != nil || !sameStore(link, domain) {
			continue
		}
		if path := productPath(link, domain); path != "" {
			products = append(products, path)
		}
		number, _ := strconv.Atoi(link.Query().Get("page"))
		if link.Path == "/search" && number == page+1 {
			next = true
		}
	}
	if !valid {
		return nil, false, errors.New("Shopify page missing search form")
	}
	return products, next, nil
}

func readAttributes(node *html.Node) map[string]string {
	values := make(map[string]string, len(node.Attr))
	for _, attr := range node.Attr {
		values[attr.Key] = attr.Val
	}
	return values
}

func sameStore(link *url.URL, domain string) bool {
	validScheme := link.Scheme == "" || link.Scheme == "https"
	sameHost := link.Host == "" || strings.TrimPrefix(link.Host, "www.") == strings.TrimPrefix(domain, "www.")
	return validScheme && sameHost && link.User == nil
}

func productPath(link *url.URL, domain string) string {
	if !sameStore(link, domain) {
		return ""
	}
	_, handle, found := strings.Cut(link.Path, "/products/")
	if !found || handle == "" || strings.ContainsAny(handle, "/\\") || strings.Contains(handle, ".") {
		return ""
	}
	return "/products/" + url.PathEscape(handle)
}
