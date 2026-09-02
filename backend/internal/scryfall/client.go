package scryfall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

type Client struct {
	Fetcher source.Client
	BaseURL string
}

type cardReply struct {
	Name     string            `json:"name"`
	Purchase map[string]string `json:"purchase_uris"`
	Prices   map[string]string `json:"prices"`
	Language string            `json:"lang"`
	Finishes []string          `json:"finishes"`
	OracleID string            `json:"oracle_id"`
}

func (c Client) FindOffers(ctx context.Context, name string) ([]offer.Offer, error) {
	target := c.BaseURL + "/cards/named?exact=" + url.QueryEscape(name)
	data, err := c.Fetcher.FetchSource(ctx, "api.scryfall.com", target)
	if err != nil {
		var status source.StatusError
		if errors.As(err, &status) && status.Code == 404 {
			return []offer.Offer{}, nil
		}
		return nil, err
	}
	var reply cardReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, fmt.Errorf("decode Scryfall response: %w", err)
	}
	return buildOffers(reply), nil
}

func buildOffers(reply cardReply) []offer.Offer {
	result := make([]offer.Offer, 0, len(reply.Purchase))
	for store, link := range reply.Purchase {
		amount := reply.Prices["usd"]
		if amount == "" {
			continue
		}
		result = append(result, offer.Offer{
			ID: reply.OracleID + ":" + store, CardName: reply.Name,
			Store: store, PriceAmount: amount, PriceCurrency: "USD",
			URL: link, Language: reply.Language, Source: "scryfall",
		})
	}
	return result
}
