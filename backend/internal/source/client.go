package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

var ErrCircuitOpen = errors.New("source circuit is open")

// bodyCapFallback keeps an unset MaxBodyBytes from truncating every response to nothing.
const bodyCapFallback = 4 << 20

// TrafficGate Coordinates calls through https://firebase.google.com/docs/firestore/manage-data/transactions.
type TrafficGate interface {
	AwaitSource(context.Context, string) error
	RecordSource(context.Context, string, time.Duration, error) error
}

type Client struct {
	HTTP         *http.Client
	Gate         TrafficGate
	Logger       *slog.Logger
	UserAgent    string
	MaxAttempts  int
	BaseDelay    time.Duration
	MaxBodyBytes int64
	storefront   bool
}

type StatusError struct {
	Code int
	Body string
}

func (e StatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("source returned HTTP %d", e.Code)
	}
	return fmt.Sprintf("source returned HTTP %d: %s", e.Code, e.Body)
}

func (c Client) FetchSource(ctx context.Context, domain, target string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= c.MaxAttempts; attempt++ {
		data, retryAfter, err := c.fetchAttempt(ctx, domain, target, attempt)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !CanRetry(err) || attempt == c.MaxAttempts {
			break
		}
		if err := waitRetry(ctx, retryAfter, c.BaseDelay, attempt); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c Client) fetchAttempt(ctx context.Context, domain, target string, attempt int) ([]byte, time.Duration, error) {
	if c.Gate != nil {
		if err := c.Gate.AwaitSource(ctx, domain); err != nil {
			return nil, 0, err
		}
	}
	started := time.Now()
	data, retry, err := c.sendRequest(ctx, target)
	if c.Gate != nil {
		_ = c.Gate.RecordSource(ctx, domain, time.Since(started), err)
	}
	if c.Logger != nil {
		c.Logger.InfoContext(ctx, "Source Request", "source", domain, "attempt", attempt, "latency_ms", time.Since(started).Milliseconds(), "error", err)
	}
	return data, retry, err
}

func (c Client) sendRequest(ctx context.Context, target string) ([]byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json, text/html;q=0.9")
	req.Header.Set("User-Agent", c.UserAgent)
	if c.storefront {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host+"/")
	}
	response, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	retry := ParseRetryAfter(response.Header.Get("Retry-After"))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, retry, StatusError{Code: response.StatusCode, Body: string(data)}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, c.bodyCap()))
	return data, retry, err
}

func (c Client) bodyCap() int64 {
	if c.MaxBodyBytes < 1 {
		return bodyCapFallback
	}
	return c.MaxBodyBytes
}

func CanRetry(err error) bool {
	if errors.Is(err, ErrCircuitOpen) {
		return false
	}
	var status StatusError
	if errors.As(err, &status) {
		return status.Code == http.StatusTooManyRequests || status.Code >= 500
	}
	return true
}

func ParseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	date, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	return max(time.Until(date), 0)
}

func waitRetry(ctx context.Context, retry, base time.Duration, attempt int) error {
	if retry <= 0 {
		factor := time.Duration(1 << (attempt - 1))
		retry = base*factor + time.Duration(rand.Int64N(int64(max(base, time.Millisecond))))
	}
	timer := time.NewTimer(retry)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// FetchStorefront Uses the Public Ajax Contract while Preserving Retries and Traffic Coordination.
func (c Client) FetchStorefront(ctx context.Context, domain, target string) ([]byte, error) {
	c.storefront = true
	return c.FetchSource(ctx, domain, target)
}
