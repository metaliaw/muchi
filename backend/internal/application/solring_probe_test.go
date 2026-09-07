package application

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/scry"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

type openGate struct{}

func (openGate) AwaitSource(context.Context, string) error { return nil }
func (openGate) RecordSource(context.Context, string, time.Duration, error) error {
	return nil
}

type memoryCache struct {
	mutex sync.Mutex
	items map[string][]offer.Offer
}

func (c *memoryCache) LoadOffers(_ context.Context, key string) ([]offer.Offer, bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	items, found := c.items[key]
	return items, found, nil
}

func (c *memoryCache) SaveOffers(_ context.Context, key string, items []offer.Offer, _ time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.items == nil {
		c.items = map[string][]offer.Offer{}
	}
	c.items[key] = items
	return nil
}

func TestSolRingProbe(t *testing.T) {
	if os.Getenv("MUCHI_LIVE") == "" {
		t.Skip("set MUCHI_LIVE=1")
	}
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	fetcher := source.Client{
		HTTP: &http.Client{Timeout: 20 * time.Second}, Gate: openGate{},
		UserAgent: userAgent, MaxAttempts: 2, BaseDelay: 250 * time.Millisecond, MaxBodyBytes: 4 << 20,
	}
	catalog := scry.Client{Fetcher: fetcher, BaseURL: "https://scry.cl"}
	cache := &memoryCache{}
	for _, run := range []struct {
		label   string
		catalog search.OfferSource
	}{{"CON SCRY", catalog}, {"SIN SCRY", nil}} {
		sources := buildOfferSources(fetcher, run.catalog, config, cache, time.Hour, nil)
		type outcome struct {
			counts map[string]int
			label  string
			failed string
		}
		results := make([]outcome, len(sources))
		var group sync.WaitGroup
		for index, one := range sources {
			group.Add(1)
			go func(index int, one search.OfferSource) {
				defer group.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				found := outcome{counts: map[string]int{}, label: fmt.Sprintf("%T", one)}
				items, err := one.FindOffers(ctx, "Sol Ring")
				if err != nil {
					found.failed = err.Error()
				}
				for _, item := range items {
					found.counts[item.Store]++
				}
				results[index] = found
			}(index, one)
		}
		group.Wait()
		counts := map[string]int{}
		total := 0
		fmt.Printf("\n== %s == fuentes=%d\n", run.label, len(sources))
		for _, found := range results {
			if found.failed != "" {
				fmt.Printf("  falla %s: %s\n", found.label, found.failed)
			}
			for name, count := range found.counts {
				counts[name] += count
				total += count
			}
		}
		names := make([]string, 0, len(counts))
		for name := range counts {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool { return counts[names[i]] > counts[names[j]] })
		fmt.Printf("  TOTAL ofertas=%d tiendas=%d\n", total, len(counts))
		for _, name := range names {
			fmt.Printf("  %4d  %s\n", counts[name], name)
		}
	}
}
