package jumpseller

import (
	"bytes"
	"encoding/json"
	"errors"
	"golang.org/x/net/html"
	"net/url"
	"strconv"
	"strings"
)

func readProduct(data []byte, path, domain string) (formReply, string, []variantReply, error) {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return formReply{}, "", nil, err
	}
	form, currency, err := readForm(document, path, domain)
	if err != nil {
		return form, "", nil, err
	}
	variants, err := readVariants(document, form.Info.Product.ID)
	return form, currency, variants, err
}
func readForm(document *html.Node, path, domain string) (formReply, string, error) {
	var form formReply
	currency := ""
	found := false
	for node := range document.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		if node.Data == "meta" && attribute(node, "property") == "product:price:currency" {
			currency = attribute(node, "content")
		}
		if node.Data != "script" || !hasClass(node, "product-form-json") {
			continue
		}
		var candidate formReply
		if err := json.Unmarshal([]byte(nodeText(node)), &candidate); err != nil {
			return form, "", err
		}
		link, err := url.Parse(candidate.Info.Product.URL)
		if err == nil && sameStore(link, domain) && link.EscapedPath() == path {
			form = candidate
			found = true
		}
	}
	if !found || form.Info.Product.ID <= 0 || form.Info.Product.Name == "" {
		return form, "", errors.New("invalid Jumpseller product form")
	}
	if len(currency) != 3 || strings.Trim(currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return form, "", errors.New("invalid Jumpseller currency")
	}
	return form, currency, nil
}

func readVariants(document *html.Node, id int64) ([]variantReply, error) {
	for node := range document.Descendants() {
		if node.Type == html.ElementNode && node.Data == "script" && hasClass(node, "product-json") && attribute(node, "data-productid") == strconv.FormatInt(id, 10) {
			var variants []variantReply
			err := json.Unmarshal([]byte(nodeText(node)), &variants)
			if err == nil && variants == nil {
				err = errors.New("invalid Jumpseller variant data")
			}
			return variants, err
		}
	}
	return nil, errors.New("Jumpseller product missing variant data")
}

func sameStore(link *url.URL, domain string) bool {
	return (link.Scheme == "" || link.Scheme == "https") && link.User == nil && (link.Host == "" || strings.TrimPrefix(link.Host, "www.") == strings.TrimPrefix(domain, "www."))
}
func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
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
	var text strings.Builder
	for child := range node.Descendants() {
		if child.Type == html.TextNode {
			text.WriteString(child.Data)
		}
	}
	return strings.TrimSpace(text.String())
}
func uniquePaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}
