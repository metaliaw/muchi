package tcgmatch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
)

const pokemonFunctionVersion = "pokemon-function-v1:"
const pokemonCatalogConcurrency = 4

type pokemonCardBrief struct {
	ID      string `json:"id"`
	LocalID string `json:"localId"`
}

type pokemonCard struct {
	ID      string         `json:"id"`
	LocalID string         `json:"localId"`
	Set     pokemonCardSet `json:"set"`
	Fields  map[string]any `json:"-"`
}

type pokemonCardSet struct {
	Name string `json:"name"`
}

// UnmarshalJSON Keeps the complete Card so the fingerprint can select every
// gameplay field without coupling its scalar types to one catalog release.
func (c *pokemonCard) UnmarshalJSON(data []byte) error {
	type card pokemonCard
	var value card
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &value.Fields); err != nil {
		return err
	}
	*c = pokemonCard(value)
	return nil
}

// namePokemonFunctions Resolves Printings, then gives equal mechanics one Key.
func (c Client) namePokemonFunctions(ctx context.Context, products []catalogProduct) {
	if c.PokemonCatalogURL == "" || len(products) == 0 {
		return
	}
	briefs, err := c.searchPokemonCards(ctx, ReadCardName(products[0].Name))
	if err != nil {
		return
	}
	cards := c.readPokemonCards(ctx, selectPokemonCards(briefs, products))
	functions := indexPokemonFunctions(cards)
	for index := range products {
		products[index].Function = functions[pokemonPrintKey(products[index].SetName, products[index].CardCode)]
	}
}

func (c Client) searchPokemonCards(ctx context.Context, name string) ([]pokemonCardBrief, error) {
	target := strings.TrimRight(c.PokemonCatalogURL, "/") + "/cards?name=" + url.QueryEscape(name)
	data, err := c.Fetcher.FetchSource(ctx, readHost(c.PokemonCatalogURL), target)
	if err != nil {
		return nil, err
	}
	var cards []pokemonCardBrief
	err = json.Unmarshal(data, &cards)
	return cards, err
}

func selectPokemonCards(cards []pokemonCardBrief, products []catalogProduct) []pokemonCardBrief {
	wanted := make(map[string]bool)
	for _, product := range products {
		wanted[readLocalID(product.CardCode)] = true
	}
	result := make([]pokemonCardBrief, 0, len(cards))
	for _, card := range cards {
		if wanted[readLocalID(card.LocalID)] {
			result = append(result, card)
		}
	}
	return result
}

func (c Client) readPokemonCards(ctx context.Context, briefs []pokemonCardBrief) []pokemonCard {
	cards := make([]pokemonCard, len(briefs))
	turns := make(chan struct{}, pokemonCatalogConcurrency)
	var group sync.WaitGroup
	for index, brief := range briefs {
		group.Add(1)
		go func() {
			defer group.Done()
			turns <- struct{}{}
			defer func() { <-turns }()
			cards[index], _ = c.readPokemonCard(ctx, brief.ID)
		}()
	}
	group.Wait()
	return cards
}

func (c Client) readPokemonCard(ctx context.Context, id string) (pokemonCard, error) {
	target := strings.TrimRight(c.PokemonCatalogURL, "/") + "/cards/" + url.PathEscape(id)
	data, err := c.Fetcher.FetchSource(ctx, readHost(c.PokemonCatalogURL), target)
	if err != nil {
		return pokemonCard{}, err
	}
	var card pokemonCard
	err = json.Unmarshal(data, &card)
	return card, err
}

func indexPokemonFunctions(cards []pokemonCard) map[string]string {
	result := make(map[string]string)
	for _, card := range cards {
		if card.ID == "" {
			continue
		}
		result[pokemonPrintKey(card.Set.Name, card.LocalID)] = fingerprintPokemonCard(card.Fields)
	}
	return result
}

func fingerprintPokemonCard(card map[string]any) string {
	fields := make(map[string]any)
	for _, name := range []string{
		"name", "category", "stage", "hp", "types", "evolveFrom",
		"abilities", "attacks", "effect", "weaknesses", "resistances", "retreat",
	} {
		if value, found := card[name]; found {
			fields[name] = value
		}
	}
	data, _ := json.Marshal(fields)
	sum := sha256.Sum256(data)
	return pokemonFunctionVersion + hex.EncodeToString(sum[:])
}

func pokemonPrintKey(set, number string) string {
	return normalizePokemonValue(set) + "|" + readLocalID(number)
}

func normalizePokemonValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "pokémon", "pokemon")
	if prefix, rest, found := strings.Cut(value, ":"); found && len(prefix) <= 4 {
		value = rest
	}
	if prefix, rest, found := strings.Cut(value, " - "); found && len(prefix) <= 4 {
		value = rest
	}
	value = strings.TrimSuffix(strings.TrimSpace(value), " base set")
	value = strings.TrimPrefix(value, "scarlet & violet ")
	return strings.Join(strings.Fields(value), " ")
}

func readLocalID(value string) string {
	value = strings.SplitN(value, "/", 2)[0]
	value = strings.TrimLeft(strings.TrimSpace(value), "0")
	if value == "" {
		return "0"
	}
	return strings.ToLower(value)
}

func readHost(value string) string {
	parsed, _ := url.Parse(value)
	return parsed.Host
}

// ReadCardName Drops the printing suffix before querying the card catalog.
func ReadCardName(value string) string {
	return strings.TrimSpace(strings.SplitN(value, " - ", 2)[0])
}
