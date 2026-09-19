package firestore

import (
	"errors"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// TestAThrottledSourceIsNotAFallenOne Covers what a 429 Means for Health.
//
// A Store Answering 429 is Awake and Well: it is Saying the Calls Arrive too
// Fast. Counting it as a Fall Opened its Circuit after five, and then nobody
// Asked it anything for a Minute — punishing a Store for Being careful.
func TestAThrottledSourceIsNotAFallenOne(t *testing.T) {
	record := sourceRecord{}
	throttled := source.StatusError{Code: 429, Body: "slow down"}
	for range 6 {
		updateSource(&record, 10*time.Millisecond, throttled)
	}

	if record.ConsecutiveFailures != 0 {
		t.Errorf("consecutive failures = %d, want none", record.ConsecutiveFailures)
	}
	if record.CircuitOpenUntil != nil {
		t.Error("a throttled store had its circuit opened")
	}
	if record.LastFailure == nil {
		t.Error("the throttling went unrecorded")
	}
}

// TestARealFailureStillOpensTheCircuit Covers the Half that must not Move.
func TestARealFailureStillOpensTheCircuit(t *testing.T) {
	record := sourceRecord{}
	for range 5 {
		updateSource(&record, 10*time.Millisecond, errors.New("connection refused"))
	}

	if record.ConsecutiveFailures != 5 {
		t.Errorf("consecutive failures = %d, want 5", record.ConsecutiveFailures)
	}
	if record.CircuitOpenUntil == nil {
		t.Error("five real failures left the circuit closed")
	}
}

// TestASuccessClearsWhatCameBefore Covers the Reset a Good Answer Brings.
func TestASuccessClearsWhatCameBefore(t *testing.T) {
	record := sourceRecord{ConsecutiveFailures: 4}
	updateSource(&record, time.Millisecond, nil)

	if record.ConsecutiveFailures != 0 || record.CircuitOpenUntil != nil {
		t.Errorf("record = %#v", record)
	}
}

// TestPacingFallsBackToItsDefault Covers a Store nobody Tuned.
func TestPacingFallsBackToItsDefault(t *testing.T) {
	store := &Store{}
	if store.pace() != sourcePacing || store.cool() != throttleCooldown {
		t.Errorf("pace = %v cool = %v", store.pace(), store.cool())
	}
	store.PaceSources(time.Second, 2*time.Minute)
	if store.pace() != time.Second || store.cool() != 2*time.Minute {
		t.Errorf("pace = %v cool = %v", store.pace(), store.cool())
	}
}
