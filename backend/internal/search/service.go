package search

// Service Joins the Services that Answer for a Search: each Owns the Ports and
// Limits its own Questions Need, and its Methods are the Service's own. The
// Callers hold one Service; a Test builds only the Part it Asks.
type Service struct {
	SearchService
	CardService
	OfferService
	StockService
	CheckoutService
	OrderService
}

// WithRepository Hands the one Repository to every Service that Reads a Search,
// so a Composition Root Names it once instead of four times.
func (s Service) WithRepository(repository SearchRepository) Service {
	s.SearchService.Repository = repository
	s.StockService.Repository = repository
	s.CheckoutService.Repository = repository
	s.OrderService.Repository = repository
	return s
}
