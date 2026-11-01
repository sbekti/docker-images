package main

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestScanPreservesConnectionAndSelectedBand(t *testing.T) {
	f := newFixture()
	a := testApp(f)
	w := request(a, "POST", "/scan", url.Values{"csrf": {a.token}, "band": {"5g"}})
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Cafe &lt;Guest&gt;") {
		t.Fatal(w.Body.String())
	}
	for _, c := range f.calls {
		if c[0] == "/interface/wifi/scan" {
			if c[1] != "=number=radio-b" {
				t.Fatal(c)
			}
		} else if !strings.HasSuffix(c[0], "/print") {
			t.Fatal("Scan changed router configuration", c)
		}
	}
	state, err := readStatus(f.run, a.config)
	if err != nil || state.Radio.Name != "radio-a" || state.Radio.Disabled {
		t.Fatal(state, err)
	}
	w = request(a, "GET", "/?ssid=HotelSecure&band=5g", nil)
	if !strings.Contains(w.Body.String(), `value="5g" checked`) || !strings.Contains(w.Body.String(), `value="HotelSecure"`) {
		t.Fatal("band or SSID lost", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "2.4 GHz") {
		t.Fatal("connected band changed in display")
	}
	f.calls = nil
	request(a, "GET", "/?scan=1", nil)
	for _, c := range f.calls {
		if c[0] == "/interface/wifi/scan" {
			t.Fatal("reload repeated scan")
		}
	}
}

func TestScanErrorsAreSpecificAndSafe(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{errors.New("API trap secret-password"), "Could not scan. Try again."},
		{&net.OpError{Op: "read", Net: "tcp", Err: context.DeadlineExceeded}, "Router connection interrupted during scan."},
	} {
		f := newFixture()
		a := testApp(f)
		a.dial = func(context.Context) (runAPI, func(), error) {
			return func(words ...string) ([]record, error) {
				if words[0] == "/interface/wifi/scan" {
					return nil, tc.err
				}
				return f.run(words...)
			}, func() {}, nil
		}
		w := request(a, "POST", "/scan", url.Values{"csrf": {a.token}})
		if !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "secret-password") || strings.Contains(w.Body.String(), "Could not apply settings") {
			t.Fatal(w.Body.String())
		}
	}
}

func TestConfirmationIsGeneric(t *testing.T) {
	for _, name := range []string{"overlay-client", "web-cache"} {
		f := newFixture()
		a := testApp(f)
		a.config.RestartContainers = []string{name}
		f.rows["/container"] = []record{{".id": "*service", "name": name, "running": "true"}}
		w := request(a, "GET", "/container?container="+name+"&action=restart", nil)
		if !strings.Contains(w.Body.String(), "Restart "+name+"?") || !strings.Contains(w.Body.String(), "This briefly interrupts the service.") || strings.Contains(w.Body.String(), "VPN") {
			t.Fatal(w.Body.String())
		}
	}
}
