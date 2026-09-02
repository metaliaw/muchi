package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchRetriesTransientStatus(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), MaxAttempts: 3, BaseDelay: time.Millisecond}
	data, err := client.FetchSource(context.Background(), "test", server.URL)
	if err != nil || string(data) != "ok" || attempts.Load() != 3 {
		t.Fatalf("FetchSource() data=%q attempts=%d err=%v", data, attempts.Load(), err)
	}
}

func TestFetchSetsSourceHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "muchi-test/1.0" || !strings.Contains(r.Header.Get("Accept"), "application/json") {
			t.Fatalf("unexpected headers: %v", r.Header)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), UserAgent: "muchi-test/1.0", MaxAttempts: 1}
	if _, err := client.FetchSource(context.Background(), "test", server.URL); err != nil {
		t.Fatal(err)
	}
}

func TestFetchStopsPermanentStatus(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), MaxAttempts: 3, BaseDelay: time.Millisecond}
	_, err := client.FetchSource(context.Background(), "test", server.URL)
	if err == nil || attempts.Load() != 1 {
		t.Fatalf("FetchSource() attempts=%d err=%v", attempts.Load(), err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	if got := ParseRetryAfter("2"); got != 2*time.Second {
		t.Fatalf("ParseRetryAfter() = %s", got)
	}
}
