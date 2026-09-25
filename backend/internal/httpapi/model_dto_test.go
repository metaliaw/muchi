package httpapi

import (
	"encoding/json"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"strings"
	"testing"
)

func TestResultKeepsWireNamesAndHidesLease(t *testing.T) {
	value := model.Result{SearchID: "search-1", Items: []model.Item{{ID: "item-1", SearchID: "search-1", Status: model.ItemRunning, LeaseOwner: "worker-private", Offers: []model.Offer{}}}}
	data, err := json.Marshal(renderResult(value))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"search_id":"search-1"`, `"sequence":0`, `"offers":[]`, `"has_more":false`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("missing %s in %s", field, data)
		}
	}
	if strings.Contains(string(data), "worker-private") || strings.Contains(string(data), "Lease") {
		t.Fatalf("lease leaked: %s", data)
	}
}

func TestCartQuoteKeepsWireNames(t *testing.T) {
	value := model.CartQuote{Currency: "CLP", Items: "1000", Shipping: "500", Total: "1500", Lines: []model.QuoteLine{{OfferID: "offer-1", Quantity: 2, UnitPrice: "500"}}, ShippingRates: []model.ShippingRate{{ID: "pickup", Name: "Pickup", Price: "500", Selected: true}}, PaymentMethods: []string{"bacs"}}
	data, err := json.Marshal(renderCartQuote(value))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"currency":"CLP","items_total":"1000","shipping_total":"500","total":"1500","lines":[{"offer_id":"offer-1","quantity":2,"unit_price":"500"}],"shipping_rates":[{"id":"pickup","name":"Pickup","price":"500","selected":true}],"payment_methods":["bacs"]}`
	if string(data) != want {
		t.Fatalf("quote = %s, want %s", data, want)
	}
}
