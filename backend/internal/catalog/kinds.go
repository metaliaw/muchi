package catalog

import "net/url"

// SealedOnly Names the Stores that Sell no Loose Card.
func (b Builder) SealedOnly() map[string]bool {
	named := map[string]bool{}
	for domain, store := range b.Shops.Config.Stores {
		if !store.SellsSingles() {
			named[domain] = true
		}
	}
	return named
}

// SinglesOnly Names the Sources that Sell Loose Cards only.
func (b Builder) SinglesOnly() map[string]bool {
	named := map[string]bool{}
	for _, provider := range b.Shops.Config.SearchProviders {
		if provider.ServesSealed() {
			continue
		}
		if host := readHost(provider.URL); host != "" {
			named[host] = true
		}
	}
	for domain, store := range b.Shops.Config.Stores {
		if !store.SellsSealed() {
			named[domain] = true
		}
	}
	return named
}

func readHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	return parsed.Host
}
