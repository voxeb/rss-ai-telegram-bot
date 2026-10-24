package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

const fakeTelegramToken = "123456789:FAKE_TOKEN_FOR_TESTS_ONLY"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func stubHTTP(t *testing.T, transport roundTripFunc) {
	t.Helper()
	previous := client
	client = &http.Client{Transport: transport}
	t.Cleanup(func() { client = previous })
}

func httpResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func assertRedacted(t *testing.T, err error, token string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a publication error")
	}
	for _, formatted := range []string{err.Error(), fmt.Sprintf("%+v", err), fmt.Sprintf("%#v", err)} {
		if strings.Contains(formatted, token) || strings.Contains(formatted, "https://api.telegram.org/bot"+token) {
			t.Fatalf("error exposes synthetic credential: %s", formatted)
		}
	}
	// A caller must not recover a credential-bearing cause by unwrapping.
	switch e := err.(type) {
	case interface{ Unwrap() []error }:
		for _, cause := range e.Unwrap() {
			assertRedacted(t, cause, token)
		}
	case interface{ Unwrap() error }:
		if cause := e.Unwrap(); cause != nil {
			assertRedacted(t, cause, token)
		}
	}
}

func TestSendRedactsErrors(t *testing.T) {
	tests := []struct {
		name    string
		cause   error
		wantIs  error
		timeout bool
		want    string
	}{
		{name: "transport", cause: errors.New("connection failed"), want: "connection failed"},
		{name: "nested URL", cause: fmt.Errorf("connection failed for https://api.telegram.org/bot%s/sendMessage", fakeTelegramToken), want: "connection failed"},
		{name: "cancellation", cause: fmt.Errorf("%s: %w", fakeTelegramToken, context.Canceled), wantIs: context.Canceled},
		{name: "context deadline", cause: context.DeadlineExceeded, wantIs: context.DeadlineExceeded, timeout: true},
		{name: "network timeout", cause: os.ErrDeadlineExceeded, wantIs: os.ErrDeadlineExceeded, timeout: true},
		{name: "HTTP response", want: "HTTP 502"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			stubHTTP(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost || req.URL.Path != "/bot"+fakeTelegramToken+"/sendMessage" {
					t.Fatal("unexpected Telegram request")
				}
				if tc.cause != nil {
					return nil, tc.cause
				}
				return httpResponse(http.StatusBadGateway, "failed request: "+req.URL.String()), nil
			})
			err := send(context.Background(), fakeTelegramToken, "@test", testEntries(1)[0], "summary")
			assertRedacted(t, err, fakeTelegramToken)
			if calls != 1 || !strings.Contains(err.Error(), "sendMessage") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("error=%v, want errors.Is(%v)", err, tc.wantIs)
			}
			var timeout interface{ Timeout() bool }
			gotTimeout := errors.As(err, &timeout) && timeout.Timeout()
			if gotTimeout != tc.timeout {
				t.Fatalf("timeout=%t, want %t", gotTimeout, tc.timeout)
			}
		})
	}
}

func TestSendRedactsRequestCreationError(t *testing.T) {
	stubHTTP(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached transport")
		return nil, nil
	})
	token := fakeTelegramToken + "%"
	err := send(context.Background(), token, "@test", testEntries(1)[0], "summary")
	assertRedacted(t, err, fakeTelegramToken)
}

func TestSendSuccess(t *testing.T) {
	calls := 0
	stubHTTP(t, func(req *http.Request) (*http.Response, error) {
		calls++
		return httpResponse(http.StatusOK, `{"ok":true}`), nil
	})
	if err := send(context.Background(), fakeTelegramToken, "@test", testEntries(1)[0], "summary"); err != nil {
		t.Fatalf("successful send returned %v", err)
	}
	if calls != 1 {
		t.Fatalf("sent %d requests, want one", calls)
	}
}

func TestTelegramFailureLogsAreRedacted(t *testing.T) {
	for _, mode := range []string{"queue", "test publication"} {
		t.Run(mode, func(t *testing.T) {
			var logs bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&logs)
			t.Cleanup(func() { log.SetOutput(previous) })
			e := testEntries(1)[0]
			telegramCalls := 0
			stubHTTP(t, func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.String() == feedURL:
					return httpResponse(200, `<rss><channel><item><guid>test</guid><title>Test article</title><link>`+e.URL+`</link></item></channel></rss>`), nil
				case req.URL.String() == e.URL:
					return httpResponse(200, `<div class="bodytext"><p>`+strings.Repeat("This article contains enough text to extract and summarize. ", 8)+`</p></div>`), nil
				case req.URL.Host == "generativelanguage.googleapis.com":
					return httpResponse(200, `{"candidates":[{"content":{"parts":[{"text":"A synthetic summary long enough for the publication test."}]}}]}`), nil
				case req.URL.Host == "api.telegram.org":
					telegramCalls++
					return nil, fmt.Errorf("connection failed for %s", req.URL)
				default:
					t.Fatalf("unexpected request to %s", req.URL.Host)
					return nil, nil
				}
			})
			var err error
			if mode == "queue" {
				q := newMemoryQueue()
				worker := q.worker()
				worker.send = func(ctx context.Context, e entry, summary string) error {
					return send(ctx, fakeTelegramToken, "@test", e, summary)
				}
				err = processArticles(context.Background(), []entry{e}, worker)
				if q.states[key(e)].Status != "retry" || !strings.Contains(logs.String(), "Processing failed") {
					t.Fatal("queue did not record and log the failed attempt")
				}
			} else {
				t.Setenv("TELEGRAM_TOKEN", fakeTelegramToken)
				t.Setenv("TELEGRAM_CHANNEL", "@test")
				t.Setenv("GEMINI_API_KEY", "fake-gemini-key")
				err = testRun(context.Background())
			}
			assertRedacted(t, err, fakeTelegramToken)
			log.Printf("%v", err) // The same formatting as main's log.Fatal, without exiting.
			if telegramCalls != 1 || strings.Contains(logs.String(), fakeTelegramToken) || !strings.Contains(logs.String(), "connection failed") {
				t.Fatalf("Telegram calls=%d logs=%s", telegramCalls, logs.String())
			}
		})
	}
}
