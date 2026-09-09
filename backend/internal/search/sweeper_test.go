package search

import (
	"context"
	"errors"
	"testing"
)

type fakeItems struct {
	waiting int
	err     error
	asked   []int
}

func (f *fakeItems) CountWaitingItems(_ context.Context, limit int) (int, error) {
	f.asked = append(f.asked, limit)
	if f.err != nil {
		return 0, f.err
	}
	return min(f.waiting, limit), nil
}

type fakeWaker struct {
	woken   []string
	failAt  int
	failure error
}

func (f *fakeWaker) WakeWorker(_ context.Context, reason string) error {
	if f.failure != nil && len(f.woken) == f.failAt {
		return f.failure
	}
	f.woken = append(f.woken, reason)
	return nil
}

func TestSweepWakesOncePerWaitingItem(t *testing.T) {
	items := &fakeItems{waiting: 3}
	queue := &fakeWaker{}

	woken, err := Sweeper{Items: items, Queue: queue, MaxWakes: 50}.SweepQueue(context.Background())

	if err != nil || woken != 3 || len(queue.woken) != 3 {
		t.Fatalf("woken=%d err=%v queue=%v", woken, err, queue.woken)
	}
}

func TestAnEmptyQueueWakesNobody(t *testing.T) {
	queue := &fakeWaker{}

	woken, err := Sweeper{Items: &fakeItems{}, Queue: queue, MaxWakes: 50}.SweepQueue(context.Background())

	if err != nil || woken != 0 || len(queue.woken) != 0 {
		t.Fatalf("an idle Sweep must stay quiet: woken=%d err=%v queue=%v", woken, err, queue.woken)
	}
}

func TestOneSweepIsCapped(t *testing.T) {
	// A Sweeper that fires hundreds of Turns at once would beat the Sources
	// harder than any Search ever does.
	items := &fakeItems{waiting: 900}
	queue := &fakeWaker{}

	woken, err := Sweeper{Items: items, Queue: queue, MaxWakes: 5}.SweepQueue(context.Background())

	if err != nil || woken != 5 || len(queue.woken) != 5 {
		t.Fatalf("woken=%d err=%v", woken, err)
	}
	if items.asked[0] != 5 {
		t.Fatalf("the Cap must reach the Count too, asked=%v", items.asked)
	}
}

func TestASweepWithoutACapStillWakesOne(t *testing.T) {
	queue := &fakeWaker{}

	woken, err := Sweeper{Items: &fakeItems{waiting: 9}, Queue: queue}.SweepQueue(context.Background())

	if err != nil || woken != 1 {
		t.Fatalf("a missing Cap must not mean an unbounded Sweep: woken=%d err=%v", woken, err)
	}
}

func TestASweepThatFailsHalfwaySaysHowFarItGot(t *testing.T) {
	broken := errors.New("cloud tasks is down")
	queue := &fakeWaker{failAt: 2, failure: broken}

	woken, err := Sweeper{Items: &fakeItems{waiting: 5}, Queue: queue, MaxWakes: 50}.SweepQueue(context.Background())

	if !errors.Is(err, broken) || woken != 2 {
		t.Fatalf("woken=%d err=%v", woken, err)
	}
}

func TestACountThatFailsWakesNobody(t *testing.T) {
	broken := errors.New("firestore is down")
	queue := &fakeWaker{}

	_, err := Sweeper{Items: &fakeItems{err: broken}, Queue: queue, MaxWakes: 50}.SweepQueue(context.Background())

	if !errors.Is(err, broken) || len(queue.woken) != 0 {
		t.Fatalf("err=%v queue=%v", err, queue.woken)
	}
}
