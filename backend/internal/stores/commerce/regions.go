package commerce

import (
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// chileRegions Names each Region by its https://www.iso.org/obp/ui/#iso:code:3166:CL
// Subdivision, the Code WooCommerce and Shopify both Build their own from.
// Keys are Folded with offer.NormalizeCard, so "Biobío" and "biobio" Meet.
var chileRegions = map[string]string{
	"arica y parinacota": "AP", "tarapaca": "TA", "antofagasta": "AN",
	"atacama": "AT", "coquimbo": "CO", "valparaiso": "VS",
	"region metropolitana": "RM", "metropolitana": "RM", "santiago": "RM",
	"libertador general bernardo o'higgins": "LI", "o'higgins": "LI", "ohiggins": "LI",
	"maule": "ML", "nuble": "NB", "biobio": "BI", "araucania": "AR", "la araucania": "AR",
	"los rios": "LR", "los lagos": "LL", "aysen": "AI", "magallanes": "MA",
	"magallanes y la antartica chilena": "MA",
}

// chileCountry Holds the Spellings a Buyer or a Store Entry Uses for Chile.
var chileCountry = map[string]bool{"cl": true, "chile": true}

// prepareAddress Writes an Address in the Codes one Platform Reads: Country
// "CL" and Region the bare Subdivision, such as "RM", for Shopify, and with
// the further "CL-" Prefix for WooCommerce's own State Code. Quoting and
// Placing Agree on it, so Checkout Deserves the Address Quoting already Priced,
// never a raw Buyer Spelling the Store API may Refuse or Silently Mis-rate.
func prepareAddress(platform string, address model.ShippingAddress) model.ShippingAddress {
	address = resolveAddress(address)
	if platform == "woocommerce" && address.Country == "CL" && len(address.Region) == 2 {
		address.Region = "CL-" + address.Region
	}
	return address
}

// resolveAddress Turns the Names a Buyer Writes into the Codes a Store Cart
// Reads: Country "CL" and Region the bare Subdivision, such as "RM". A Code the
// Buyer already Sent Passes through; an Unknown Name Passes as Written, and the
// Store Decides.
func resolveAddress(address model.ShippingAddress) model.ShippingAddress {
	if !chileCountry[model.NormalizeCard(address.Country)] {
		return address
	}
	address.Country = "CL"
	region := model.NormalizeCard(address.Region)
	region = strings.TrimPrefix(strings.TrimPrefix(region, "region del "), "region de ")
	if code, found := chileRegions[region]; found {
		address.Region = code
		return address
	}
	if code := strings.ToUpper(strings.TrimPrefix(region, "cl-")); len(code) == 2 {
		address.Region = code
	}
	return address
}
