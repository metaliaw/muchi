package handler

import "time"

type offerDTO struct {
	ID               string            `json:"id"`
	CardName         string            `json:"card_name"`
	Store            string            `json:"store"`
	PriceAmount      string            `json:"price_amount"`
	PriceCurrency    string            `json:"price_currency"`
	URL              string            `json:"url"`
	Image            string            `json:"image,omitempty"`
	VariantID        string            `json:"variant_id,omitempty"`
	Language         string            `json:"language,omitempty"`
	Condition        string            `json:"condition,omitempty"`
	Finish           string            `json:"finish,omitempty"`
	Edition          string            `json:"edition,omitempty"`
	Kind             string            `json:"kind,omitempty"`
	Locations        []string          `json:"locations,omitempty"`
	CardKey          string            `json:"card_key,omitempty"`
	Source           string            `json:"source"`
	StockStatus      string            `json:"stock_status"`
	StockQuantity    *int              `json:"stock_quantity,omitempty"`
	Suspicious       bool              `json:"suspicious"`
	SuspiciousReason string            `json:"suspicious_reason,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type stockReadingDTO struct {
	Status   string `json:"stock_status"`
	Quantity *int   `json:"stock_quantity,omitempty"`
}

type shippingAddressDTO struct {
	Country  string `json:"country"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	Postcode string `json:"postcode,omitempty"`
}

type cartQuoteDTO struct {
	Currency       string            `json:"currency"`
	Items          string            `json:"items_total"`
	Shipping       string            `json:"shipping_total"`
	Total          string            `json:"total"`
	Lines          []quoteLineDTO    `json:"lines"`
	ShippingRates  []shippingRateDTO `json:"shipping_rates"`
	PaymentMethods []string          `json:"payment_methods"`
	Notices        []string          `json:"notices,omitempty"`
}

type quoteLineDTO struct {
	OfferID   string `json:"offer_id"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

type shippingRateDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    string `json:"price"`
	Selected bool   `json:"selected"`
}

type sourceFaultDTO struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

type jobDTO struct {
	ID          string     `json:"id"`
	Game        string     `json:"game,omitempty"`
	Status      string     `json:"status"`
	Total       int        `json:"total"`
	Processed   int        `json:"processed"`
	CurrentCard string     `json:"current_card,omitempty"`
	Found       int        `json:"found"`
	NotFound    int        `json:"not_found"`
	Errors      int        `json:"errors"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	IncidentID  string     `json:"incident_id,omitempty"`
}

type itemDTO struct {
	ID             string           `json:"id"`
	SearchID       string           `json:"search_id"`
	Game           string           `json:"game"`
	Position       int              `json:"position"`
	Sequence       int              `json:"sequence"`
	OriginalName   string           `json:"original_name"`
	NormalizedName string           `json:"normalized_name"`
	Quantity       int              `json:"quantity"`
	Status         string           `json:"status"`
	Attempts       int              `json:"attempts"`
	Source         string           `json:"source,omitempty"`
	ErrorCode      string           `json:"error_code,omitempty"`
	ErrorMessage   string           `json:"error_message,omitempty"`
	Offers         []offerDTO       `json:"offers"`
	VerifyStock    bool             `json:"verify_stock,omitempty"`
	Match          string           `json:"match,omitempty"`
	Kind           string           `json:"kind,omitempty"`
	Faults         []sourceFaultDTO `json:"faults,omitempty"`
}

type resultDTO struct {
	SearchID string    `json:"search_id"`
	Items    []itemDTO `json:"items"`
	Cursor   int       `json:"cursor"`
	HasMore  bool      `json:"has_more"`
}

type sourceHealthDTO struct {
	Source                        string     `json:"source"`
	Platform                      string     `json:"platform,omitempty"`
	Enabled                       bool       `json:"enabled"`
	EstimatedResponseMilliseconds int64      `json:"estimated_response_ms"`
	LastSuccess                   *time.Time `json:"last_success,omitempty"`
	LastFailure                   *time.Time `json:"last_failure,omitempty"`
	ConsecutiveFailures           int        `json:"consecutive_failures"`
	LatencyMilliseconds           int64      `json:"latency_ms"`
	CircuitOpenUntil              *time.Time `json:"circuit_open_until,omitempty"`
}

type cartRequestDTO struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
}

type checkoutLineDTO struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
	URL      string `json:"url"`
}

type storeCheckoutDTO struct {
	Store      string            `json:"store"`
	Domain     string            `json:"domain"`
	Mode       string            `json:"mode"`
	URL        string            `json:"url,omitempty"`
	Lines      []checkoutLineDTO `json:"lines"`
	Quote      *cartQuoteDTO     `json:"quote,omitempty"`
	QuoteError string            `json:"quote_error,omitempty"`
}

type orderDTO struct {
	ID         string            `json:"id"`
	SearchID   string            `json:"search_id"`
	Store      string            `json:"store"`
	Domain     string            `json:"domain"`
	Lines      []checkoutLineDTO `json:"lines"`
	Status     string            `json:"status"`
	StoreOrder string            `json:"store_order,omitempty"`
	PaymentURL string            `json:"payment_url,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type offerStockDTO struct {
	ID string `json:"id"`
	stockReadingDTO
}

// stockRequest Names the Offers the Caller Wants Looked at again.

type stockRequest struct {
	Offers []string `json:"offers"`
}

// checkoutRequest Names the Offers the Buyer Chose and how many of each.

type checkoutRequest struct {
	Items    []cartRequestDTO    `json:"items"`
	Shipping *shippingAddressDTO `json:"shipping,omitempty"`
}

// orderRequest Names the Offers to Place a real Order for, all at the same

// Store. Shipping is Required here where checkoutRequest Leaves it optional:

// a Quote can Skip the Store, an Order cannot.

// linkRequest Names the Lines a Buyer is about to Take to a Store's own Cart.

type linkRequest struct {
	Items []cartRequestDTO `json:"items"`
}

// reportRequest Carries the Order Number the Store Showed the Buyer.

type reportRequest struct {
	StoreOrder string `json:"store_order"`
}

type orderRequest struct {
	Items    []cartRequestDTO   `json:"items"`
	Shipping shippingAddressDTO `json:"shipping"`
}

type refreshInventoryReply struct {
	StoreID   string `json:"store_id"`
	List      string `json:"list,omitempty"`
	Refreshed int    `json:"refreshed"`
}

// createSearchBody is the untrusted HTTP shape of a search request.

type createSearchBody struct {
	Game    string            `json:"game"`
	Cards   []searchCardBody  `json:"cards"`
	Options searchOptionsBody `json:"options"`
}

type searchCardBody struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type searchOptionsBody struct {
	VerifyStock bool   `json:"verify_stock"`
	Match       string `json:"match"`
	Kind        string `json:"kind"`
}

// offerQuery is the HTTP shape before match and kind become business values.

type offerQuery struct {
	Game  string
	Name  string
	Match string
	Kind  string
}

type wooCommerceOrderWebhook struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}
