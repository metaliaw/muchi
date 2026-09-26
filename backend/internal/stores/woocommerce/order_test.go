package woocommerce

import (
	"context"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// TestPlaceOrderRefusesThePlaceholderBuyer Keeps a real Order from ever
// Landing on the reserved, undeliverable Placeholder Mailbox: PlaceOrder
// Refuses before any Call, until virtualBuyerEmail Names a real Mailbox.
func TestPlaceOrderRefusesThePlaceholderBuyer(t *testing.T) {
	sessions := &cartSessions{}
	client := Client{Domain: "woo.test", Sessions: sessions}
	ring := model.Offer{ID: "ring", Source: "woo.test", VariantID: "7"}
	_, err := client.PlaceOrder(context.Background(), []model.CartLine{{Offer: ring, Quantity: 1}}, model.ShippingAddress{Country: "CL"})
	if err != ErrVirtualBuyerNotConfigured {
		t.Fatalf("err = %v, want ErrVirtualBuyerNotConfigured", err)
	}
	if len(sessions.calls) != 0 {
		t.Fatalf("calls = %+v, want none before the buyer is real", sessions.calls)
	}
}
