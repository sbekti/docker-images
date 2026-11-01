package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestSignalAndSecurity(t *testing.T) {
	for _, tc := range []struct {
		value any
		level int
		color string
	}{{-55, 4, "strong"}, {-56, 3, "strong"}, {-65, 3, "strong"}, {-66, 2, "fair"}, {-75, 2, "fair"}, {-76, 1, "weak"}, {"", 0, ""}, {0, 0, ""}, {"invalid", 0, ""}} {
		got := signalStrength(tc.value)
		if got.Level != tc.level || got.Color != tc.color {
			t.Fatalf("%v: %+v", tc.value, got)
		}
	}
	for raw, want := range map[string]string{"": "Open", "wpa2-psk": "WPA2", "wpa3-psk": "WPA3", "wpa2-psk,wpa3-psk": "WPA2/3", "wpa2-eap": "WPA2-EAP", "owe": "OWE", "unrecognized": "Other"} {
		if got := securityLabel(raw); got != want {
			t.Fatalf("%s: %s", raw, got)
		}
	}
}

func TestContainerConfirmationAndGuards(t *testing.T) {
	f := newFixture()
	a := testApp(f)
	a.config.RestartContainers = []string{"overlay-client"}
	f.rows["/container"] = []record{{".id": "*service", "name": "overlay-client", "running": "true"}}
	// Recovery controls remain available when Wi-Fi cannot be read.
	f.failOn = "/interface/wifi"
	w := request(a, "GET", "/container?container=overlay-client&action=restart", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Restart overlay-client?") {
		t.Fatal(w.Body.String())
	}
	for _, c := range f.calls {
		if !strings.HasSuffix(c[0], "/print") {
			t.Fatal("GET mutated state", c)
		}
	}
	f.calls = nil
	if w = request(a, "POST", "/container", nil); w.Code != 403 || len(f.calls) > 0 {
		t.Fatal("missing CSRF accepted")
	}
	for _, action := range []string{"start", "restart", "remove"} {
		w = request(a, "POST", "/container", url.Values{"csrf": {a.token}, "container": {"unlisted"}, "action": {action}})
		if w.Code != 400 {
			t.Fatal("unlisted target accepted", w.Code)
		}
	}
	f.failOn = "/container/restart"
	w = request(a, "POST", "/container", url.Values{"csrf": {a.token}, "container": {"overlay-client"}, "action": {"restart"}})
	if strings.Contains(w.Body.String(), "secret-password") {
		t.Fatal("API secret leaked")
	}
	f.failOn = ""
	f.calls = nil
	request(a, "POST", "/container", url.Values{"csrf": {a.token}, "container": {"overlay-client"}, "action": {"restart"}})
	count := 0
	for _, c := range f.calls {
		if c[0] == "/container/restart" {
			count++
			if c[1] != "=number=*service" {
				t.Fatal(c)
			}
		}
	}
	if count != 1 {
		t.Fatalf("restart count %d", count)
	}
	f.calls = nil
	request(a, "GET", "/?result=container", nil)
	for _, c := range f.calls {
		if !strings.HasSuffix(c[0], "/print") {
			t.Fatal("reload repeated action", c)
		}
	}
}

func TestContainerStatesAndStart(t *testing.T) {
	for _, state := range []string{"starting", "stopping", "starting-with-healthcheck", "unhealthy", "extracting"} {
		got := containerStatus(record{"name": "example", state: "true", "healthy": "true"})
		if got.Action != "" {
			t.Fatalf("transitional state has action: %+v", got)
		}
	}
	f := newFixture()
	a := testApp(f)
	a.config.RestartContainers = []string{"overlay-client"}
	f.rows["/container"] = []record{{".id": "*service", "name": "overlay-client", "stopped": "true"}}
	request(a, "POST", "/container", url.Values{"csrf": {a.token}, "container": {"overlay-client"}, "action": {"start"}})
	found := false
	for _, c := range f.calls {
		if c[0] == "/container/start" {
			found = true
			if c[1] != "=.id=*service" {
				t.Fatal(c)
			}
		}
	}
	if !found {
		t.Fatal("start not requested")
	}
	f.calls = nil
	request(a, "POST", "/container", url.Values{"csrf": {a.token}, "container": {"overlay-client"}, "action": {"remove"}})
	for _, c := range f.calls {
		if !strings.HasSuffix(c[0], "/print") {
			t.Fatal("invalid action executed", c)
		}
	}
}
