package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const reportedHeadline = `<span data-lab-italic_desktop="italic" class="italic m-italic">Reg</span> readers land on 'Binface' as name for UK space cleanup mission`
const cleanHeadline = `Reg readers land on 'Binface' as name for UK space cleanup mission`

func TestParseFeedHeadlines(t *testing.T) {
	tests := []struct {
		name, title, want string
	}{
		{"reported CDATA", "<![CDATA[" + reportedHeadline + "]]>", cleanHeadline},
		{"XML escaped markup", `&lt;span class="italic"&gt;Reg&lt;/span&gt; readers`, "Reg readers"},
		{"inline punctuation", `<![CDATA[pre<b>fix</b>: <i>Reg</i>'s <em>new <strong>idea</strong></em>]]>`, "prefix: Reg's new idea"},
		{"entities", `<![CDATA[AT&amp;T&nbsp;– &#39;Reg&#39; &#x1F680; &quot;news&quot;]]>`, "AT&T – 'Reg' 🚀 \"news\""},
		{"plain text", `Rock &amp; roll: 2 &lt; 3 and 5 &gt; 4 — café`, "Rock & roll: 2 < 3 and 5 > 4 — café"},
		{"encoded literal tag", `<![CDATA[Explain &lt;span&gt; &amp; &amp;lt; in HTML]]>`, "Explain <span> & &lt; in HTML"},
		{"whitespace", "<![CDATA[ \tReg\n  readers\u00a0 land\r\non Mars  ]]>", "Reg readers land on Mars"},
		{"line breaks", `<![CDATA[Reg<br>readers<br/>land<div>on Mars</div>today]]>`, "Reg readers land on Mars today"},
		{"ignored content", `<![CDATA[Reg<!-- comment --><script>bad()</script><style>.bad{}</style><template>hidden</template> readers]]>`, "Reg readers"},
		{"unclosed inline element", `<![CDATA[<span class="italic">Reg readers]]>`, "Reg readers"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(`<rss><channel><item><guid>stable-id</guid><title>` + tc.title + `</title><link>https://example.com/article?a=1&amp;b=2</link><pubDate>Sun, 25 Oct 2026 10:00:00 GMT</pubDate><description>Original description</description></item></channel></rss>`)
			var raw feed
			if err := xml.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}
			got, err := parseFeed(data)
			if err != nil || len(got.Entries) != 1 {
				t.Fatalf("parseFeed: entries=%d, err=%v", len(got.Entries), err)
			}
			want := raw.Entries[0]
			want.Title = tc.want
			if !reflect.DeepEqual(got.Entries[0], want) {
				t.Errorf("entry=%+v, want %+v", got.Entries[0], want)
			}
			if key(got.Entries[0]) != key(raw.Entries[0]) {
				t.Fatal("headline normalization changed article identity")
			}
		})
	}
}

func TestParseFeedInvalidXML(t *testing.T) {
	if _, err := parseFeed([]byte(`<rss><channel><item>`)); err == nil {
		t.Fatal("malformed XML accepted")
	}
}

func TestQueueReceivesNormalizedHeadline(t *testing.T) {
	f, err := parseFeed([]byte(`<rss><channel><item><guid>test</guid><title><![CDATA[` + reportedHeadline + `]]></title><link>https://example.com/article</link></item></channel></rss>`))
	if err != nil {
		t.Fatal(err)
	}
	q := newMemoryQueue()
	w := q.worker()
	claim := w.claim
	w.claim = func(ctx context.Context, e entry, pass claimPass) (string, bool, error) {
		if e.Title != cleanHeadline {
			t.Errorf("claim title=%q", e.Title)
		}
		return claim(ctx, e, pass)
	}
	w.summarize = func(_ context.Context, title, body string) (string, error) {
		if title != cleanHeadline {
			return "", fmt.Errorf("unexpected title %q", title)
		}
		return "summary", nil
	}
	sent := 0
	w.send = func(_ context.Context, e entry, summary string) error {
		sent++
		if e.Title != cleanHeadline {
			t.Errorf("send title=%q", e.Title)
		}
		return nil
	}
	if err := processArticles(context.Background(), f.Entries, w); err != nil {
		t.Fatal(err)
	}
	if sent != 1 || q.states[key(f.Entries[0])].Status != "published" {
		t.Fatal("normalized article was not published exactly once")
	}
}

func TestTestRunNormalizesHeadline(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", fakeTelegramToken)
	t.Setenv("TELEGRAM_CHANNEL", "@test")
	t.Setenv("GEMINI_API_KEY", "fake-gemini-key")
	const articleURL = "https://example.com/article?a=1&b=2"
	const summary = "A synthetic summary with <literal> text & enough detail for the publication test."
	geminiCalls, telegramCalls := 0, 0
	stubHTTP(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.String() == feedURL:
			return httpResponse(200, `<rss><channel><item><guid>test</guid><title><![CDATA[`+reportedHeadline+`]]></title><link>https://example.com/article?a=1&amp;b=2</link></item></channel></rss>`), nil
		case req.URL.String() == articleURL:
			return httpResponse(200, `<div class="bodytext"><p>`+strings.Repeat("This article contains enough text to extract and summarize. ", 8)+`</p></div>`), nil
		case req.URL.Host == "generativelanguage.googleapis.com":
			geminiCalls++
			var payload struct {
				Contents []struct {
					Parts []struct{ Text string }
				}
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Contents) != 1 || len(payload.Contents[0].Parts) != 1 {
				t.Fatal("unexpected Gemini payload")
			}
			if prompt := payload.Contents[0].Parts[0].Text; !strings.Contains(prompt, "TITLE: "+cleanHeadline+"\n\nARTICLE:") {
				t.Errorf("Gemini prompt contains an unnormalized headline: %s", prompt)
			}
			return httpResponse(200, `{"candidates":[{"content":{"parts":[{"text":"`+summary+`"}]}}]}`), nil
		case req.URL.Host == "api.telegram.org":
			telegramCalls++
			var payload struct {
				Text      string `json:"text"`
				ParseMode string `json:"parse_mode"`
				ChatID    string `json:"chat_id"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			want := "<b>Reg readers land on &#39;Binface&#39; as name for UK space cleanup mission</b>\n\n[TEST] A synthetic summary with &lt;literal&gt; text &amp; enough detail for the publication test.\n\n<a href=\"https://example.com/article?a=1&amp;b=2\">Read full article →</a>"
			if payload.Text != want || payload.ParseMode != "HTML" || payload.ChatID != "@test" {
				t.Errorf("unexpected Telegram payload: %+v", payload)
			}
			return httpResponse(200, `{"ok":true}`), nil
		default:
			t.Fatalf("unexpected request to %s", req.URL.Host)
			return nil, nil
		}
	})
	if err := testRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	if geminiCalls != 1 || telegramCalls != 1 {
		t.Fatalf("Gemini calls=%d, Telegram calls=%d; want one each", geminiCalls, telegramCalls)
	}
}
