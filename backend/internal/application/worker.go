package application

import (
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// BuildSearchWorker Gives every runtime the same search worker configuration.
func BuildSearchWorker(runtime Runtime, settings config.Config) search.Worker {
	return search.Worker{
		Repository:      runtime.Vault,
		Offers:          runtime.Service.OfferService,
		Stocks:          runtime.Service.Stocks,
		Owner:           settings.WorkerID,
		LeaseDuration:   settings.LeaseDuration,
		PollInterval:    settings.PollInterval,
		StockCheckLimit: settings.StockCheckLimit,
	}
}
