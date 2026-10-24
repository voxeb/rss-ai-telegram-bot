package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"
)

const (
	maxPerRun          = 5
	maxArticleAttempts = 5 // Includes the first attempt and expired processing leases.
	processingLease    = 5 * time.Minute
	retryBaseDelay     = 5 * time.Minute
)

type claimPass int

const (
	newArticles claimPass = iota
	retryArticles
)

// Missing fields in existing Firestore documents retain their zero values.
type articleState struct {
	Status     string    `firestore:"status"`
	Owner      string    `firestore:"owner"`
	Attempts   int64     `firestore:"attempts"`
	LeaseUntil time.Time `firestore:"leaseUntil"`
}

// prepareClaim is shared by the Firestore transaction and the in-memory tests.
// A changed state without an acquired claim retires an exhausted, expired lease.
func prepareClaim(s articleState, owner string, pass claimPass, now time.Time) (articleState, bool) {
	if s.Status == "published" || s.Status == "seeded" || s.Status == "failed" || s.LeaseUntil.After(now) {
		return s, false
	}
	if s.Attempts >= maxArticleAttempts {
		s.Status, s.Owner, s.LeaseUntil = "failed", "", now
		return s, false
	}
	// Legacy retry/processing records have no attempts field, but must still
	// yield to unseen articles. They receive a fresh bounded attempt budget.
	isRetry := s.Attempts > 0 || s.Status == "retry" || s.Status == "processing"
	if isRetry != (pass == retryArticles) {
		return s, false
	}
	s.Status, s.Owner = "processing", owner
	s.Attempts++
	s.LeaseUntil = now.Add(processingLease)
	return s, true
}

func completeClaim(s articleState, owner string, published bool, now time.Time) (articleState, error) {
	if s.Status != "processing" || owner == "" || s.Owner != owner {
		return s, errors.New("claim no longer owned")
	}
	s.Owner, s.LeaseUntil = "", now
	switch {
	case published:
		s.Status = "published"
	case s.Attempts >= maxArticleAttempts:
		s.Status = "failed"
	default:
		s.Status = "retry"
		// Attempts 1..4 wait 5, 10, 20, and 40 minutes respectively.
		delay := retryBaseDelay
		for attempt := int64(1); attempt < s.Attempts; attempt++ {
			delay *= 2
		}
		s.LeaseUntil = now.Add(delay)
	}
	return s, nil
}

// External operations are injected so queue behavior can be tested without
// credentials, real publications, or network calls.
type publicationWorker struct {
	claim     func(context.Context, entry, claimPass) (string, bool, error)
	finish    func(context.Context, entry, string, bool) error
	extract   func(context.Context, string) (string, error)
	summarize func(context.Context, string, string) (string, error)
	send      func(context.Context, entry, string) error
}

func (w publicationWorker) publish(ctx context.Context, e entry) error {
	body, err := w.extract(ctx, e.link())
	if err != nil {
		return fmt.Errorf("article extraction: %w", err)
	}
	summary, err := w.summarize(ctx, e.Title, body)
	if err != nil {
		return fmt.Errorf("Gemini: %w", err)
	}
	if err := w.send(ctx, e, summary); err != nil {
		return fmt.Errorf("Telegram: %w", err)
	}
	return nil
}

func processArticles(ctx context.Context, entries []entry, w publicationWorker) error {
	entries = append([]entry(nil), entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		return parsePublishedAt(entries[i].Updated).Before(parsePublishedAt(entries[j].Updated))
	})

	published, attempted := 0, 0
	var failures []error
	defer func() {
		log.Printf("Published %d new articles (attempted %d, failed %d)", published, attempted, len(failures))
	}()
	// Prefer first attempts regardless of publication date. Retries use only
	// the remaining budget, so poison articles cannot monopolize future runs.
	seen := make(map[string]bool)
	for _, pass := range []claimPass{newArticles, retryArticles} {
		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if attempted >= maxPerRun {
				break
			}
			if e.link() == "" || seen[key(e)] {
				continue
			}
			owner, acquired, err := w.claim(ctx, e, pass)
			if err != nil {
				return fmt.Errorf("claim %s: %w", e.link(), err)
			}
			if !acquired {
				continue
			}
			seen[key(e)] = true
			attempted++
			if err := w.publish(ctx, e); err != nil {
				log.Printf("Processing failed %s: %v", e.link(), err)
				failures = append(failures, fmt.Errorf("%s: %w", e.link(), err))
				if releaseErr := w.finish(ctx, e, owner, false); releaseErr != nil {
					return fmt.Errorf("processing error: %w; failed to release claim: %w", err, releaseErr)
				}
				continue
			}
			// Telegram accepted the message. If this Firestore write fails,
			// retrying after lease expiry can still produce a duplicate.
			if err := w.finish(ctx, e, owner, true); err != nil {
				return fmt.Errorf("published to Telegram but Firestore update failed: %w", err)
			}
			published++
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if attempted > 0 && published == 0 {
		return fmt.Errorf("all %d publication attempts failed: %w", attempted, errors.Join(failures...))
	}
	return nil
}
