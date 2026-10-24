package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	nethtml "golang.org/x/net/html"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const feedURL = "https://www.theregister.com/headlines.atom"

type feed struct {
	Entries []entry `xml:"channel>item"`
}

type entry struct {
	ID      string `xml:"guid"`
	Title   string `xml:"title"`
	Summary string `xml:"description"`
	Updated string `xml:"pubDate"`
	URL     string `xml:"link"`
}

func (e entry) link() string {
	return e.URL
}
func key(e entry) string {
	h := sha256.Sum256([]byte(e.ID + "|" + e.link()))
	return hex.EncodeToString(h[:])
}

var client = &http.Client{Timeout: 25 * time.Second}

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RegisterNewsBot/1.0 (+https://t.me/register_news)")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s: %s", url, res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, 2<<20))
}
func articleText(ctx context.Context, url string) (string, error) {
	b, err := get(ctx, url)
	if err != nil {
		return "", err
	}

	root, err := nethtml.Parse(bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	// Find the actual article body, not recommended article cards.
	var chosen *nethtml.Node
	var findBody func(*nethtml.Node)
	findBody = func(n *nethtml.Node) {
		if chosen != nil {
			return
		}
		if n.Type == nethtml.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" {
					for _, c := range strings.Fields(a.Val) {
						if c == "bodytext" {
							chosen = n
							return
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findBody(c)
		}
	}
	findBody(root)

	if chosen == nil {
		return "", errors.New("article bodytext not found")
	}

	var paragraphs []string
	var collect func(*nethtml.Node)
	collect = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "aside", "form":
				return
			case "p", "h2":
				var sb strings.Builder
				var text func(*nethtml.Node)
				text = func(x *nethtml.Node) {
					if x.Type == nethtml.TextNode {
						sb.WriteString(x.Data)
						sb.WriteByte(' ')
					}
					for c := x.FirstChild; c != nil; c = c.NextSibling {
						text(c)
					}
				}
				text(n)
				value := strings.Join(strings.Fields(sb.String()), " ")
				if len(value) > 35 {
					paragraphs = append(paragraphs, value)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(chosen)

	result := strings.Join(paragraphs, "\n")
	if len(result) > 24000 {
		result = result[:24000]
	}
	if len(result) < 250 {
		return "", errors.New("article text too short")
	}
	return result, nil
}
func summarize(ctx context.Context, apiKey, title, body string) (string, error) {
	prompt := "Summarize this technology news article in English in 2–4 sentences, maximum 80 words. State the main news, key factual details, and practical implications. Do not invent facts, speculate, use introductory filler, or repeat the headline. Return only the summary.\n\nTITLE: " + title + "\n\nARTICLE:\n" + body
	payload := map[string]any{"contents": []any{map[string]any{"parts": []any{map[string]string{"text": prompt}}}}, "generationConfig": map[string]any{"temperature": 0.2, "maxOutputTokens": 300}}
	b, _ := json.Marshal(payload)
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash-lite:generateContent"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(res.Body, 65536))
	if res.StatusCode != 200 {
		return "", fmt.Errorf("Gemini HTTP %d: %s", res.StatusCode, string(out))
	}
	var decoded struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		return "", err
	}
	if len(decoded.Candidates) == 0 {
		return "", errors.New("Gemini returned no candidates")
	}
	var result strings.Builder
	for _, p := range decoded.Candidates[0].Content.Parts {
		result.WriteString(p.Text)
	}
	s := strings.TrimSpace(result.String())
	if len(s) < 40 {
		return "", errors.New("Gemini returned empty/short summary")
	}
	return s, nil
}
func send(ctx context.Context, token, channel string, e entry, summary string) error {
	msg := "<b>" + stdhtml.EscapeString(e.Title) + "</b>\n\n" + stdhtml.EscapeString(summary) + "\n\n<a href=\"" + stdhtml.EscapeString(e.link()) + "\">Read full article →</a>"
	payload, _ := json.Marshal(map[string]any{"chat_id": channel, "text": msg, "parse_mode": "HTML", "disable_web_page_preview": false})
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode != 200 {
		return fmt.Errorf("Telegram HTTP %d: %s", res.StatusCode, string(b))
	}
	return nil
}
func env(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing environment variable %s", k)
	}
	return v
}

// parsePublishedAt uses the actual RFC1123-style RSS pubDate, not lexical order.
func parsePublishedAt(value string) time.Time {
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822} {
		if t, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return t
		}
	}
	return time.Time{}
}

// claimArticle uses a Firestore transaction so concurrent executions cannot
// process the same item while its lease is active.
func claimArticle(ctx context.Context, db *firestore.Client, e entry) (string, bool, error) {
	doc := db.Collection("articles").Doc(key(e))
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", false, err
	}
	owner := hex.EncodeToString(nonce[:])
	acquired := false
	err := db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		now := time.Now()
		if err == nil {
			data := snap.Data()
			state, _ := data["status"].(string)
			if state == "published" || state == "seeded" {
				return nil
			}
			if until, ok := data["leaseUntil"].(time.Time); ok && until.After(now) {
				return nil
			}
		}
		fields := map[string]any{
			"status": "processing", "owner": owner,
			"leaseUntil": now.Add(5 * time.Minute),
			"url":        e.link(), "title": e.Title,
			"updatedAt": firestore.ServerTimestamp,
		}
		if err == nil {
			if err := tx.Set(doc, fields, firestore.MergeAll); err != nil {
				return err
			}
		} else {
			if err := tx.Create(doc, fields); err != nil {
				return err
			}
		}
		acquired = true
		return nil
	})
	return owner, acquired, err
}

// finishArticle only changes a claim still owned by this execution.
func finishArticle(ctx context.Context, db *firestore.Client, e entry, owner string, published bool) error {
	doc := db.Collection("articles").Doc(key(e))
	return db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			return err
		}
		if snap.Data()["owner"] != owner {
			return fmt.Errorf("claim for %s no longer owned", e.link())
		}
		state := "retry"
		if published {
			state = "published"
		}
		fields := map[string]any{"status": state, "owner": "", "updatedAt": firestore.ServerTimestamp}
		if published {
			fields["leaseUntil"] = time.Now()
		} else {
			fields["leaseUntil"] = time.Now().Add(2 * time.Minute)
		}
		return tx.Set(doc, fields, firestore.MergeAll)
	})
}

func run(ctx context.Context) error {
	project := env("GOOGLE_CLOUD_PROJECT")
	token := env("TELEGRAM_TOKEN")
	gemini := env("GEMINI_API_KEY")
	channel := env("TELEGRAM_CHANNEL")
	db, err := firestore.NewClient(ctx, project)
	if err != nil {
		return err
	}
	defer db.Close()

	b, err := get(ctx, feedURL)
	if err != nil {
		return err
	}
	var f feed
	if err := xml.Unmarshal(b, &f); err != nil {
		return err
	}
	if len(f.Entries) == 0 {
		return errors.New("no RSS items found")
	}
	sort.SliceStable(f.Entries, func(i, j int) bool {
		return parsePublishedAt(f.Entries[i].Updated).Before(parsePublishedAt(f.Entries[j].Updated))
	})

	meta := db.Collection("settings").Doc("feed")
	_, err = meta.Get(ctx)
	if status.Code(err) == codes.NotFound {
		batch := db.Batch()
		for _, e := range f.Entries {
			if e.link() == "" {
				continue
			}
			batch.Set(db.Collection("articles").Doc(key(e)), map[string]any{
				"status": "seeded", "url": e.link(), "created": firestore.ServerTimestamp,
			})
		}
		batch.Set(meta, map[string]any{"initialized": firestore.ServerTimestamp})
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
		log.Printf("Initialized: skipped %d existing articles", len(f.Entries))
		return nil
	}
	if err != nil {
		return err
	}

	published, attempted := 0, 0
	const maxPerRun = 5
	for _, e := range f.Entries {
		if attempted >= maxPerRun {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e.link() == "" {
			continue
		}
		owner, acquired, err := claimArticle(ctx, db, e)
		if err != nil {
			return err
		}
		if !acquired {
			continue
		}
		attempted++
		body, err := articleText(ctx, e.link())
		if err == nil {
			var summary string
			summary, err = summarize(ctx, gemini, e.Title, body)
			if err == nil {
				err = send(ctx, token, channel, e, summary)
			}
		}
		if err != nil {
			log.Printf("Processing failed %s: %v", e.link(), err)
			if releaseErr := finishArticle(ctx, db, e, owner, false); releaseErr != nil {
				return fmt.Errorf("processing error: %v; failed to release claim: %w", err, releaseErr)
			}
			continue
		}
		// Telegram accepted the message. If this Firestore write fails,
		// retrying after lease expiry can produce a duplicate.
		if err := finishArticle(ctx, db, e, owner, true); err != nil {
			return fmt.Errorf("published to Telegram but Firestore update failed: %w", err)
		}
		published++
	}
	log.Printf("Published %d new articles (attempted %d)", published, attempted)
	return nil
}

func testRun(ctx context.Context) error {
	b, err := get(ctx, feedURL)
	if err != nil {
		return err
	}

	var f feed
	if err := xml.Unmarshal(b, &f); err != nil {
		return err
	}
	if len(f.Entries) == 0 {
		return errors.New("RSS feed is empty")
	}

	e := f.Entries[0]
	log.Printf("Testing article: %s", e.Title)

	body, err := articleText(ctx, e.link())
	if err != nil {
		return fmt.Errorf("article extraction: %w", err)
	}
	log.Printf("Extracted %d bytes", len(body))

	summary, err := summarize(
		ctx,
		env("GEMINI_API_KEY"),
		e.Title,
		body,
	)
	if err != nil {
		return fmt.Errorf("Gemini: %w", err)
	}

	log.Printf("Summary: %s", summary)

	if err := send(
		ctx,
		env("TELEGRAM_TOKEN"),
		env("TELEGRAM_CHANNEL"),
		e,
		"[TEST] "+summary,
	); err != nil {
		return fmt.Errorf("Telegram: %w", err)
	}

	log.Println("Test published successfully")
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		110*time.Second,
	)
	defer cancel()

	var err error
	if os.Getenv("TEST_MODE") == "true" {
		err = testRun(ctx)
	} else {
		err = run(ctx)
	}

	if err != nil {
		log.Fatal(err)
	}
}
