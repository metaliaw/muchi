package scrycl

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Client struct {
	Fetcher SourceFetcher
	BaseURL string
	// ExcludeCommunity Drops the Offers Scry.cl hosts itself: a Community Seller
	// holds a page under the Scry.cl Domain, never a Storefront of its own.
	ExcludeCommunity bool
}

func (c Client) Search(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	return c.FindOffers(ctx, query)
}

// FindOffers Reads the Saved Offers Published on a Scry.cl Card Page.
func (c Client) FindOffers(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	name := query.Name
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid Scry.cl URL")
	}
	slug := buildCardSlug(name)
	if slug == "" {
		return nil, errors.New("empty Scry.cl card slug")
	}
	target := strings.TrimRight(c.BaseURL, "/") + "/card/" + slug
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		var status source.StatusError
		if errors.As(err, &status) && status.Code == 404 {
			return []model.Offer{}, nil
		}
		return nil, err
	}
	items, err := readOffers(data)
	if err != nil {
		return nil, err
	}
	if c.ExcludeCommunity {
		items = withoutCommunityOffers(items, base.Host)
	}
	return items, nil
}

// SourceName Identifies this Provider the Way the Health Report Names it.
func (c Client) SourceName() string {
	if base, err := url.Parse(c.BaseURL); err == nil && base.Host != "" {
		return base.Host
	}
	return c.BaseURL
}
