package woocommerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// VerifyWebhookSignature Checks the `X-WC-Webhook-Signature` Header the
// Store API Signs every Webhook Delivery with: base64(HMAC-SHA256(secret,
// rawBody)). https://woocommerce.github.io/woocommerce-rest-api-docs/#webhooks
func VerifyWebhookSignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

// WebhookOrderStatus Reads what a `pending` Order should Become when a
// Store's own `order.updated` Webhook Names its new Status. Answers false
// for a Status that Names no Move: "on-hold" and "pending" itself Settle
// nothing yet.
func WebhookOrderStatus(rawStatus string) (model.OrderStatus, bool) {
	switch rawStatus {
	case "processing", "completed":
		return model.OrderConfirmed, true
	case "cancelled", "failed", "refunded":
		return model.OrderReleased, true
	default:
		return "", false
	}
}
