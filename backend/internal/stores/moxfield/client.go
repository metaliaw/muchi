package moxfield

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// SourceFetcher Reads Published Inventory through the Shared Traffic Gate.
type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type InventoryCache interface {
	LoadOffers(context.Context, string) ([]model.Offer, bool, error)
	SaveOffers(context.Context, string, []model.Offer, time.Duration) error
	DropOffers(context.Context, string) error
}

// Client Resolves one Configured List when its Cached Inventory Expires.
// Construct once per Runtime; do not Copy after Use.
type Client struct {
	Fetcher SourceFetcher
	Cache   InventoryCache
	Logger  *slog.Logger
	TTL     time.Duration
	Store   string
	StoreID string
	Label   string
	ListURL string
	Rate    int
	mutex   sync.Mutex
}

var deckIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ExtractListID(value string) (string, error) {
	link, err := url.Parse(value)
	if err != nil || link.Scheme != "https" || (link.Host != "moxfield.com" && link.Host != "www.moxfield.com") || link.User != nil {
		return "", fmt.Errorf("invalid Moxfield list URL %q", value)
	}
	parts := strings.Split(strings.Trim(link.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "decks" || !deckIDPattern.MatchString(parts[1]) {
		return "", fmt.Errorf("invalid Moxfield list URL %q", value)
	}
	return parts[1], nil
}

func (c *Client) FindOffers(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	id, err := ExtractListID(c.ListURL)
	if err != nil {
		return nil, err
	}
	items, err := c.loadInventory(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Moxfield list %s (%s): %w", c.Label, id, err)
	}
	result := make([]model.Offer, 0)
	for _, item := range items {
		if query.AcceptsTitle(item.CardName) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (c *Client) loadInventory(ctx context.Context, id string) ([]model.Offer, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := c.inventoryKey(id)
	if c.Cache != nil {
		if items, found, err := c.Cache.LoadOffers(ctx, key); err == nil && found {
			return items, nil
		}
	}
	items, err := c.fetchInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Cache != nil && c.TTL > 0 {
		if err := c.Cache.SaveOffers(ctx, key, items, c.TTL); err != nil && c.Logger != nil {
			c.Logger.WarnContext(ctx, "Inventory Cache Failed", "list", id, "error", err)
		}
	}
	return items, nil
}

func (c *Client) fetchInventory(ctx context.Context, id string) ([]model.Offer, error) {
	data, err := c.Fetcher.FetchSource(ctx, "api2.moxfield.com", "https://api2.moxfield.com/v3/decks/all/"+id)
	if err != nil {
		return nil, err
	}
	items, missing, err := c.readInventory(data, id)
	if missing > 0 && c.Logger != nil {
		c.Logger.WarnContext(ctx, "Inventory Prices Missing", "list", id, "count", missing)
	}
	return items, err
}

// DropInventory Forgets the Cached List, so the next Search Reads it from Moxfield.
// A Store that Adds a Card cannot Wait for the TTL to Notice it.
func (c *Client) DropInventory(ctx context.Context) error {
	id, err := ExtractListID(c.ListURL)
	if err != nil {
		return err
	}
	if c.Cache == nil {
		return nil
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.Cache.DropOffers(ctx, c.inventoryKey(id))
}

// inventoryKey Binds the Cached List to every Field that Shapes its Offers.
func (c *Client) inventoryKey(id string) string {
	return fmt.Sprintf("moxfield:v1:%x", sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%d", c.StoreID, c.Store, c.Label, id, c.Rate))))
}

// SourceName Names the List, not the Store: one Store may Hold several.
func (c *Client) SourceName() string { return c.StoreID + ":" + c.Label }
