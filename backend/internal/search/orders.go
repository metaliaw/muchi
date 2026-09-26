package search

import (
	"context"
	"net/url"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// PlaceOrder Checks one Store's Cart out for real, for Offers this Search
// already Found. Unlike CheckoutLinks, every Line must Belong to the same
// Store: Placing an Order is a single Transaction, and Splitting one Request
// across Stores would Hide which one actually Charged.
//
// The Idempotency Key Guards against a doubled Tap the way CreateSearch does:
// a retried Call with the same Key and the same Cart Answers the Order
// already Placed, never a second one.
func (s Service) PlaceOrder(ctx context.Context, id, key string, requests []model.CartRequest, address model.ShippingAddress) (model.Order, error) {
	if id == "" || key == "" || len(requests) == 0 || len(requests) > maxCheckedOffers || s.Orderer == nil || s.Orders == nil {
		return model.Order{}, ErrInvalid
	}
	for _, request := range requests {
		if request.OfferID == "" || request.Quantity <= 0 || request.Quantity > maxCheckoutUnits {
			return model.Order{}, ErrInvalid
		}
	}
	if address.Country == "" {
		return model.Order{}, ErrInvalid
	}
	found, err := s.collectSearchOffers(ctx, id)
	if err != nil {
		return model.Order{}, err
	}
	domain := ""
	var lines []model.CartLine
	for _, request := range requests {
		item, known := found[request.OfferID]
		if !known {
			return model.Order{}, ErrNotFound
		}
		itemDomain := ""
		if link, err := url.Parse(item.URL); err == nil {
			itemDomain = link.Host
		}
		if domain == "" {
			domain = itemDomain
		} else if domain != itemDomain {
			return model.Order{}, ErrInvalid
		}
		lines = append(lines, model.CartLine{Offer: item, Quantity: request.Quantity})
	}
	order, err := s.Orderer.PlaceOrder(ctx, domain, lines, address)
	if err != nil {
		return model.Order{}, err
	}
	order.SearchID = id
	hash := HashPayload(map[string]any{"search_id": id, "action": "place-order", "domain": domain, "items": requests})
	return s.Orders.CreateOrder(ctx, key, hash, order)
}
