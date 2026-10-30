package woocommerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	body := []byte(`{"id":501,"status":"processing"}`)
	if !VerifyWebhookSignature("shh", body, sign("shh", body)) {
		t.Fatal("valid signature was rejected")
	}
	cases := []struct {
		name      string
		secret    string
		signature string
	}{
		{"wrong secret", "shh", sign("other", body)},
		{"empty secret", "", sign("shh", body)},
		{"empty signature", "shh", ""},
		{"tampered body", "shh", sign("shh", []byte(`{"id":999,"status":"processing"}`))},
	}
	for _, test := range cases {
		if VerifyWebhookSignature(test.secret, body, test.signature) {
			t.Errorf("%s: signature was accepted", test.name)
		}
	}
}

func TestWebhookOrderStatus(t *testing.T) {
	cases := []struct {
		raw  string
		want model.OrderStatus
		ok   bool
	}{
		{"processing", model.OrderConfirmed, true},
		{"completed", model.OrderConfirmed, true},
		{"cancelled", model.OrderReleased, true},
		{"failed", model.OrderReleased, true},
		{"refunded", model.OrderReleased, true},
		{"on-hold", "", false},
		{"pending", "", false},
		{"", "", false},
	}
	for _, test := range cases {
		got, ok := WebhookOrderStatus(test.raw)
		if got != test.want || ok != test.ok {
			t.Errorf("%q: got (%q, %v), want (%q, %v)", test.raw, got, ok, test.want, test.ok)
		}
	}
}
