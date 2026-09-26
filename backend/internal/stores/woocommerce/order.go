package woocommerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// virtualBuyerName and virtualBuyerEmail Name Muchi itself as the Buyer of
// Record until a real Person's Identity Enters the Checkout Request. Every
// Order this Places Lands, at worst, in a Mailbox Muchi Reads — never in a
// Stranger's.
//
// TODO: orders@muchi.invalid is a reserved, undeliverable Address (RFC 2606).
// Replace it with a real Muchi Mailbox before any Order is Placed against a
// live Store; until then, PlaceOrder Refuses to Run.
const (
	virtualBuyerName  = "Muchi"
	virtualBuyerEmail = "orders@muchi.invalid"
)

// ErrVirtualBuyerNotConfigured Answers a PlaceOrder Call while the Virtual
// Buyer is still the Placeholder: Real Money should never Move on a Mailbox
// nobody Reads.
var ErrVirtualBuyerNotConfigured = errors.New("woocommerce: virtual buyer email is a placeholder")

// checkoutReply Reads only what PlaceOrder Needs back; the Store API Answers
// far more per https://github.com/woocommerce/woocommerce/blob/trunk/plugins/woocommerce/src/StoreApi/docs/checkout.md
type checkoutReply struct {
	OrderID       int    `json:"order_id"`
	Status        string `json:"status"`
	PaymentResult struct {
		PaymentStatus string `json:"payment_status"`
		RedirectURL   string `json:"redirect_url"`
	} `json:"payment_result"`
}

// PlaceOrder Fills a Cart, Picks its Store-Selected Shipping Rate, and
// Checks it out by Bank Transfer (`bacs`): the one Payment Method the Store
// API Confirms without a Browser or a Card. The Order Lands "pending
// payment"; nothing here Waits for the Transfer to Arrive.
func (c Client) PlaceOrder(ctx context.Context, lines []model.CartLine, address model.ShippingAddress) (model.Order, error) {
	if virtualBuyerEmail == "orders@muchi.invalid" {
		return model.Order{}, ErrVirtualBuyerNotConfigured
	}
	if c.Sessions == nil || len(lines) == 0 {
		return model.Order{}, ErrNotQuotable
	}
	token, cart, _, err := c.fillCart(ctx, lines, address)
	if err != nil {
		return model.Order{}, err
	}
	rate := selectedRate(cart)
	if rate == "" {
		return model.Order{}, fmt.Errorf("%s offered no shipping rate", c.Domain)
	}
	session := http.Header{"Cart-Token": {token}}
	base := "https://" + c.Domain + "/wp-json/wc/store/v1"
	rateBody, _ := json.Marshal(map[string]string{"package_id": "0", "rate_id": rate})
	if _, _, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodPost, Target: base + "/cart/select-shipping-rate", Body: rateBody, Header: session}); err != nil {
		return model.Order{}, fmt.Errorf("select shipping rate %s: %w", rate, err)
	}
	person := map[string]string{
		"first_name": virtualBuyerName, "last_name": virtualBuyerName, "email": virtualBuyerEmail,
		"country": address.Country, "state": address.Region, "city": address.City, "postcode": address.Postcode,
	}
	body, _ := json.Marshal(map[string]any{
		"billing_address": person, "shipping_address": person, "payment_method": "bacs",
	})
	data, _, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodPost, Target: base + "/checkout", Body: body, Header: session})
	if err != nil {
		return model.Order{}, fmt.Errorf("checkout %s: %w", c.Domain, err)
	}
	var reply checkoutReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return model.Order{}, fmt.Errorf("decode checkout %s: %w", c.Domain, err)
	}
	if reply.OrderID == 0 {
		return model.Order{}, fmt.Errorf("%s checkout answered no order", c.Domain)
	}
	return model.Order{
		Store: c.Name, Domain: c.Domain, Status: model.OrderPending,
		StoreOrder: fmt.Sprintf("%d", reply.OrderID), PaymentURL: reply.PaymentResult.RedirectURL,
	}, nil
}

// selectedRate Reads the Rate the Cart already Picked; a fresh Cart Selects
// the cheapest one on its own, so PlaceOrder never has to Guess.
func selectedRate(cart cartReply) string {
	for _, pack := range cart.Packages {
		for _, candidate := range pack.Rates {
			if candidate.Selected {
				return candidate.ID
			}
		}
	}
	return ""
}
