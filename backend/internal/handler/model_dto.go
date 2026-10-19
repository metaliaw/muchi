package handler

import (
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

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

func renderStockReading(value model.StockReading) stockReadingDTO {
	return stockReadingDTO{
		Status:   value.Status,
		Quantity: value.Quantity,
	}
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

func renderResult(value model.Result) resultDTO {
	return resultDTO{
		SearchID: value.SearchID,
		Items:    renderItemList(value.Items),
		Cursor:   value.Cursor,
		HasMore:  value.HasMore,
	}
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
