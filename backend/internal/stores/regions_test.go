package stores

import (
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// TestResolveAddressReadsStoreRegionNames Covers every Region stores.yaml
// Writes, and the Codes a Caller may Send instead.
func TestResolveAddressReadsStoreRegionNames(t *testing.T) {
	for region, want := range map[string]string{
		"Región Metropolitana": "RM", "Biobío": "BI", "Los Lagos": "LL", "Magallanes": "MA",
		"Valparaíso": "VS", "Región de Ñuble": "NB", "CL-RM": "RM", "rm": "RM", "Narnia": "Narnia",
	} {
		got := resolveAddress(offer.ShippingAddress{Country: "Chile", Region: region})
		if got.Country != "CL" || got.Region != want {
			t.Errorf("%s: got %+v, want %s", region, got, want)
		}
	}
	if got := resolveAddress(offer.ShippingAddress{Country: "AR", Region: "Córdoba"}); got.Region != "Córdoba" {
		t.Errorf("foreign address changed: %+v", got)
	}
}
