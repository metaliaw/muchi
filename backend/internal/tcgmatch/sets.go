package tcgmatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SetCatalog Lists the Sets one Game Has, and Remembers them for a While.
//
// A Set List Changes when a Set Releases, a few Times a Year, and every Sealed
// Search Reads it. Asking the Catalog once per Question would Spend a Request
// on an Answer that Barely Moves.
type SetCatalog struct {
	Fetcher SourceFetcher
	BaseURL string
	Game    string
	TTL     time.Duration

	guard   sync.Mutex
	names   []string
	readAt  time.Time
	lastErr error
}

type setsReply struct {
	Sets []struct {
		Name string `json:"name"`
		TCG  string `json:"tcg"`
	} `json:"sets"`
}

// GameSets Answers the Set Names of this Game, from Memory when it Can.
func (c *SetCatalog) GameSets(ctx context.Context) ([]string, error) {
	c.guard.Lock()
	defer c.guard.Unlock()
	if c.names != nil && time.Since(c.readAt) < c.TTL {
		return c.names, nil
	}
	names, err := c.readSets(ctx)
	if err != nil {
		// A stale List Answers better than no List: the Sets it Misses are the
		// ones Released since it was Read.
		if c.names != nil {
			return c.names, nil
		}
		return nil, err
	}
	c.names, c.readAt = names, time.Now()
	return names, nil
}

func (c *SetCatalog) readSets(ctx context.Context) ([]string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || c.Game == "" {
		return nil, errors.New("invalid TCGMatch set catalog")
	}
	target := strings.TrimRight(c.BaseURL, "/") + "/sets?tcg=" + url.QueryEscape(c.Game)
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		return nil, err
	}
	var reply setsReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, fmt.Errorf("decode TCGMatch sets: %w", err)
	}
	if len(reply.Sets) == 0 {
		return nil, errors.New("TCGMatch answered no sets")
	}
	names := make([]string, 0, len(reply.Sets))
	for _, set := range reply.Sets {
		if set.Name != "" && set.TCG == c.Game {
			names = append(names, setAliases(set.Name)...)
		}
	}
	return names, nil
}

// minAlias Keeps a Tail long enough to Name a Set on its own.
const minAlias = 4

// setAliases Answers the Ways a Store may Write one Set.
//
// TCGMatch Leads a Set with its Code — `ME05: Pitch Black`, `SV09: Journey
// Together` — and a Store Writes only the Words: `Mega Evolution - Pitch
// Black`. Both Forms Name the Set, so both Belong to the List.
func setAliases(name string) []string {
	aliases := []string{name}
	if _, tail, found := strings.Cut(name, ": "); found && len(strings.TrimSpace(tail)) >= minAlias {
		aliases = append(aliases, strings.TrimSpace(tail))
	}
	return aliases
}
