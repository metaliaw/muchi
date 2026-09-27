package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// memoryOrderRepository Plays a real OrderRepository well enough for
// ConfirmOrder's own Contract: Create, find by Store Order, and Move,
// guarded the same from-Status way db.Store's does.
type memoryOrderRepository struct{ orders map[string]model.Order }

func (r *memoryOrderRepository) CreateOrder(_ context.Context, _, _ string, order model.Order) (model.Order, error) {
	if r.orders == nil {
		r.orders = map[string]model.Order{}
	}
	order.ID = "order_" + order.StoreOrder
	r.orders[order.ID] = order
	return order, nil
}
func (r *memoryOrderRepository) GetOrder(_ context.Context, id string) (model.Order, error) {
	order, found := r.orders[id]
	if !found {
		return model.Order{}, model.ErrOrderNotFound
	}
	return order, nil
}
func (r *memoryOrderRepository) FindOrderByStoreOrder(_ context.Context, domain, storeOrder string) (model.Order, error) {
	for _, order := range r.orders {
		if order.Domain == domain && order.StoreOrder == storeOrder {
			return order, nil
		}
	}
	return model.Order{}, model.ErrOrderNotFound
}
func (r *memoryOrderRepository) ReportOrder(_ context.Context, id, storeOrder string) (model.Order, error) {
	order, found := r.orders[id]
	if !found {
		return model.Order{}, model.ErrOrderNotFound
	}
	if order.Status != model.OrderLinked {
		return model.Order{}, model.ErrOrderConflict
	}
	order.Status, order.StoreOrder = model.OrderReported, storeOrder
	r.orders[id] = order
	return order, nil
}
func (r *memoryOrderRepository) MoveOrderStatus(_ context.Context, id string, from, to model.OrderStatus) (model.Order, error) {
	order, found := r.orders[id]
	if !found {
		return model.Order{}, model.ErrOrderNotFound
	}
	if order.Status != from {
		return model.Order{}, model.ErrOrderConflict
	}
	order.Status = to
	r.orders[id] = order
	return order, nil
}

func sendWebhook(t *testing.T, api API, domain, secret, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/webhooks/woocommerce/"+domain+"/orders", strings.NewReader(body))
	if secret != "" {
		request.Header.Set("X-WC-Webhook-Signature", signBody(secret, []byte(body)))
	}
	recorder := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(recorder, request)
	return recorder
}

// signBody Mirrors woocommerce.VerifyWebhookSignature's own Construction, so
// the Test Signs the same way a real Store would.
func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// TestOrderWebhookMovesOrderOnValidSignature Spells the whole Path a real
// Store's own Delivery Takes: Verify, Map Status, Confirm.
func TestOrderWebhookMovesOrderOnValidSignature(t *testing.T) {
	repository := &memoryOrderRepository{}
	if _, err := repository.CreateOrder(context.Background(), "k", "h", model.Order{
		Domain: "woo.test", StoreOrder: "501", Status: model.OrderPending,
	}); err != nil {
		t.Fatal(err)
	}
	api := API{Searches: search.Service{Orders: repository}, OrderWebhookSecret: "shh"}
	body := `{"id":501,"status":"processing"}`
	recorder := sendWebhook(t, api, "woo.test", "shh", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body)
	}
	order, err := repository.GetOrder(context.Background(), "order_501")
	if err != nil || order.Status != model.OrderConfirmed {
		t.Fatalf("order = %+v err = %v", order, err)
	}
}

// TestOrderWebhookRejectsBadSignature Keeps a Forged Delivery from Moving
// anything: without the Secret, a Caller can Confirm or Release Orders it
// never Placed.
func TestOrderWebhookRejectsBadSignature(t *testing.T) {
	repository := &memoryOrderRepository{}
	if _, err := repository.CreateOrder(context.Background(), "k", "h", model.Order{
		Domain: "woo.test", StoreOrder: "501", Status: model.OrderPending,
	}); err != nil {
		t.Fatal(err)
	}
	api := API{Searches: search.Service{Orders: repository}, OrderWebhookSecret: "shh"}
	body := `{"id":501,"status":"processing"}`
	if recorder := sendWebhook(t, api, "woo.test", "wrong-secret", body); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
	order, err := repository.GetOrder(context.Background(), "order_501")
	if err != nil || order.Status != model.OrderPending {
		t.Fatalf("order = %+v err = %v, want untouched", order, err)
	}
}

// TestOrderWebhookRefusesWithoutASecretConfigured Keeps an unconfigured
// Deployment from accepting an unsigned Delivery as valid by Default.
func TestOrderWebhookRefusesWithoutASecretConfigured(t *testing.T) {
	api := API{Searches: search.Service{Orders: &memoryOrderRepository{}}}
	if recorder := sendWebhook(t, api, "woo.test", "", `{"id":501,"status":"processing"}`); recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", recorder.Code)
	}
}

// TestOrderWebhookIgnoresAStatusThatNamesNoMove Keeps "on-hold" and other
// unmapped Statuses from Erroring: they Ack 200 so the Store never Retries
// or Disables the Webhook over a Delivery that was never a Failure.
func TestOrderWebhookIgnoresAStatusThatNamesNoMove(t *testing.T) {
	repository := &memoryOrderRepository{}
	if _, err := repository.CreateOrder(context.Background(), "k", "h", model.Order{
		Domain: "woo.test", StoreOrder: "501", Status: model.OrderPending,
	}); err != nil {
		t.Fatal(err)
	}
	api := API{Searches: search.Service{Orders: repository}, OrderWebhookSecret: "shh"}
	body := `{"id":501,"status":"on-hold"}`
	if recorder := sendWebhook(t, api, "woo.test", "shh", body); recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body)
	}
	order, err := repository.GetOrder(context.Background(), "order_501")
	if err != nil || order.Status != model.OrderPending {
		t.Fatalf("order = %+v err = %v, want untouched", order, err)
	}
}

// TestOrderWebhookAcksAnUnknownOrder Keeps a Webhook for an Order this
// Domain never Placed (or already Forgot) from reading as a server Failure.
func TestOrderWebhookAcksAnUnknownOrder(t *testing.T) {
	api := API{Searches: search.Service{Orders: &memoryOrderRepository{}}, OrderWebhookSecret: "shh"}
	body := `{"id":999,"status":"completed"}`
	if recorder := sendWebhook(t, api, "woo.test", "shh", body); recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body)
	}
}

// TestLinkedOrderMovesToReportedOnlyOnce Spells the Buyer-Reported Path: a
// Link Records a `linked` Order at one Store, the Buyer's Report Moves it to
// `reported` with the Store's Number, a Repeat Answers the same Order, and a
// different Number after that Conflicts instead of Overwriting.
func TestLinkedOrderMovesToReportedOnlyOnce(t *testing.T) {
	repository := &memoryOrderRepository{}
	api := API{Searches: search.Service{Repository: &checkoutStore{}, Orders: repository}, Token: "secret"}
	call := func(path, key, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer secret")
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		recorder := httptest.NewRecorder()
		api.BuildHandler().ServeHTTP(recorder, request)
		return recorder
	}
	linked := call("/v1/searches/search_one/orders/links", "link-key", `{"items":[{"offer_id":"bolt","quantity":1}]}`)
	if linked.Code != http.StatusCreated || !strings.Contains(linked.Body.String(), `"status":"linked"`) || !strings.Contains(linked.Body.String(), `"domain":"shop.test"`) {
		t.Fatalf("link status = %d body = %s", linked.Code, linked.Body)
	}
	if mixed := call("/v1/searches/search_one/orders/links", "mixed-key", `{"items":[{"offer_id":"bolt","quantity":1},{"offer_id":"woo","quantity":1}]}`); mixed.Code != http.StatusBadRequest {
		t.Fatalf("mixed-store link status = %d", mixed.Code)
	}
	report := func(body string) *httptest.ResponseRecorder {
		return call("/v1/searches/search_one/orders/order_/report", "", body)
	}
	if first := report(`{"store_order":" #1042 "}`); first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"status":"reported"`) || !strings.Contains(first.Body.String(), `"store_order":"#1042"`) {
		t.Fatalf("report status = %d body = %s", first.Code, first.Body)
	}
	if again := report(`{"store_order":"#1042"}`); again.Code != http.StatusOK {
		t.Fatalf("repeated report status = %d", again.Code)
	}
	if other := report(`{"store_order":"#9999"}`); other.Code != http.StatusConflict {
		t.Fatalf("different number status = %d", other.Code)
	}
	if empty := report(`{"store_order":"  "}`); empty.Code != http.StatusBadRequest {
		t.Fatalf("empty number status = %d", empty.Code)
	}
	if foreign := call("/v1/searches/search_two/orders/order_/report", "", `{"store_order":"#1"}`); foreign.Code != http.StatusNotFound {
		t.Fatalf("other search status = %d", foreign.Code)
	}
}
