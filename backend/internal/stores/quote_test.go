package stores

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// recordingSessions Remembers every Call and Fails them all, which is enough
// to See the Address each Platform was Sent.
type recordingSessions struct{ calls []source.SessionRequest }

func (r *recordingSessions) SendSession(_ context.Context, _ string, request source.SessionRequest) ([]byte, http.Header, error) {
	r.calls = append(r.calls, request)
	if request.Method == http.MethodGet && strings.HasSuffix(request.Target, "/cart") {
		return []byte(`{}`), http.Header{"Cart-Token": {"tok"}}, nil
	}
	return []byte(`{}`), nil, nil
}

// TestQuoterSendsEachPlatformItsRegionCode Reads a Region Name the Way
// stores.yaml Writes it and Hands WooCommerce "CL-RM" and Shopify "RM".
func TestQuoterSendsEachPlatformItsRegionCode(t *testing.T) {
	sessions := &recordingSessions{}
	quoter := Quoter{Sessions: sessions, Config: Config{Stores: map[string]StoreConfig{
		"woo.test":  {Platform: "woocommerce", Enabled: true},
		"shop.test": {Platform: "shopify", Enabled: true},
		"js.test":   {Platform: "jumpseller", Enabled: true},
	}}}
	address := offer.ShippingAddress{Country: "Chile", Region: "Región Metropolitana"}
	woo := offer.Offer{Source: "woo.test", VariantID: "7"}
	_, _ = quoter.QuoteCart(context.Background(), "woo.test", []offer.CartLine{{Offer: woo, Quantity: 1}}, address)
	if last := string(sessions.calls[len(sessions.calls)-1].Body); !strings.Contains(last, `"state":"CL-RM"`) || !strings.Contains(last, `"country":"CL"`) {
		t.Fatalf("woocommerce address = %s", last)
	}
	shop := offer.Offer{Source: "shop.test", VariantID: "11"}
	_, _ = quoter.QuoteCart(context.Background(), "shop.test", []offer.CartLine{{Offer: shop, Quantity: 1}}, address)
	if last := sessions.calls[len(sessions.calls)-1].Target; !strings.Contains(last, "province%5D=RM&") {
		t.Fatalf("shopify rates target = %s", last)
	}
	if _, err := quoter.QuoteCart(context.Background(), "js.test", nil, address); err != offer.ErrNoQuote {
		t.Fatalf("jumpseller err = %v", err)
	}
}
