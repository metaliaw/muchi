package db

import (
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type offerPayload struct {
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

func renderOffer(value model.Offer) offerPayload {
	return offerPayload{
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
func renderOfferList(values []model.Offer) []offerPayload {
	if values == nil {
		return nil
	}
	result := make([]offerPayload, len(values))
	for i, value := range values {
		result[i] = renderOffer(value)
	}
	return result
}

func buildOffer(value offerPayload) model.Offer {
	return model.Offer{
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
		Kind:             model.ProductKind(value.Kind),
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
func buildOfferList(values []offerPayload) []model.Offer {
	if values == nil {
		return nil
	}
	result := make([]model.Offer, len(values))
	for i, value := range values {
		result[i] = buildOffer(value)
	}
	return result
}

type jobPayload struct {
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

func renderJob(value model.Job) jobPayload {
	return jobPayload{
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

func buildJob(value jobPayload) model.Job {
	return model.Job{
		ID:          value.ID,
		Game:        model.Game(value.Game),
		Status:      model.JobStatus(value.Status),
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

type itemPayload struct {
	ID             string               `json:"id"`
	SearchID       string               `json:"search_id"`
	Game           string               `json:"game"`
	Position       int                  `json:"position"`
	Sequence       int                  `json:"sequence"`
	OriginalName   string               `json:"original_name"`
	NormalizedName string               `json:"normalized_name"`
	Quantity       int                  `json:"quantity"`
	Status         string               `json:"status"`
	Attempts       int                  `json:"attempts"`
	Source         string               `json:"source,omitempty"`
	ErrorCode      string               `json:"error_code,omitempty"`
	ErrorMessage   string               `json:"error_message,omitempty"`
	Offers         []offerPayload       `json:"offers"`
	VerifyStock    bool                 `json:"verify_stock,omitempty"`
	Match          string               `json:"match,omitempty"`
	Kind           string               `json:"kind,omitempty"`
	Faults         []sourceFaultPayload `json:"faults,omitempty"`
}

func renderItem(value model.Item) itemPayload {
	return itemPayload{
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

func buildItem(value itemPayload) model.Item {
	return model.Item{
		ID:             value.ID,
		SearchID:       value.SearchID,
		Game:           model.Game(value.Game),
		Position:       value.Position,
		Sequence:       value.Sequence,
		OriginalName:   value.OriginalName,
		NormalizedName: value.NormalizedName,
		Quantity:       value.Quantity,
		Status:         model.ItemStatus(value.Status),
		Attempts:       value.Attempts,
		Source:         value.Source,
		ErrorCode:      value.ErrorCode,
		ErrorMessage:   value.ErrorMessage,
		Offers:         buildOfferList(value.Offers),
		VerifyStock:    value.VerifyStock,
		Match:          model.MatchMode(value.Match),
		Kind:           model.ProductKind(value.Kind),
		Faults:         buildSourceFaultList(value.Faults),
	}
}

type checkoutLinePayload struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
	URL      string `json:"url"`
}

func renderCheckoutLines(values []model.CheckoutLine) []checkoutLinePayload {
	if values == nil {
		return nil
	}
	result := make([]checkoutLinePayload, len(values))
	for i, value := range values {
		result[i] = checkoutLinePayload{OfferID: value.OfferID, Quantity: value.Quantity, URL: value.URL}
	}
	return result
}

func buildCheckoutLines(values []checkoutLinePayload) []model.CheckoutLine {
	if values == nil {
		return nil
	}
	result := make([]model.CheckoutLine, len(values))
	for i, value := range values {
		result[i] = model.CheckoutLine{OfferID: value.OfferID, Quantity: value.Quantity, URL: value.URL}
	}
	return result
}

type orderPayload struct {
	ID         string                `json:"id"`
	SearchID   string                `json:"search_id"`
	Store      string                `json:"store"`
	Domain     string                `json:"domain"`
	Lines      []checkoutLinePayload `json:"lines"`
	Status     string                `json:"status"`
	StoreOrder string                `json:"store_order,omitempty"`
	PaymentURL string                `json:"payment_url,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

func renderOrder(value model.Order) orderPayload {
	return orderPayload{
		ID:         value.ID,
		SearchID:   value.SearchID,
		Store:      value.Store,
		Domain:     value.Domain,
		Lines:      renderCheckoutLines(value.Lines),
		Status:     string(value.Status),
		StoreOrder: value.StoreOrder,
		PaymentURL: value.PaymentURL,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
	}
}

func buildOrder(value orderPayload) model.Order {
	return model.Order{
		ID:         value.ID,
		SearchID:   value.SearchID,
		Store:      value.Store,
		Domain:     value.Domain,
		Lines:      buildCheckoutLines(value.Lines),
		Status:     model.OrderStatus(value.Status),
		StoreOrder: value.StoreOrder,
		PaymentURL: value.PaymentURL,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
	}
}

type sourceFaultPayload struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

func renderSourceFault(value model.SourceFault) sourceFaultPayload {
	return sourceFaultPayload{
		Source: value.Source,
		Reason: value.Reason,
	}
}
func renderSourceFaultList(values []model.SourceFault) []sourceFaultPayload {
	if values == nil {
		return nil
	}
	result := make([]sourceFaultPayload, len(values))
	for i, value := range values {
		result[i] = renderSourceFault(value)
	}
	return result
}

func buildSourceFault(value sourceFaultPayload) model.SourceFault {
	return model.SourceFault{
		Source: value.Source,
		Reason: value.Reason,
	}
}
func buildSourceFaultList(values []sourceFaultPayload) []model.SourceFault {
	if values == nil {
		return nil
	}
	result := make([]model.SourceFault, len(values))
	for i, value := range values {
		result[i] = buildSourceFault(value)
	}
	return result
}
