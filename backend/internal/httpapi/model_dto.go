package httpapi

import (
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

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

func renderOffer(value model.Offer) offerDTO {
	return offerDTO{
		ID:               value.ID,
		CardName:         value.CardName,
		Store:            value.Store,
		PriceAmount:      value.PriceAmount,
		PriceCurrency:    value.PriceCurrency,
		URL:              value.URL,
		Image:            value.Image,
		VariantID:        value.VariantID,
		Language:         value.Language,
		Condition:        value.Condition,
		Finish:           value.Finish,
		Edition:          value.Edition,
		Kind:             string(value.Kind),
		Locations:        value.Locations,
		CardKey:          value.CardKey,
		Source:           value.Source,
		StockStatus:      value.StockStatus,
		StockQuantity:    value.StockQuantity,
		Suspicious:       value.Suspicious,
		SuspiciousReason: value.SuspiciousReason,
		Metadata:         value.Metadata,
	}
}
func renderOfferList(values []model.Offer) []offerDTO {
	if values == nil {
		return nil
	}
	result := make([]offerDTO, len(values))
	for i, value := range values {
		result[i] = renderOffer(value)
	}
	return result
}

type stockReadingDTO struct {
	Status   string `json:"stock_status"`
	Quantity *int   `json:"stock_quantity,omitempty"`
}

func renderStockReading(value model.StockReading) stockReadingDTO {
	return stockReadingDTO{
		Status:   value.Status,
		Quantity: value.Quantity,
	}
}

type shippingAddressDTO struct {
	Country  string `json:"country"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	Postcode string `json:"postcode,omitempty"`
}

func buildShippingAddress(value shippingAddressDTO) model.ShippingAddress {
	return model.ShippingAddress{
		Country:  value.Country,
		Region:   value.Region,
		City:     value.City,
		Postcode: value.Postcode,
	}
}

func buildShippingAddressPointer(value *shippingAddressDTO) *model.ShippingAddress {
	if value == nil {
		return nil
	}
	result := buildShippingAddress(*value)
	return &result
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

func renderCartQuote(value model.CartQuote) cartQuoteDTO {
	return cartQuoteDTO{
		Currency:       value.Currency,
		Items:          value.Items,
		Shipping:       value.Shipping,
		Total:          value.Total,
		Lines:          renderQuoteLineList(value.Lines),
		ShippingRates:  renderShippingRateList(value.ShippingRates),
		PaymentMethods: value.PaymentMethods,
		Notices:        value.Notices,
	}
}

func renderCartQuotePointer(value *model.CartQuote) *cartQuoteDTO {
	if value == nil {
		return nil
	}
	result := renderCartQuote(*value)
	return &result
}

type quoteLineDTO struct {
	OfferID   string `json:"offer_id"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

func renderQuoteLine(value model.QuoteLine) quoteLineDTO {
	return quoteLineDTO{
		OfferID:   value.OfferID,
		Quantity:  value.Quantity,
		UnitPrice: value.UnitPrice,
	}
}
func renderQuoteLineList(values []model.QuoteLine) []quoteLineDTO {
	if values == nil {
		return nil
	}
	result := make([]quoteLineDTO, len(values))
	for i, value := range values {
		result[i] = renderQuoteLine(value)
	}
	return result
}

type shippingRateDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    string `json:"price"`
	Selected bool   `json:"selected"`
}

func renderShippingRate(value model.ShippingRate) shippingRateDTO {
	return shippingRateDTO{
		ID:       value.ID,
		Name:     value.Name,
		Price:    value.Price,
		Selected: value.Selected,
	}
}
func renderShippingRateList(values []model.ShippingRate) []shippingRateDTO {
	if values == nil {
		return nil
	}
	result := make([]shippingRateDTO, len(values))
	for i, value := range values {
		result[i] = renderShippingRate(value)
	}
	return result
}

type sourceFaultDTO struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

func renderSourceFault(value model.SourceFault) sourceFaultDTO {
	return sourceFaultDTO{
		Source: value.Source,
		Reason: value.Reason,
	}
}
func renderSourceFaultList(values []model.SourceFault) []sourceFaultDTO {
	if values == nil {
		return nil
	}
	result := make([]sourceFaultDTO, len(values))
	for i, value := range values {
		result[i] = renderSourceFault(value)
	}
	return result
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

func renderJob(value model.Job) jobDTO {
	return jobDTO{
		ID:          value.ID,
		Game:        string(value.Game),
		Status:      string(value.Status),
		Total:       value.Total,
		Processed:   value.Processed,
		CurrentCard: value.CurrentCard,
		Found:       value.Found,
		NotFound:    value.NotFound,
		Errors:      value.Errors,
		CreatedAt:   value.CreatedAt,
		StartedAt:   value.StartedAt,
		UpdatedAt:   value.UpdatedAt,
		FinishedAt:  value.FinishedAt,
		IncidentID:  value.IncidentID,
	}
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

func renderItem(value model.Item) itemDTO {
	return itemDTO{
		ID:             value.ID,
		SearchID:       value.SearchID,
		Game:           string(value.Game),
		Position:       value.Position,
		Sequence:       value.Sequence,
		OriginalName:   value.OriginalName,
		NormalizedName: value.NormalizedName,
		Quantity:       value.Quantity,
		Status:         string(value.Status),
		Attempts:       value.Attempts,
		Source:         value.Source,
		ErrorCode:      value.ErrorCode,
		ErrorMessage:   value.ErrorMessage,
		Offers:         renderOfferList(value.Offers),
		VerifyStock:    value.VerifyStock,
		Match:          string(value.Match),
		Kind:           string(value.Kind),
		Faults:         renderSourceFaultList(value.Faults),
	}
}
func renderItemList(values []model.Item) []itemDTO {
	if values == nil {
		return nil
	}
	result := make([]itemDTO, len(values))
	for i, value := range values {
		result[i] = renderItem(value)
	}
	return result
}

type resultDTO struct {
	SearchID string    `json:"search_id"`
	Items    []itemDTO `json:"items"`
	Cursor   int       `json:"cursor"`
	HasMore  bool      `json:"has_more"`
}

func renderResult(value model.Result) resultDTO {
	return resultDTO{
		SearchID: value.SearchID,
		Items:    renderItemList(value.Items),
		Cursor:   value.Cursor,
		HasMore:  value.HasMore,
	}
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

func renderSourceHealth(value model.SourceHealth) sourceHealthDTO {
	return sourceHealthDTO{
		Source:                        value.Source,
		Platform:                      value.Platform,
		Enabled:                       value.Enabled,
		EstimatedResponseMilliseconds: value.EstimatedResponseMilliseconds,
		LastSuccess:                   value.LastSuccess,
		LastFailure:                   value.LastFailure,
		ConsecutiveFailures:           value.ConsecutiveFailures,
		LatencyMilliseconds:           value.LatencyMilliseconds,
		CircuitOpenUntil:              value.CircuitOpenUntil,
	}
}
func renderSourceHealthList(values []model.SourceHealth) []sourceHealthDTO {
	if values == nil {
		return nil
	}
	result := make([]sourceHealthDTO, len(values))
	for i, value := range values {
		result[i] = renderSourceHealth(value)
	}
	return result
}

type cartRequestDTO struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
}

func buildCartRequest(value cartRequestDTO) model.CartRequest {
	return model.CartRequest{
		OfferID:  value.OfferID,
		Quantity: value.Quantity,
	}
}
func buildCartRequestList(values []cartRequestDTO) []model.CartRequest {
	if values == nil {
		return nil
	}
	result := make([]model.CartRequest, len(values))
	for i, value := range values {
		result[i] = buildCartRequest(value)
	}
	return result
}

type checkoutLineDTO struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
	URL      string `json:"url"`
}

func renderCheckoutLine(value model.CheckoutLine) checkoutLineDTO {
	return checkoutLineDTO{
		OfferID:  value.OfferID,
		Quantity: value.Quantity,
		URL:      value.URL,
	}
}
func renderCheckoutLineList(values []model.CheckoutLine) []checkoutLineDTO {
	if values == nil {
		return nil
	}
	result := make([]checkoutLineDTO, len(values))
	for i, value := range values {
		result[i] = renderCheckoutLine(value)
	}
	return result
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

func renderStoreCheckout(value model.StoreCheckout) storeCheckoutDTO {
	return storeCheckoutDTO{
		Store:      value.Store,
		Domain:     value.Domain,
		Mode:       value.Mode,
		URL:        value.URL,
		Lines:      renderCheckoutLineList(value.Lines),
		Quote:      renderCartQuotePointer(value.Quote),
		QuoteError: value.QuoteError,
	}
}
func renderStoreCheckoutList(values []model.StoreCheckout) []storeCheckoutDTO {
	if values == nil {
		return nil
	}
	result := make([]storeCheckoutDTO, len(values))
	for i, value := range values {
		result[i] = renderStoreCheckout(value)
	}
	return result
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

func renderOrder(value model.Order) orderDTO {
	return orderDTO{
		ID:         value.ID,
		SearchID:   value.SearchID,
		Store:      value.Store,
		Domain:     value.Domain,
		Lines:      renderCheckoutLineList(value.Lines),
		Status:     string(value.Status),
		StoreOrder: value.StoreOrder,
		PaymentURL: value.PaymentURL,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
	}
}

type offerStockDTO struct {
	ID string `json:"id"`
	stockReadingDTO
}

func renderOfferStock(value model.OfferStock) offerStockDTO {
	return offerStockDTO{
		ID:              value.ID,
		stockReadingDTO: renderStockReading(value.StockReading),
	}
}
func renderOfferStockList(values []model.OfferStock) []offerStockDTO {
	if values == nil {
		return nil
	}
	result := make([]offerStockDTO, len(values))
	for i, value := range values {
		result[i] = renderOfferStock(value)
	}
	return result
}
