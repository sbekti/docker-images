package main

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
)

func TestPowerConfirmationAndReplay(t *testing.T) {
	for _, action := range []string{"restart", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture()
			a := testApp(f)
			f.failOn = "/interface/wifi"
			post := func(form url.Values) *httptest.ResponseRecorder {
				r := httptest.NewRequest("POST", "http://"+a.host+"/power", strings.NewReader(form.Encode()))
				r.RemoteAddr = "192.0.2.1:1000"
				r.Header.Set("Origin", "http://"+a.host)
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				a.ServeHTTP(w, r)
				return w
			}
			if w := post(nil); w.Code != 403 || len(f.calls) != 0 {
				t.Fatal("CSRF guard", w.Code, f.calls)
			}
			if w := request(a, "GET", "/power?action=invalid", nil); w.Code != 400 || len(f.calls) != 0 {
				t.Fatal("invalid action accepted")
			}
			w := request(a, "GET", "/power?action="+action, nil)
			if w.Code != 200 || !strings.Contains(w.Body.String(), "router?") || !strings.Contains(w.Body.String(), "router.example") {
				t.Fatal(w.Body.String())
			}
			if len(f.calls) != 1 || f.calls[0][0] != "/system/identity/print" {
				t.Fatal("confirmation executed action", f.calls)
			}
			confirm := a.powerConfirmation.Load()
			form := url.Values{"csrf": {a.token}, "action": {action}, "confirmation": {confirm.Nonce}}
			if w := post(url.Values{"csrf": {a.token}, "action": {action}, "confirmation": {"bad"}}); w.Code != 403 {
				t.Fatal(w.Code)
			}
			f.calls = nil
			w = post(form)
			command := "/system/reboot"
			if action == "shutdown" {
				command = "/system/shutdown"
			}
			if w.Code != 303 || len(f.calls) != 1 || f.calls[0][0] != command {
				t.Fatal(w.Code, f.calls)
			}
			if w := post(form); w.Code != 403 || len(f.calls) != 1 {
				t.Fatal("replay accepted", w.Code, f.calls)
			}
			f.calls = nil
			w = request(a, "GET", "/power?result=1", nil)
			if w.Code != 200 || len(f.calls) != 0 || !strings.Contains(w.Body.String(), "Return to router") {
				t.Fatal("result retried action")
			}
			request(a, "GET", "/power?action="+action, nil)
			if a.powerConfirmation.Load().Nonce == confirm.Nonce {
				t.Fatal("nonce reused")
			}
		})
	}
}

func TestPowerRejectionAndLostConnection(t *testing.T) {
	for _, err := range []error{&routeros.DeviceError{Sentence: &proto.Sentence{Map: map[string]string{"message": "private-api-secret"}}}, io.EOF, errors.New("private-api-secret")} {
		f := newFixture()
		a := testApp(f)
		original := a.dial
		a.dial = func(ctx context.Context) (runAPI, func(), error) {
			run, close, errDial := original(ctx)
			return func(words ...string) ([]record, error) {
				if words[0] == "/system/reboot" {
					return nil, err
				}
				return run(words...)
			}, close, errDial
		}
		request(a, "GET", "/power?action=restart", nil)
		confirm := a.powerConfirmation.Load()
		w := request(a, "POST", "/power", url.Values{"csrf": {a.token}, "action": {"restart"}, "confirmation": {confirm.Nonce}})
		if strings.Contains(w.Body.String(), "private-api-secret") || !strings.Contains(w.Body.String(), "Return to router") || a.powerConfirmation.Load() != nil {
			t.Fatal(w.Body.String())
		}
	}
}
