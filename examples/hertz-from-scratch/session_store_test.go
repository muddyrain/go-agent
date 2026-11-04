package main

import (
	"sync"
	"testing"
)

func TestSessionStoreCountsAndIsolation(t *testing.T) {
	store := newSessionStore()
	for _, tt := range []struct {
		id   string
		want int
	}{
		{"a", 1},
		{"a", 2},
		{"b", 1},
		{"a", 3},
		{"b", 2},
	} {
		if got := store.nextTurn(tt.id); got != tt.want {
			t.Fatalf("nextTurn(%q)=%d want=%d", tt.id, got, tt.want)
		}
	}
	if got := newSessionStore().nextTurn("a"); got != 1 {
		t.Fatalf("separate stores must start independently: turn=%d", got)
	}
}

func TestSessionStoreConcurrentIncrements(t *testing.T) {
	store := newSessionStore()
	const count = 64
	turns := make(chan int, count)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			turns <- store.nextTurn("same-session")
		}()
	}
	close(start)
	wg.Wait()
	close(turns)

	seen := make(map[int]bool)
	for turn := range turns {
		if turn < 1 || turn > count || seen[turn] {
			t.Fatalf("invalid or duplicate turn=%d", turn)
		}
		seen[turn] = true
	}
	if len(seen) != count {
		t.Fatalf("unique turns=%d want=%d", len(seen), count)
	}
	if got := store.nextTurn("same-session"); got != count+1 {
		t.Fatalf("increments lost: next turn=%d want=%d", got, count+1)
	}
	if got := store.nextTurn("other-session"); got != 1 {
		t.Fatalf("different session count leaked: turn=%d", got)
	}
}
