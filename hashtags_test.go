package main

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestParseFeedCategories(t *testing.T) {
	tests := []struct {
		name, categories string
		want             []string
	}{
		{"missing", "", nil},
		{"empty", "<category/>", []string{""}},
		{"single", "<category>security</category>", []string{"security"}},
		{"ordered", "<category>unknown</category><category>science</category><category>security</category>", []string{"unknown", "science", "security"}},
		{"CDATA and whitespace", "<category><![CDATA[ \tAI  and\nML ]]></category>", []string{" \tAI  and\nML "}},
		{"XML entities", "<category>unknown &amp; &lt;literal&gt;</category>", []string{"unknown & <literal>"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(`<rss version="2.0"><channel><category>channel category</category><item><guid>stable-id</guid><title>Headline</title><link>https://example.com/article</link>` + tc.categories + `</item><item><guid>other</guid><title>Other</title><link>https://example.com/other</link></item></channel></rss>`)
			f, err := parseFeed(data)
			if err != nil || len(f.Entries) != 2 {
				t.Fatalf("parseFeed: entries=%d, err=%v", len(f.Entries), err)
			}
			e := f.Entries[0]
			if !reflect.DeepEqual(e.Categories, tc.want) || f.Entries[1].Categories != nil {
				t.Fatalf("categories=%q, other=%q; want %q, nil", e.Categories, f.Entries[1].Categories, tc.want)
			}
			if key(e) != key(entry{ID: "stable-id", URL: "https://example.com/article"}) {
				t.Fatal("RSS categories changed article identity")
			}
		})
	}
}

func TestCategoryHashtagDictionaryAndNormalization(t *testing.T) {
	tests := []struct{ category, want string }{
		{"ai and ml", "#AI"},
		{"security", "#Security"},
		{"cyber-crime", "#Security"},
		{"systems", "#Systems"},
		{"personal tech", "#PersonalTech"},
		{"virtualization", "#Virtualization"},
		{"networks", "#Networking"},
		{"public sector", "#PublicSector"},
		{"os platforms", "#OperatingSystems"},
		{"devops", "#DevOps"},
		{"software", "#Software"},
		{"applications", "#Applications"},
		{"science", "#Science"},
		{"columnists", "#Opinion"},
		{"offbeat", "#Offbeat"},
		{"bofh", "#BOFH"},
	}
	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			normalizedVariant := " \t\n" + strings.ReplaceAll(strings.ToUpper(tc.category), " ", " \t\r\n\u00a0 ") + "\u2003 "
			for _, input := range []string{tc.category, normalizedVariant} {
				if got := categoryHashtag([]string{input}); got != tc.want {
					t.Errorf("categoryHashtag(%q)=%q, want %q", input, got, tc.want)
				}
			}
		})
	}
}

func TestCategoryHashtagSelection(t *testing.T) {
	tests := []struct {
		name       string
		categories []string
		want       string
	}{
		{"missing", nil, ""},
		{"empty list", []string{}, ""},
		{"empty values", []string{"", " \t\n\u00a0"}, ""},
		{"unknown", []string{"new section", "<b>security</b>", "security, cyber-crime", "cyber crime"}, ""},
		{"skip unknown and empty", []string{"new section", "", " ScIeNcE ", "security"}, "#Science"},
		{"RSS order", []string{"software", "ai and ml"}, "#Software"},
		{"reversed RSS order", []string{"ai and ml", "software"}, "#AI"},
		{"same result", []string{"cyber-crime", "security"}, "#Security"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := categoryHashtag(tc.categories); got != tc.want {
				t.Errorf("categoryHashtag(%q)=%q, want %q", tc.categories, got, tc.want)
			}
		})
	}
}

func TestPublicationHashtags(t *testing.T) {
	const articleURL = "https://example.com/article?a=1&b=2"
	const summary = "A synthetic summary with <literal> text & enough detail for the publication test."
	tests := []struct{ name, categories, paragraph string }{
		{"recognized", "<category>  AI \t and\n ML </category>", "\n\n#AI"},
		{"first recognized", "<category>new section</category><category/><category>science</category><category>security</category>", "\n\n#Science"},
		{"missing", "", ""},
		{"empty", "<category> \t\n </category>", ""},
		{"unknown", "<category>&lt;script&gt;unknown&lt;/script&gt;</category>", ""},
	}
	for _, mode := range []string{"normal", "test"} {
		for _, tc := range tests {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				t.Setenv("TELEGRAM_TOKEN", fakeTelegramToken)
				t.Setenv("TELEGRAM_CHANNEL", "@test")
				t.Setenv("GEMINI_API_KEY", "fake-gemini-key")
				data := []byte(`<rss><channel><item><guid>test</guid><title><![CDATA[News &amp; &lt;literal&gt;]]></title><link>https://example.com/article?a=1&amp;b=2</link>` + tc.categories + `</item></channel></rss>`)
				geminiCalls, telegramCalls := 0, 0
				stubHTTP(t, func(req *http.Request) (*http.Response, error) {
					switch {
					case req.URL.String() == feedURL:
						return httpResponse(200, string(data)), nil
					case req.URL.String() == articleURL:
						return httpResponse(200, `<div class="bodytext"><p>`+strings.Repeat("This article contains enough text to extract and summarize. ", 8)+`</p></div>`), nil
					case req.URL.Host == "generativelanguage.googleapis.com":
						geminiCalls++
						return httpResponse(200, `{"candidates":[{"content":{"parts":[{"text":"`+summary+`"}]}}]}`), nil
					case req.URL.Host == "api.telegram.org":
						telegramCalls++
						var payload struct {
							Text                  string `json:"text"`
							ParseMode             string `json:"parse_mode"`
							ChatID                string `json:"chat_id"`
							DisableWebPagePreview bool   `json:"disable_web_page_preview"`
						}
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						prefix := ""
						if mode == "test" {
							prefix = "[TEST] "
						}
						want := "<b>News &amp; &lt;literal&gt;</b>\n\n" + prefix + "A synthetic summary with &lt;literal&gt; text &amp; enough detail for the publication test." + tc.paragraph + "\n\n<a href=\"https://example.com/article?a=1&amp;b=2\">Read full article →</a>"
						if payload.Text != want || payload.ParseMode != "HTML" || payload.ChatID != "@test" || payload.DisableWebPagePreview {
							t.Errorf("unexpected Telegram payload: %+v; want text=%q", payload, want)
						}
						return httpResponse(200, `{"ok":true}`), nil
					default:
						t.Fatalf("unexpected request to %s", req.URL.Host)
						return nil, nil
					}
				})
				if mode == "test" {
					if err := testRun(context.Background()); err != nil {
						t.Fatal(err)
					}
				} else {
					f, err := parseFeed(data)
					if err != nil {
						t.Fatal(err)
					}
					q := newMemoryQueue()
					w := q.worker()
					w.extract = articleText
					w.summarize = func(ctx context.Context, title, body string) (string, error) {
						return summarize(ctx, "fake-gemini-key", title, body)
					}
					w.send = func(ctx context.Context, e entry, summary string) error {
						return send(ctx, fakeTelegramToken, "@test", e, summary)
					}
					if err := processArticles(context.Background(), f.Entries, w); err != nil {
						t.Fatal(err)
					}
					if q.states[key(f.Entries[0])].Status != "published" {
						t.Fatal("article was not marked published")
					}
				}
				if geminiCalls != 1 || telegramCalls != 1 {
					t.Fatalf("Gemini calls=%d, Telegram calls=%d; want one each", geminiCalls, telegramCalls)
				}
			})
		}
	}
}
