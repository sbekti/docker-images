package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil || !slices.ContainsFunc(a.trusted, func(p netip.Prefix) bool { return p.Contains(peer.Addr().Unmap()) }) || !slices.Contains(strings.Split(a.host, ","), r.Host) {
		http.Error(w, "Access denied.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path != "/" && r.URL.Path != "/container" || r.Method == http.MethodPost && !map[string]bool{"/scan": true, "/connect": true, "/disconnect": true, "/wifi": true, "/container": true, "/ping": true, "/dns": true}[r.URL.Path] {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		if err := r.ParseForm(); err != nil || r.Header.Get("Origin") != "http://"+r.Host ||
			r.Header.Get("Sec-Fetch-Site") == "cross-site" || subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(a.token)) != 1 {
			http.Error(w, "Reload the page and try again.", http.StatusForbidden)
			return
		}
		if !a.mu.TryLock() {
			a.respond(w, r, http.StatusConflict, view{Token: a.token, Error: "busy"})
			return
		}
		defer a.mu.Unlock()
	}
	if r.URL.Path == "/ping" || r.URL.Path == "/dns" {
		a.diagnose(w, r)
		return
	}
	// Finish a requested band handover even if the phone briefly reconnects.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 40*time.Second)
	defer cancel()
	run, closeConn, err := a.dial(ctx)
	if err != nil {
		a.respond(w, r, http.StatusServiceUnavailable, view{Token: a.token, Error: "router"})
		return
	}
	defer closeConn()
	if r.URL.Path == "/container" {
		a.container(w, r, run)
		return
	}
	m := readMaintenance(run, a.config)
	s, err := readStatus(run, a.config)
	if err != nil {
		a.respond(w, r, http.StatusServiceUnavailable, view{Token: a.token, status: s, Maintenance: m, Error: "wifi"})
		return
	}
	v := view{status: s, Token: a.token, SelectedSSID: s.SSID, SelectedBand: s.Band, Maintenance: m}
	if r.Method == http.MethodGet {
		v.Error = r.URL.Query().Get("error")
		switch r.URL.Query().Get("result") {
		case "start":
			v.Message = "Start requested."
		case "restart":
			v.Message = "Restart requested."
		}
		if band := r.URL.Query().Get("band"); band == "2g" || band == "5g" {
			v.SelectedBand = band
		}
		if r.URL.Query().Has("scan") {
			if cached := a.scan.Load(); cached != nil {
				v.Networks = cached.Networks
				v.SelectedBand = cached.Band
				if len(v.Networks) == 0 {
					v.Message = "No networks found."
				}
			}
		}
		if r.URL.Query().Has("ssid") {
			v.SelectedSSID = r.URL.Query().Get("ssid")
		}
	} else {
		if !s.WiFiReady {
			v.Error = "wifi"
			a.respond(w, r, http.StatusOK, v)
			return
		}
		switch r.URL.Path {
		case "/wifi":
			selected := map[string]bool{}
			for _, ssid := range r.PostForm["wifi"] {
				if !slices.ContainsFunc(s.WiFi, func(ap accessPoint) bool { return ap.SSID == ssid }) {
					v.Error = "selection"
					break
				}
				selected[ssid] = true
			}
			if v.Error == "" {
				err = applyAPs(run, a.config, s, selected)
			}
		case "/scan", "/connect":
			band := r.PostForm.Get("band")
			if band == "" {
				band = "2g"
			}
			if band != "2g" && band != "5g" {
				v.Error = "band"
				break
			}
			v.SelectedBand = band
			ssid, password := r.PostForm.Get("ssid"), r.PostForm.Get("password")
			v.SelectedSSID = ssid
			if r.URL.Path == "/connect" && (!utf8.ValidString(ssid) || len(ssid) == 0 || len(ssid) > 32 || strings.ContainsAny(ssid, "\x00\r\n") ||
				password != "" && (len(password) < 8 || len(password) > 63)) {
				v.Error = "network"
				break
			}
			radio := s.bandRadio(a.config, band)
			if radio.ID == "" {
				v.Error = "band"
				break
			}
			if r.URL.Path == "/scan" {
				// Scanning must not hand over the connected radio or change DHCP.
				var rows []record
				rows, err = run("/interface/wifi/scan", "=number="+radio.Name, "=duration=8s")
				v.Networks = networks(rows)
				if err != nil {
					v.Error = "scan"
					var transport net.Error
					if errors.As(err, &transport) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, context.Canceled) {
						v.Error = "scan-connection"
					}
					// Record the class only: API error text may contain credentials.
					log.Printf("scan failed (%T)", err)
				}
			} else {
				err = selectBand(run, a.config, s, band)
				if err == nil {
					err = connect(run, a.config, radio, ssid, password)
				}
			}
		case "/disconnect":
			_, err = run("/interface/wifi/set", "=.id="+s.Radio.ID, "=disabled=yes")
			if err == nil && s.DHCP.ID != "" {
				_, err = run("/ip/dhcp-client/set", "=.id="+s.DHCP.ID, "=disabled=yes")
			}
		}
		if err != nil && v.Error == "" {
			// API traps may contain user-supplied credentials. Never render or log them.
			v.Error = "apply"
		}
	}
	a.respond(w, r, http.StatusOK, v)
}

// Redirect form results to GET so reload never repeats a scan or settings change.
func (a *app) respond(w http.ResponseWriter, r *http.Request, code int, v view) {
	if r.Method == http.MethodGet {
		if r.URL.Query().Has("diagnostic") {
			v.Diagnostic = a.diagnostic.Load()
			v.DiagnosticsOpen = true
		}
		a.render(w, code, v)
		return
	}
	query := url.Values{}
	if v.Error != "" {
		query.Set("error", v.Error)
		query.Set("ssid", v.SelectedSSID)
	} else if r.URL.Path == "/scan" {
		a.scan.Store(&scanResult{Band: v.SelectedBand, Networks: v.Networks})
		query.Set("scan", "1")
	}
	if v.SelectedBand == "2g" || v.SelectedBand == "5g" {
		query.Set("band", v.SelectedBand)
	}
	target := "/"
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (a *app) render(w http.ResponseWriter, code int, v view) {
	v.Error = formErrors[v.Error]
	tmpl := page
	if path := os.Getenv("PAGE_FILE"); path != "" {
		var err error
		tmpl, err = template.New("page.html").Funcs(templateFuncs).ParseFiles(path)
		if err != nil {
			log.Print("Cannot load page template")
			http.Error(w, "Cannot load page.", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_ = tmpl.Execute(w, v)
}
