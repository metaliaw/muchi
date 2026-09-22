package shopify

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// The Theme Prints the same Number that Caps the Quantity Box a Buyer Sees.
const countedPage = `<div class="quantity">Cantidad</div>
<script>window.product = {"selected_variant_id": 10,"inventories": {
  "10": {"inventory_management":"shopify","inventory_policy":"deny","inventory_quantity": 4,"inventory_message":"4 disponibles"},
  "11": {"inventory_management":"shopify","inventory_policy":"deny","inventory_quantity": 0,"inventory_message":"Sold out"}}}</script>`

func TestThePageCountsWhatTheAjaxProductWillNot(t *testing.T) {
	client := Client{Domain: "cards.test", Fetcher: fixtureFetcher(func(target string) ([]byte, error) {
		u, _ := url.Parse(target)
		switch u.Path {
		case "/products/ring.js":
			return []byte(productFixture), nil
		case "/products/ring":
			if u.Query().Get("variant") != "10" {
				return nil, errors.New("page asked without its variant")
			}
			return []byte(countedPage), nil
		}
		return nil, errors.New("unexpected request: " + target)
	})}

	reading, err := client.CheckStock(context.Background(),
		offer.Offer{URL: "https://cards.test/products/ring?variant=10"})

	if err != nil || reading.Status != "available" || reading.Quantity == nil || *reading.Quantity != 4 {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}
}

func TestAPageWithoutACounterKeepsItsSilence(t *testing.T) {
	// Inventar una Cifra Convertiría un Silencio en una Promesa.
	if units := readVariantUnits([]byte(`{"id":10,"available":true}`), "10"); units != nil {
		t.Fatalf("units=%d", *units)
	}
}

func TestTheCountBelongsToTheVariantAsked(t *testing.T) {
	units := readVariantUnits([]byte(countedPage), "11")
	if units == nil || *units != 0 {
		t.Fatalf("units=%v", units)
	}
}

// Ineko Writes no Map: it Prints the Sentence a Buyer Reads, and that Number
// is the one its Quantity Box Stops at.
func TestThePageThatOnlySaysItInWords(t *testing.T) {
	page := []byte(`<div class="stock-urgency-message">🔥 ¡Apúrate, quedan solo 4 unidades en stock! 🔥</div>`)
	units := readVariantUnits(page, "44801729364243")
	if units == nil || *units != 4 {
		t.Fatalf("units=%v", units)
	}
}
