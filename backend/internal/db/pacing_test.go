package db

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

// TestALongFallIsAskedLessOften Covers the Wait Growing with the Fall.
//
// A Store down for a Week Received five Calls a Minute under the fixed
// Cooldown. Each one Cost its Timeout to the Search that Asked, and the fallen
// Store one more Knock.
func TestALongFallIsAskedLessOften(t *testing.T) {
	record := sourceRecord{}
	fall := errors.New("connection refused")
	waits := []time.Duration{}
	for range 12 {
		updateSource(&record, 10*time.Millisecond, fall)
		if record.CircuitOpenUntil != nil {
			waits = append(waits, time.Until(*record.CircuitOpenUntil).Round(time.Minute))
		}
	}

	if waits[0] != time.Minute {
		t.Errorf("first wait = %s, want a minute", waits[0])
	}
	if waits[3] != 8*time.Minute {
		t.Errorf("fourth wait = %s, want eight minutes", waits[3])
	}
	// La Ultima ya Toco el Techo: una Tienda que Vuelve no Espera mas que eso.
	if waits[len(waits)-1] != time.Hour {
		t.Errorf("last wait = %s, want an hour", waits[len(waits)-1])
	}
}

// TestTheCircuitNeverWaitsForeverEvenAfterManyFalls Covers the Ceiling.
func TestTheCircuitNeverWaitsForeverEvenAfterManyFalls(t *testing.T) {
	if wait := circuitWait(39); wait != time.Hour {
		t.Errorf("wait after 39 falls = %s, want an hour", wait)
	}
	if wait := circuitWait(5000); wait != time.Hour {
		t.Errorf("wait after 5000 falls = %s, want an hour", wait)
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
