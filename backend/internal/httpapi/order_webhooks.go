package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

// maxWebhookBodyBytes Bounds one Delivery. A Store's own Order Payload is a
// few KB; anything past this is not a WooCommerce Webhook.
const maxWebhookBodyBytes = 1 << 20

func (a API) registerOrderWebhookRoutes(mux *http.ServeMux) {
	mux.Handle("POST /v1/webhooks/woocommerce/{domain}/orders", http.HandlerFunc(a.receiveWooCommerceOrderWebhook))
}

type wooCommerceOrderWebhook struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

// receiveWooCommerceOrderWebhook Moves the Order a Store's own `order.updated`
// Webhook Names to whichever Status its new Status Means. Every other Route
// in this API Trusts the caller's Bearer Token; this one Trusts the Store's
// HMAC Signature instead, because the Store — not a Muchi Client — Calls it.
func (a API) receiveWooCommerceOrderWebhook(w http.ResponseWriter, r *http.Request) {
	if a.OrderWebhookSecret == "" {
		http.Error(w, "Webhook not configured", http.StatusServiceUnavailable)
		return
	}
	domain := r.PathValue("domain")
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if !woocommerce.VerifyWebhookSignature(a.OrderWebhookSecret, body, r.Header.Get("X-WC-Webhook-Signature")) {
		a.log().Warn("Order Webhook Signature Rejected", "domain", domain)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}
	var delivery wooCommerceOrderWebhook
	if err := json.Unmarshal(body, &delivery); err != nil || delivery.ID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	target, moves := woocommerce.WebhookOrderStatus(delivery.Status)
	if !moves {
		// "on-hold", "pending" and any Status this App does not yet Read Name
		// no Move; Acknowledging keeps the Store from Retrying or Disabling
		// the Webhook over a Delivery that was never an Error.
		w.WriteHeader(http.StatusOK)
		return
	}
	storeOrder := strconv.Itoa(delivery.ID)
	_, err = a.Searches.ConfirmOrder(r.Context(), domain, storeOrder, target)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, model.ErrOrderNotFound), errors.Is(err, model.ErrOrderConflict):
		// Unknown to Muchi, or already Moved past this Status: neither is
		// this Delivery's Fault, and Retrying would not Change either.
		a.log().Info("Order Webhook Ignored", "domain", domain, "store_order", storeOrder, "target", target, "reason", err)
		w.WriteHeader(http.StatusOK)
	default:
		a.log().Error("Order Webhook Failed", "domain", domain, "store_order", storeOrder, "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func (a API) log() *slog.Logger {
	if a.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return a.Logger
}
