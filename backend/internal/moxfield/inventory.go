package moxfield

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type cardReply struct {
	ID       string                     `json:"id"`
	Name     string                     `json:"name"`
	Set      string                     `json:"set"`
	Language string                     `json:"lang"`
	Prices   map[string]json.RawMessage `json:"prices"`
}

type entryReply struct {
	ID        string                     `json:"id"`
	Quantity  int                        `json:"quantity"`
	Finish    string                     `json:"finish"`
	IsFoil    bool                       `json:"isFoil"`
	Card      cardReply                  `json:"card"`
	Printings []entryReply               `json:"printingData"`
	Prices    map[string]json.RawMessage `json:"prices"`
}

type deckReply struct {
	Boards map[string]struct {
		Cards map[string]entryReply `json:"cards"`
	} `json:"boards"`
}

func (c *Client) readInventory(data []byte, id string) ([]offer.Offer, int, error) {
	var deck deckReply
	if err := json.Unmarshal(data, &deck); err != nil {
		return nil, 0, fmt.Errorf("decode inventory: %w", err)
	}
	if deck.Boards == nil || c.Rate < 1 {
		return nil, 0, errors.New("invalid Moxfield inventory or conversion rate")
	}
	items := make([]offer.Offer, 0)
	missing := 0
	for board, content := range deck.Boards {
		for key, entry := range content.Cards {
			for _, printing := range expandPrintings(entry) {
				if printing.Quantity <= 0 {
					continue
				}
				item, ok := c.buildOffer(printing, id, board+":"+key)
				if !ok {
					missing++
					continue
				}
				items = append(items, item)
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, missing, nil
}

// expandPrintings Uses each Printing's Price and Quantity, never the Aggregate twice.
func expandPrintings(entry entryReply) []entryReply {
	if len(entry.Printings) == 0 {
		return []entryReply{entry}
	}
	for i := range entry.Printings {
		printing := &entry.Printings[i]
		if printing.Card.Name == "" {
			printing.Card.Name = entry.Card.Name
		}
		if printing.Card.ID == "" {
			printing.Card.ID = printing.ID
		}
		if printing.Prices != nil {
			printing.Card.Prices = printing.Prices
		}
	}
	return entry.Printings
}

func (c *Client) buildOffer(entry entryReply, id, key string) (offer.Offer, bool) {
	finish, priceKey := selectFinish(entry)
	price, ok := convertPrice(entry.Card.Prices[priceKey], c.Rate)
	if !ok || entry.Card.Name == "" || entry.Card.ID == "" {
		return offer.Offer{}, false
	}
	variant := strings.Join([]string{id, key, entry.Card.ID, finish}, ":")
	store := c.Store
	if store == "" {
		store = c.StoreID
	}
	// The List Counts its Copies, so the Offer Carries that Count from the
	// Start: a Buyer Choosing four Copies Deserves to Know the List Holds two
	// before Asking anyone. It is the same Number the Check Reads again later.
	units := entry.Quantity
	return offer.Offer{
		ID: "moxfield:" + c.StoreID + ":" + variant, VariantID: variant,
		CardName: entry.Card.Name, Store: store, PriceAmount: price, PriceCurrency: "CLP",
		URL: "https://moxfield.com/decks/" + id, Source: "moxfield", Finish: finish,
		Language: entry.Card.Language, StockStatus: "available", StockQuantity: &units,
		Metadata: map[string]string{
			"list": c.Label, "set": entry.Card.Set, "quantity": strconv.Itoa(entry.Quantity),
			"ck_usd":         strings.Trim(string(entry.Card.Prices[priceKey]), `"`),
			"clp_per_ck_usd": strconv.Itoa(c.Rate),
		},
	}, true
}

func selectFinish(entry entryReply) (string, string) {
	switch entry.Finish {
	case "etched":
		return "etched", "ck_etched"
	case "foil":
		return "foil", "ck_foil"
	case "", "nonFoil":
		if entry.IsFoil {
			return "foil", "ck_foil"
		}
		return "nonfoil", "ck"
	default:
		return entry.Finish, ""
	}
}

// convertPrice Rounds CK USD × the List Rate to the Nearest Peso, half Up.
func convertPrice(raw json.RawMessage, rate int) (string, bool) {
	value := strings.Trim(string(raw), `"`)
	price, ok := new(big.Rat).SetString(value)
	if !ok || price.Sign() <= 0 || rate < 1 {
		return "", false
	}
	price.Mul(price, new(big.Rat).SetInt64(int64(rate)))
	price.Add(price, big.NewRat(1, 2))
	pesos := new(big.Int).Quo(price.Num(), price.Denom())
	return pesos.String(), pesos.Sign() > 0
}
