package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
)

// A small stateful API fixture keeps HTTP behavior tests independent of hardware.
type fixture struct {
	mu     sync.Mutex
	rows   map[string][]record
	calls  [][]string
	failOn string
}

func newFixture() *fixture {
	f := &fixture{rows: map[string][]record{
		"/system/identity": {{"name": "router.example"}},
		"/interface/wifi": {
			{".id": "*2", "name": "radio-a", "configuration": stationProfile, "channel.band": "2ghz-ax", "configuration.country": "United States", "running": "true", "disabled": "false"},
			{".id": "*2e", "name": "radio-a-eap", "master-interface": "radio-a", "configuration": "enterprise", "disabled": "true"},
			{".id": "*5", "name": "radio-b", "channel.band": "5ghz-ax", "configuration": "personal", "running": "true", "disabled": "false"},
			{".id": "*5e", "name": "radio-b-eap", "master-interface": "radio-b", "configuration": "enterprise", "disabled": "false"},
			{".id": "*6", "name": "radio-c", "channel.band": "6ghz-ax", "configuration": "personal-6", "running": "true", "disabled": "false"},
			{".id": "*6e", "name": "radio-c-eap", "master-interface": "radio-c", "configuration": "enterprise-6", "disabled": "false"},
			{".id": "*2h", "name": "radio-a-private", "master-interface": "radio-a", "configuration": "private", "disabled": "true"},
			{".id": "*5h", "name": "radio-b-private", "master-interface": "radio-b", "configuration": "private", "disabled": "true"},
		},
		"/interface/wifi/configuration":      {{".id": "*p", "name": stationProfile, "ssid": "HotelGuest", "security.authentication-types": ""}},
		"/ip/dhcp-client":                    {{".id": "*d", "interface": "radio-a", "status": "bound", "address": "192.0.2.20/24"}},
		"/interface/list/member":             {{".id": "*w", "list": "wan", "interface": "radio-a"}, {".id": "*e", "list": "wan", "interface": "wired-a"}},
		"/ip/route":                          {{"dst-address": "192.0.2.0/0", "routing-table": "main", "active": "true", "immediate-gw": "192.0.2.1%radio-a"}},
		"/ip/address":                        {{"interface": "radio-a", "address": "192.0.2.20/24", "disabled": "false"}},
		"/ip/dns":                            {{"servers": "192.0.2.1", "dynamic-servers": "192.0.2.1,192.0.2.1,192.0.2.1"}},
		"/interface/wifi/registration-table": {{"interface": "radio-a", "signal": "-56"}},
		"/interface/wifi/scan":               {{"ssid": "Cafe <Guest>", "security": "", "signal": "-42"}, {"ssid": "HotelSecure", "security": "WPA2-PSK", "signal": "-55"}},
	}}
	f.rows["/interface/wifi/configuration"] = append(f.rows["/interface/wifi/configuration"], record{"name": "personal", "ssid": "Example personal"}, record{"name": "enterprise", "ssid": "Example enterprise"}, record{"name": "private", "ssid": "Example private"}, record{"name": "personal-6", "ssid": "Example personal"}, record{"name": "enterprise-6", "ssid": "Example enterprise"})
	f.rows["/ip/dhcp-client"] = append(f.rows["/ip/dhcp-client"], record{".id": "*db", "interface": "radio-b", "disabled": "true"})
	return f

}

func (f *fixture) run(words ...string) ([]record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string{}, words...))
	if f.failOn != "" && strings.Contains(strings.Join(words, " "), f.failOn) {
		return nil, errors.New("API trap containing secret-password")
	}
	i := strings.LastIndex(words[0], "/")
	path, action := words[0][:i], words[0][i+1:]
	values, filters := record{}, record{}
	for _, word := range words[1:] {
		key, value, _ := strings.Cut(word[1:], "=")
		if word[0] == '=' {
			values[key] = value
		} else if word[0] == '?' {
			filters[key] = value
		}
	}
	if action == "scan" {
		return f.rows[words[0]], nil
	}
	if action == "add" {
		values[".id"] = "*new"
		f.rows[path] = append(f.rows[path], values)
		if path == "/ip/dhcp-client" {
			f.lease(values)
		}
		return nil, nil
	}
	var result, kept []record
	for _, row := range f.rows[path] {
		match := true
		for k, v := range filters {
			match = match && row[k] == v
		}
		if id := values[".id"]; id != "" {
			match = match && row[".id"] == id
		}
		if match {
			switch action {
			case "print":
				result = append(result, row)
			case "set":
				for k, v := range values {
					if strings.HasPrefix(k, "!") {
						delete(row, k[1:])
					} else {
						if k == "disabled" {
							v = map[string]string{"yes": "true", "no": "false", "true": "true", "false": "false"}[v]
						}
						row[k] = v
					}
				}
				if path == "/ip/dhcp-client" {
					f.lease(row)
				}
			case "remove":
				continue
			}
		}
		kept = append(kept, row)
	}
	if action == "remove" {
		f.rows[path] = kept
	}
	return result, nil
}

func (f *fixture) lease(row record) {
	active := "true"
	if row["disabled"] == "true" || row["disabled"] == "yes" {
		active = "false"
	}
	row["status"], row["address"] = "bound", "192.0.2.20/24"
	f.rows["/ip/route"] = []record{{"dst-address": "192.0.2.0/0", "routing-table": "main", "active": active, "immediate-gw": "192.0.2.1%" + row["interface"]}}
}

const stationProfile = "example-station"

func exampleConfig() config {
	return config{StationProfile: stationProfile, WANList: "example-wan", Radios: []radioConfig{
		{Interface: "radio-a", Uplink: true, APs: []apBinding{{Profile: "personal"}, {Profile: "enterprise", VirtualInterface: "radio-a-eap"}, {Profile: "private", VirtualInterface: "radio-a-private"}}},
		{Interface: "radio-b", Uplink: true, APs: []apBinding{{Profile: "personal"}, {Profile: "enterprise", VirtualInterface: "radio-b-eap"}, {Profile: "private", VirtualInterface: "radio-b-private"}}},
		{Interface: "radio-c", APs: []apBinding{{Profile: "personal-6"}, {Profile: "enterprise-6", VirtualInterface: "radio-c-eap"}}},
	}}
}
func testApp(f *fixture) *app {
	return &app{host: "router.example", token: "test-token", trusted: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}, config: exampleConfig(), dial: func(context.Context) (runAPI, func(), error) { return f.run, func() {}, nil }}
}
func request(a *app, method, path string, form url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+a.host+path, strings.NewReader(form.Encode()))
	r.RemoteAddr = "192.0.2.42:1234"
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://"+a.host)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code == 303 {
		return request(a, "GET", w.Header().Get("Location"), nil)
	}
	return w
}
func TestDynamicIdentityAndSSIDSelections(t *testing.T) {
	for mask := range 8 {
		f := newFixture()
		a := testApp(f)
		form := url.Values{"csrf": {a.token}}
		for i, n := range []string{"Example personal", "Example enterprise", "Example private"} {
			if mask&(1<<i) != 0 {
				form.Add("wifi", n)
			}
		}
		w := request(a, "POST", "/wifi", form)
		if w.Code != 200 || strings.Contains(w.Body.String(), "Could not apply") {
			t.Fatal(w.Body.String())
		}
		state, e := readStatus(f.run, a.config)
		if e != nil {
			t.Fatal(e)
		}
		for _, n := range state.WiFi {
			want := slices.Contains(form["wifi"], n.SSID)
			if n.Enabled != want {
				t.Fatal(mask, n)
			}
		}
		if state.Radio.Profile != stationProfile {
			t.Fatal("reserved radio changed")
		}
		f.rows["/system/identity"][0]["name"] = "renamed.example"
		w = request(a, "GET", "/", nil)
		if !strings.Contains(w.Body.String(), "<h1>renamed.example</h1>") {
			t.Fatal(w.Body.String())
		}
	}
}
func TestHandoverAndPreparedDHCP(t *testing.T) {
	f := newFixture()
	a := testApp(f)
	w := request(a, "POST", "/connect", url.Values{"csrf": {a.token}, "band": {"5g"}, "ssid": {"Venue"}, "password": {"example-password"}})
	if strings.Contains(w.Body.String(), "Could not apply") {
		t.Fatal(w.Body.String())
	}
	s, e := readStatus(f.run, a.config)
	if e != nil || s.Radio.Name != "radio-b" {
		t.Fatal(s, e)
	}
	for _, c := range f.calls {
		if c[0] == "/ip/dhcp-client/add" || strings.Contains(strings.Join(c, " "), "default-route-distance") {
			t.Fatal("app wrote DHCP policy")
		}
	}
}
func TestRefreshAndGuards(t *testing.T) {
	f := newFixture()
	a := testApp(f)
	w := request(a, "POST", "/scan", url.Values{"csrf": {a.token}})
	if !strings.Contains(w.Body.String(), "Cafe &lt;Guest&gt;") {
		t.Fatal(w.Body.String())
	}
	f.calls = nil
	request(a, "GET", "/?scan=1", nil)
	for _, c := range f.calls {
		if !strings.HasSuffix(c[0], "/print") {
			t.Fatal(c)
		}
	}
	f.calls = nil
	w = request(a, "POST", "/wifi", nil)
	if w.Code != 403 || len(f.calls) > 0 {
		t.Fatal("CSRF guard failed")
	}
}
func TestMissingObjectsDoNotMutate(t *testing.T) {
	f := newFixture()
	f.rows["/ip/dhcp-client"] = nil
	a := testApp(f)
	w := request(a, "POST", "/wifi", url.Values{"csrf": {a.token}, "wifi": {"Example private"}})
	if !strings.Contains(w.Body.String(), "unavailable") {
		t.Fatal(w.Body.String())
	}
	for _, c := range f.calls {
		if !strings.HasSuffix(c[0], "/print") {
			t.Fatal(c)
		}
	}
}
func TestConfigValidation(t *testing.T) {
	c := exampleConfig()
	if c.validate() != nil {
		t.Fatal("valid example rejected")
	}
	c.Radios[1].Interface = c.Radios[0].Interface
	if c.validate() == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestConfiguredNetworkCounts(t *testing.T) {
	for _, count := range []int{1, 4} {
		f := newFixture()
		a := testApp(f)
		if count == 1 {
			a.config.Radios = a.config.Radios[:2]
			for i := range a.config.Radios {
				a.config.Radios[i].APs = a.config.Radios[i].APs[:1]
			}
		} else {
			f.rows["/interface/wifi/configuration"] = append(f.rows["/interface/wifi/configuration"], record{"name": "extra-profile", "ssid": "Fourth example"})
			f.rows["/interface/wifi"] = append(f.rows["/interface/wifi"], record{".id": "*extra", "name": "additional-interface", "master-interface": "radio-b", "configuration": "extra-profile", "disabled": "true"})
			a.config.Radios[1].APs = append(a.config.Radios[1].APs, apBinding{Profile: "extra-profile", VirtualInterface: "additional-interface"})
		}
		state, err := readStatus(f.run, a.config)
		if err != nil || !state.WiFiReady || len(state.WiFi) != count {
			t.Fatalf("count=%d state=%+v error=%v", count, state.WiFi, err)
		}
		selected := map[string]bool{}
		for _, n := range state.WiFi {
			selected[n.SSID] = true
		}
		if err = applyAPs(f.run, a.config, state, selected); err != nil {
			t.Fatal(err)
		}
	}
}
