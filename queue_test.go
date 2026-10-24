package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

var errPublication = errors.New("publication unavailable")

type memoryQueue struct {
	now       time.Time
	states    map[string]articleState
	attempted []string
	sent      []string
}

func newMemoryQueue() *memoryQueue {
	return &memoryQueue{
		now:    time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC),
		states: make(map[string]articleState),
	}
}

func (q *memoryQueue) worker() publicationWorker {
	return publicationWorker{
		claim: func(ctx context.Context, e entry, pass claimPass) (string, bool, error) {
			if err := ctx.Err(); err != nil {
				return "", false, err
			}
			owner := fmt.Sprintf("owner-%d", len(q.attempted))
			next, acquired := prepareClaim(q.states[key(e)], owner, pass, q.now)
			q.states[key(e)] = next
			if acquired {
				q.attempted = append(q.attempted, e.ID)
			}
			return owner, acquired, nil
		},
		finish: func(ctx context.Context, e entry, owner string, published bool) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			next, err := completeClaim(q.states[key(e)], owner, published, q.now)
			if err == nil {
				q.states[key(e)] = next
			}
			return err
		},
		extract:   func(context.Context, string) (string, error) { return "article body", nil },
		summarize: func(context.Context, string, string) (string, error) { return "summary", nil },
		send: func(_ context.Context, e entry, _ string) error {
			q.sent = append(q.sent, e.ID)
			return nil
		},
	}
}

func testEntries(n int) []entry {
	entries := make([]entry, n)
	for i := range entries {
		entries[i] = entry{
			ID: fmt.Sprint(i), Title: fmt.Sprintf("article %d", i),
			URL:     fmt.Sprintf("https://example.com/%d", i),
			Updated: time.Date(2026, time.October, 24, 0, i, 0, 0, time.UTC).Format(time.RFC1123Z),
		}
	}
	return entries
}

// Reproduce the reported failure at each stage, preserving state between jobs.
func TestPoisonArticlesYieldToNewArticles(t *testing.T) {
	for _, stage := range []string{"extraction", "Gemini", "Telegram"} {
		t.Run(stage, func(t *testing.T) {
			q := newMemoryQueue()
			w := q.worker()
			entries := testEntries(6)
			newest := entries[5]
			switch stage {
			case "extraction":
				w.extract = func(_ context.Context, url string) (string, error) {
					if url != newest.URL {
						return "", errPublication
					}
					return "body", nil
				}
			case "Gemini":
				w.summarize = func(_ context.Context, title, body string) (string, error) {
					if title != newest.Title {
						return "", errPublication
					}
					return "summary", nil
				}
			case "Telegram":
				send := w.send
				w.send = func(ctx context.Context, e entry, summary string) error {
					if e.ID != newest.ID {
						return errPublication
					}
					return send(ctx, e, summary)
				}
			}
			// RSS arrives newest-first; first attempts remain chronological.
			for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
				entries[i], entries[j] = entries[j], entries[i]
			}
			if err := processArticles(context.Background(), entries, w); !errors.Is(err, errPublication) {
				t.Fatalf("first job error = %v, want publication failure", err)
			}
			if want := []string{"0", "1", "2", "3", "4"}; !reflect.DeepEqual(q.attempted, want) {
				t.Fatalf("first attempts = %v, want %v", q.attempted, want)
			}
			q.now = q.now.Add(retryBaseDelay) // All five poison articles are eligible again.
			q.attempted = nil
			if err := processArticles(context.Background(), entries, w); err != nil {
				t.Fatalf("partially successful job: %v", err)
			}
			if want := []string{"5", "0", "1", "2", "3"}; !reflect.DeepEqual(q.attempted, want) {
				t.Fatalf("second attempts = %v, want fresh article first: %v", q.attempted, want)
			}
			if !reflect.DeepEqual(q.sent, []string{"5"}) || q.states[key(newest)].Status != "published" {
				t.Fatalf("new article was not published: sent=%v state=%+v", q.sent, q.states[key(newest)])
			}
		})
	}
}

func TestAttemptLimitCountsFailuresAndSuccesses(t *testing.T) {
	for _, failing := range []bool{false, true} {
		t.Run(fmt.Sprintf("failing=%t", failing), func(t *testing.T) {
			q := newMemoryQueue()
			w := q.worker()
			if failing {
				w.extract = func(context.Context, string) (string, error) { return "", errPublication }
			}
			entries := testEntries(12)
			q.states[key(entries[0])] = articleState{Status: "seeded"}
			q.states[key(entries[1])] = articleState{Status: "published"}
			q.states[key(entries[2])] = articleState{Status: "failed", Attempts: 5}
			q.states[key(entries[3])] = articleState{Status: "processing", Owner: "other", Attempts: 1, LeaseUntil: q.now.Add(time.Minute)}
			q.states[key(entries[4])] = articleState{Status: "retry", Attempts: 1, LeaseUntil: q.now.Add(time.Minute)}
			input := append([]entry{{ID: "missing-link"}}, entries...)
			input = append(input, entries[5], entries[5])
			err := processArticles(context.Background(), input, w)
			if failing != errors.Is(err, errPublication) {
				t.Fatalf("error = %v, failing=%t", err, failing)
			}
			if want := []string{"5", "6", "7", "8", "9"}; !reflect.DeepEqual(q.attempted, want) {
				t.Fatalf("attempts = %v, want %v", q.attempted, want)
			}
			if q.states[key(entries[10])].Attempts != 0 || q.states[key(entries[11])].Attempts != 0 {
				t.Fatal("processed articles beyond the attempt limit")
			}
		})
	}
}

func TestRetryLifecycle(t *testing.T) {
	for _, succeedAt := range []int64{0, 3, maxArticleAttempts} {
		t.Run(fmt.Sprintf("succeedAt=%d", succeedAt), func(t *testing.T) {
			q := newMemoryQueue()
			w := q.worker()
			e := testEntries(1)[0]
			w.extract = func(context.Context, string) (string, error) {
				if q.states[key(e)].Attempts == succeedAt {
					return "body", nil
				}
				return "", errPublication
			}
			for attempt := int64(1); attempt <= maxArticleAttempts; attempt++ {
				err := processArticles(context.Background(), []entry{e}, w)
				s := q.states[key(e)]
				if s.Attempts != attempt || s.Owner != "" {
					t.Fatalf("attempt %d: state=%+v", attempt, s)
				}
				if attempt == succeedAt {
					if err != nil || s.Status != "published" {
						t.Fatalf("recovery: state=%+v err=%v", s, err)
					}
					break
				}
				if !errors.Is(err, errPublication) {
					t.Fatalf("attempt %d: error=%v", attempt, err)
				}
				if attempt == maxArticleAttempts {
					if s.Status != "failed" {
						t.Fatalf("retry budget not exhausted: %+v", s)
					}
					break
				}
				wantDelay := []time.Duration{5, 10, 20, 40}[attempt-1] * time.Minute
				if s.Status != "retry" || !s.LeaseUntil.Equal(q.now.Add(wantDelay)) {
					t.Fatalf("attempt %d: incorrect backoff: %+v", attempt, s)
				}
				q.now = s.LeaseUntil.Add(-time.Nanosecond)
				before := len(q.attempted)
				if err := processArticles(context.Background(), []entry{e}, w); err != nil || len(q.attempted) != before {
					t.Fatalf("backoff did not skip work: attempted=%v err=%v", q.attempted, err)
				}
				q.now = s.LeaseUntil
			}
			q.now = q.now.Add(24 * time.Hour)
			before := len(q.attempted)
			if err := processArticles(context.Background(), []entry{e}, w); err != nil || len(q.attempted) != before {
				t.Fatalf("terminal article retried: attempts=%v err=%v", q.attempted, err)
			}
		})
	}
}

func TestClaimsRespectLeasesHistoryAndRetryBudget(t *testing.T) {
	now := newMemoryQueue().now
	tests := []struct {
		name     string
		state    articleState
		pass     claimPass
		acquired bool
		retired  bool
	}{
		{name: "unseen", pass: newArticles, acquired: true},
		{name: "unseen not a retry", pass: retryArticles},
		{name: "legacy retry yields", state: articleState{Status: "retry"}, pass: newArticles},
		{name: "legacy retry eligible", state: articleState{Status: "retry"}, pass: retryArticles, acquired: true},
		{name: "legacy expired processing yields", state: articleState{Status: "processing"}, pass: newArticles},
		{name: "legacy expired processing eligible", state: articleState{Status: "processing"}, pass: retryArticles, acquired: true},
		{name: "live lease", state: articleState{Status: "processing", Attempts: 1, LeaseUntil: now.Add(time.Second)}, pass: retryArticles},
		{name: "backoff", state: articleState{Status: "retry", Attempts: 1, LeaseUntil: now.Add(time.Second)}, pass: retryArticles},
		{name: "lease boundary", state: articleState{Status: "processing", Attempts: 1, LeaseUntil: now}, pass: retryArticles, acquired: true},
		{name: "last attempt", state: articleState{Status: "retry", Attempts: 4}, pass: retryArticles, acquired: true},
		{name: "crashed final attempt", state: articleState{Status: "processing", Owner: "old", Attempts: 5}, pass: retryArticles, retired: true},
		{name: "live final attempt", state: articleState{Status: "processing", Owner: "old", Attempts: 5, LeaseUntil: now.Add(time.Second)}, pass: retryArticles},
		{name: "published", state: articleState{Status: "published"}, pass: newArticles},
		{name: "seeded", state: articleState{Status: "seeded"}, pass: newArticles},
		{name: "failed", state: articleState{Status: "failed", Attempts: 5}, pass: retryArticles},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, acquired := prepareClaim(tt.state, "new-owner", tt.pass, now)
			if acquired != tt.acquired {
				t.Fatalf("acquired=%t, want %t", acquired, tt.acquired)
			}
			switch {
			case acquired:
				if next.Attempts != tt.state.Attempts+1 || next.Status != "processing" || next.Owner != "new-owner" || !next.LeaseUntil.Equal(now.Add(processingLease)) {
					t.Fatalf("incorrect claim: %+v", next)
				}
			case tt.retired:
				if next.Status != "failed" || next.Owner != "" || next.Attempts != tt.state.Attempts {
					t.Fatalf("incorrect retirement: %+v", next)
				}
			default:
				if next != tt.state {
					t.Fatalf("skipped claim changed state: %+v -> %+v", tt.state, next)
				}
			}
		})
	}
}

func TestCompleteClaimRejectsStaleOwner(t *testing.T) {
	for _, published := range []bool{false, true} {
		s := articleState{Status: "processing", Owner: "current", Attempts: 2}
		if next, err := completeClaim(s, "stale", published, time.Now()); err == nil || next != s {
			t.Fatalf("stale owner changed state: %+v, err=%v", next, err)
		}
	}
}

func TestAtMostOneAttemptPerArticlePerRun(t *testing.T) {
	for _, previousAttempts := range []int64{0, 1} {
		t.Run(fmt.Sprintf("previousAttempts=%d", previousAttempts), func(t *testing.T) {
			q := newMemoryQueue()
			w := q.worker()
			e := testEntries(1)[0]
			if previousAttempts > 0 {
				q.states[key(e)] = articleState{Status: "retry", Attempts: previousAttempts}
			}
			w.extract = func(context.Context, string) (string, error) { return "", errPublication }
			finish := w.finish
			w.finish = func(ctx context.Context, e entry, owner string, published bool) error {
				err := finish(ctx, e, owner, published)
				q.now = q.now.Add(time.Hour) // Even if the retry delay expires within this run.
				return err
			}
			err := processArticles(context.Background(), []entry{e, e, e}, w)
			if !errors.Is(err, errPublication) || len(q.attempted) != 1 || q.states[key(e)].Attempts != previousAttempts+1 {
				t.Fatalf("duplicate attempt: attempted=%v state=%+v err=%v", q.attempted, q.states[key(e)], err)
			}
		})
	}
}

func TestProcessingErrorsRemainObservable(t *testing.T) {
	storageErr := errors.New("Firestore unavailable")
	for _, operation := range []string{"claim", "release", "mark published"} {
		t.Run(operation, func(t *testing.T) {
			q := newMemoryQueue()
			w := q.worker()
			switch operation {
			case "claim":
				w.claim = func(context.Context, entry, claimPass) (string, bool, error) { return "", false, storageErr }
			case "release":
				w.extract = func(context.Context, string) (string, error) { return "", errPublication }
				w.finish = func(context.Context, entry, string, bool) error { return storageErr }
			case "mark published":
				w.finish = func(context.Context, entry, string, bool) error { return storageErr }
			}
			err := processArticles(context.Background(), testEntries(2), w)
			if !errors.Is(err, storageErr) {
				t.Fatalf("lost storage error: %v", err)
			}
			if operation == "release" && !errors.Is(err, errPublication) {
				t.Fatalf("lost processing error: %v", err)
			}
			if operation == "mark published" && !strings.Contains(err.Error(), "published to Telegram") {
				t.Fatalf("missing delivery warning: %v", err)
			}
			if len(q.attempted) > 1 {
				t.Fatalf("continued after storage failure: %v", q.attempted)
			}
		})
	}
}

func TestNoEligibleArticlesSucceeds(t *testing.T) {
	q := newMemoryQueue()
	if err := processArticles(context.Background(), nil, q.worker()); err != nil {
		t.Fatal(err)
	}
	entries := testEntries(3)
	q.states[key(entries[0])] = articleState{Status: "published"}
	q.states[key(entries[1])] = articleState{Status: "retry", Attempts: 1, LeaseUntil: q.now.Add(time.Minute)}
	q.states[key(entries[2])] = articleState{Status: "failed", Attempts: 5}
	if err := processArticles(context.Background(), entries, q.worker()); err != nil || len(q.attempted) != 0 {
		t.Fatalf("no work: attempted=%v err=%v", q.attempted, err)
	}
}

func TestCancellationFailsJob(t *testing.T) {
	for _, when := range []string{"before work", "during extraction", "after final publication"} {
		t.Run(when, func(t *testing.T) {
			q := newMemoryQueue()
			w := q.worker()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch when {
			case "before work":
				cancel()
			case "during extraction":
				w.extract = func(context.Context, string) (string, error) { cancel(); return "", ctx.Err() }
			case "after final publication":
				finish := w.finish
				w.finish = func(ctx context.Context, e entry, owner string, published bool) error {
					err := finish(ctx, e, owner, published)
					cancel()
					return err
				}
			}
			if err := processArticles(ctx, testEntries(1), w); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
		})
	}
}

func TestPublicationStopsAtFailingStage(t *testing.T) {
	for _, stage := range []string{"extract", "summarize", "send"} {
		t.Run(stage, func(t *testing.T) {
			var calls []string
			call := func(name string) error {
				calls = append(calls, name)
				if name == stage {
					return errPublication
				}
				return nil
			}
			w := publicationWorker{
				extract:   func(context.Context, string) (string, error) { return "body", call("extract") },
				summarize: func(context.Context, string, string) (string, error) { return "summary", call("summarize") },
				send:      func(context.Context, entry, string) error { return call("send") },
			}
			if err := w.publish(context.Background(), testEntries(1)[0]); !errors.Is(err, errPublication) {
				t.Fatal(err)
			}
			want := map[string][]string{"extract": {"extract"}, "summarize": {"extract", "summarize"}, "send": {"extract", "summarize", "send"}}[stage]
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls=%v, want=%v", calls, want)
			}
		})
	}
}
