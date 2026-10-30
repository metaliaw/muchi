package search

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

func queryGameSources(ctx context.Context, provider Provider, hasProvider bool, sources []OfferSource, query model.CardQuery) ([]model.Offer, []model.SourceFault, error) {
	var items []model.Offer
	var faults []model.SourceFault
	var lastErr error
	if hasProvider {
		found, err := provider.Search(ctx, query)
		items = append(items, found...)
		if err != nil {
			faults = append(faults, model.SourceFault{Source: provider.SourceName(), Reason: err.Error()})
			lastErr = err
		}
	}
	if len(sources) > 0 {
		found, storeFaults, direct, err := querySources(ctx, sources, query)
		items = preferDirectOffers(items, direct)
		items = append(items, found...)
		faults = append(faults, storeFaults...)
		if err != nil {
			lastErr = err
		}
	}
	return items, faults, lastErr
}

// preferDirectOffers Drops Aggregated Offers when their Store Answered Directly.
func preferDirectOffers(items []model.Offer, direct map[string]bool) []model.Offer {
	kept := items[:0]
	for _, item := range items {
		if direct[readOfferHost(item)] {
			continue
		}
		kept = append(kept, item)
	}
	return kept
}

func readOfferHost(item model.Offer) string {
	link, err := url.Parse(item.URL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(link.Hostname()), "www.")
}

func querySources(ctx context.Context, sources []OfferSource, query model.CardQuery) ([]model.Offer, []model.SourceFault, map[string]bool, error) {
	results := make([][]model.Offer, len(sources))
	errors := make([]error, len(sources))
	turns := make(chan struct{}, maxConcurrentSources)
	var group sync.WaitGroup
	for index, source := range sources {
		group.Add(1)
		go querySource(ctx, &group, turns, source, query, &results[index], &errors[index])
	}
	group.Wait()
	items, faults, err := combineSources(sources, results, errors)
	return items, faults, readDirectHosts(sources, errors), err
}

func readDirectHosts(sources []OfferSource, errors []error) map[string]bool {
	result := make(map[string]bool)
	for index, source := range sources {
		name := strings.ToLower(source.SourceName())
		if errors[index] == nil && strings.Contains(name, ".") && !strings.ContainsAny(name, ":/") {
			result[strings.TrimPrefix(name, "www.")] = true
		}
	}
	return result
}

func querySource(ctx context.Context, group *sync.WaitGroup, turns chan struct{}, source OfferSource, query model.CardQuery, items *[]model.Offer, sourceErr *error) {
	defer group.Done()
	select {
	case turns <- struct{}{}:
		defer func() { <-turns }()
	case <-ctx.Done():
		*sourceErr = ctx.Err()
		return
	}
	*items, *sourceErr = source.FindOffers(ctx, query)
}

// combineSources Keeps every Failure by Name. A Source that Fell while others
// Answered used to Vanish, and a partial Answer Looked exactly like a full one.
func combineSources(sources []OfferSource, results [][]model.Offer, errors []error) ([]model.Offer, []model.SourceFault, error) {
	var result []model.Offer
	var faults []model.SourceFault
	var lastErr error
	for index, items := range results {
		if errors[index] != nil {
			faults = append(faults, model.SourceFault{Source: sources[index].SourceName(), Reason: errors[index].Error()})
			lastErr = errors[index]
		}
		result = append(result, items...)
	}
	return result, faults, lastErr
}
