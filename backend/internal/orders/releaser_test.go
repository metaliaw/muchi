package orders

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExpirer struct {
	olderThan time.Duration
	limit     int
	released  int
	err       error
}

func (f *fakeExpirer) ReleaseExpiredOrders(_ context.Context, olderThan time.Duration, limit int) (int, error) {
	f.olderThan, f.limit = olderThan, limit
	return f.released, f.err
}

// TestReleaseOrdersDefaultsAgeAndLimit Keeps a zero-value Releaser Safe to
// Run: an unset MaxAge or MaxReleases must not Release everything at once.
func TestReleaseOrdersDefaultsAgeAndLimit(t *testing.T) {
	expirer := &fakeExpirer{released: 3}
	released, err := Releaser{Orders: expirer}.ReleaseOrders(context.Background())
	if err != nil || released != 3 {
		t.Fatalf("released = %d, err = %v", released, err)
	}
	if expirer.olderThan != 24*time.Hour || expirer.limit != 50 {
		t.Fatalf("olderThan = %s, limit = %d", expirer.olderThan, expirer.limit)
	}
}

// TestReleaseOrdersHonorsConfiguredAgeAndLimit Keeps an explicit MaxAge and
// MaxReleases from being silently Replaced by the Defaults.
func TestReleaseOrdersHonorsConfiguredAgeAndLimit(t *testing.T) {
	expirer := &fakeExpirer{}
	releaser := Releaser{Orders: expirer, MaxAge: time.Hour, MaxReleases: 5}
	if _, err := releaser.ReleaseOrders(context.Background()); err != nil {
		t.Fatal(err)
	}
	if expirer.olderThan != time.Hour || expirer.limit != 5 {
		t.Fatalf("olderThan = %s, limit = %d", expirer.olderThan, expirer.limit)
	}
}

// TestReleaseOrdersPropagatesFailure Keeps a Query Failure from Reading like
// zero Orders were Due.
func TestReleaseOrdersPropagatesFailure(t *testing.T) {
	failure := errors.New("firestore unavailable")
	expirer := &fakeExpirer{err: failure}
	if _, err := (Releaser{Orders: expirer}).ReleaseOrders(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}
