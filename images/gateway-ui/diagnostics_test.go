package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
)

func TestDiagnosticRepliesAndErrors(t *testing.T) {
	got := replyRecords(&routeros.Reply{Done: &proto.Sentence{Map: map[string]string{"ret": "192.0.2.1"}}})
	if len(got) != 1 || got[0]["ret"] != "192.0.2.1" {
		t.Fatal(got)
	}
	for _, received := range []string{"5", "3", "0"} {
		result := runDiagnostic(func(words ...string) ([]record, error) {
			if strings.Join(words, " ") != "/tool/ping =address=192.0.2.1 =count=5 =interval=1s" {
				t.Fatal(words)
			}
			return []record{{"host": "192.0.2.1", "sent": "5", "received": received, "avg-rtt": "28ms728us"}}, nil
		}, diagnosticResult{Tool: "Ping", Target: "192.0.2.1"})
		if result.Replies != received || received == "0" && result.Error != "No replies received." || received == "3" && result.Loss != "40" {
			t.Fatal(result)
		}
	}
	result := runDiagnostic(func(words ...string) ([]record, error) {
		if strings.Join(words, " ") != "/resolve =domain-name=example.com =type=ipv4 =server=192.0.2.53" {
			t.Fatal(words)
		}
		return []record{{"ret": "192.0.2.1"}}, nil
	}, diagnosticResult{Tool: "DNS lookup", Target: "example.com", Server: "192.0.2.53"})
	if result.Address != "192.0.2.1" || result.Error != "" {
		t.Fatal(result)
	}
	for _, err := range []error{context.DeadlineExceeded, errors.New("failure: dns name does not exist"), errors.New("private-api-secret")} {
		result = runDiagnostic(func(...string) ([]record, error) { return nil, err }, diagnosticResult{Tool: "DNS lookup"})
		if result.Error == "" || strings.Contains(result.Error, "private-api-secret") {
			t.Fatal(result)
		}
	}
}

func TestDiagnosticHTTP(t *testing.T) {
	f := newFixture()
	a := testApp(f)
	post := func(path string, form url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://"+a.host+path, strings.NewReader(form.Encode()))
		r.RemoteAddr = "192.0.2.10:1234"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://"+a.host)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		return w
	}
	if w := request(a, "POST", "/ping", nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	for _, target := range []string{"", "https://example.com", "2001:db8::1", "<script>", "example.com;reboot"} {
		f.calls = nil
		w := post("/ping", url.Values{"csrf": {a.token}, "target": {target}})
		if w.Code != 303 || len(f.calls) != 0 || a.diagnostic.Load().Error == "" {
			t.Fatal(target, w.Code, f.calls)
		}
	}
	f.calls = nil
	post("/dns", url.Values{"csrf": {a.token}, "target": {"example.com"}, "server": {"not-an-ip"}})
	if len(f.calls) != 0 {
		t.Fatal(f.calls)
	}
	f.failOn = "/interface/wifi"
	post("/dns", url.Values{"csrf": {a.token}, "target": {"example.com"}})
	if len(f.calls) != 1 || f.calls[0][0] != "/resolve" {
		t.Fatal(f.calls)
	}
	f.calls = nil
	w := request(a, "GET", "/?diagnostic=1", nil)
	if !strings.Contains(w.Body.String(), "DNS lookup result") || !strings.Contains(w.Body.String(), "Diagnostics</summary>") {
		t.Fatal(w.Body.String())
	}
	for _, call := range f.calls {
		if call[0] == "/resolve" || call[0] == "/tool/ping" {
			t.Fatal("reload reran test")
		}
	}
}
