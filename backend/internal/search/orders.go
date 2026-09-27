package search

import (
	"context"
	"net/url"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// maxStoreOrderLength Bounds the Order Number a Buyer Types: Store Numbers
// are Short, and the Field is Free Text from a Person.
const maxStoreOrderLength = 64

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

// LinkOrder Records the Cart Link a Buyer is about to Follow, for a Store
// Muchi cannot Place an Order at. The same-Store Rule and the Idempotency Key
// Work as in PlaceOrder; the Order Starts `linked`, and only the Buyer's own
// Report Moves it on.
func (s Service) LinkOrder(ctx context.Context, id, key string, requests []model.CartRequest) (model.Order, error) {
	if id == "" || key == "" || len(requests) == 0 || len(requests) > maxCheckedOffers || s.Orders == nil {
		return model.Order{}, ErrInvalid
	}
	for _, request := range requests {
		if request.OfferID == "" || request.Quantity <= 0 || request.Quantity > maxCheckoutUnits {
			return model.Order{}, ErrInvalid
		}
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
	plan := s.planCheckout(domain, lines)
	order := model.Order{
		SearchID: id, Store: plan.Store, Domain: plan.Domain, Lines: plan.Lines,
		Status: model.OrderLinked, PaymentURL: plan.URL,
	}
	hash := HashPayload(map[string]any{"search_id": id, "action": "link-order", "domain": domain, "items": requests})
	return s.Orders.CreateOrder(ctx, key, hash, order)
}

// ReportOrder Records what the Buyer Says the Store Answered: the Order
// Number it Showed. A Report Repeating the Number already Stored is a silent
// no-op, so a doubled Tap Answers the same Order.
func (s Service) ReportOrder(ctx context.Context, id, orderID, storeOrder string) (model.Order, error) {
	storeOrder = strings.TrimSpace(storeOrder)
	if storeOrder == "" || len(storeOrder) > maxStoreOrderLength || s.Orders == nil {
		return model.Order{}, ErrInvalid
	}
	order, err := s.GetOrder(ctx, id, orderID)
	if err != nil {
		return model.Order{}, err
	}
	if order.Status == model.OrderReported && order.StoreOrder == storeOrder {
		return order, nil
	}
	return s.Orders.ReportOrder(ctx, orderID, storeOrder)
}

// GetOrder Reads one Order back, but only through the Search that Placed it:
// an Order Id Asked under another Search Answers Not Found, the same as one
// that never Existed, so a Path cannot Reach an Order it did not Create.
func (s Service) GetOrder(ctx context.Context, id, orderID string) (model.Order, error) {
	if id == "" || orderID == "" || s.Orders == nil {
		return model.Order{}, ErrInvalid
	}
	order, err := s.Orders.GetOrder(ctx, orderID)
	if err != nil {
		return model.Order{}, err
	}
	if order.SearchID != id {
		return model.Order{}, model.ErrOrderNotFound
	}
	return order, nil
}

// ConfirmOrder Moves the Order a Store's own Webhook Names, by the Id the
// Store gave it, to whichever Status the Webhook Reported. It is Idempotent
// two ways: a redelivered Webhook Naming the Status the Order already Holds
// is a silent no-op, and a Webhook Naming a Status the Order already Moved
// past Loses to whichever Caller Won that Race, per MoveOrderStatus.
func (s Service) ConfirmOrder(ctx context.Context, domain, storeOrder string, target model.OrderStatus) (model.Order, error) {
	if domain == "" || storeOrder == "" || s.Orders == nil {
		return model.Order{}, ErrInvalid
	}
	order, err := s.Orders.FindOrderByStoreOrder(ctx, domain, storeOrder)
	if err != nil {
		return model.Order{}, err
	}
	if order.Status == target {
		return order, nil
	}
	return s.Orders.MoveOrderStatus(ctx, order.ID, model.OrderPending, target)
}
