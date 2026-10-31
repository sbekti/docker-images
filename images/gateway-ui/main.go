// gateway-ui is a small, server-rendered Gateway panel for RouterOS.
package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-routeros/routeros/v3"
)

//go:embed page.html
var pageHTML string

var page = template.Must(template.New("page").Parse(pageHTML))

// Only these fixed messages can be displayed through a redirect URL.
var formErrors = map[string]string{
	"router":    "Cannot reach the router. Refresh to try again.",
	"wifi":      "Router Wi-Fi settings are unavailable. Refresh to try again.",
	"selection": "Check the Wi-Fi selection.",
	"band":      "Check the Wi-Fi band.",
	"network":   "Check the network name and password.",
	"apply":     "Could not apply settings. Refresh and retry.",
	"busy":      "Busy. Retry shortly.",
}

type record = map[string]string
type runAPI func(...string) ([]record, error)
type dialAPI func(context.Context) (runAPI, func(), error)

type app struct {
	host, token string
	trusted     []netip.Prefix
	config      config
	dial        dialAPI
	mu          sync.Mutex                // One radio change or scan at a time, without a job queue.
	scan        atomic.Pointer[[]network] // Latest scan only; no credentials or saved networks.
}

type network struct {
	SSID, Security string
	Signal         int
}

type connection struct {
	Name, Address, Gateway string
}

type status struct {
	Hostname, Band, SSID, Link, Uplink, Signal, DNS string
	Connections                                     []connection
	Radio                                           radioState
	DHCP                                            dhcpState
	Radios                                          []radioState
	WiFi                                            []accessPoint
	WiFiReady                                       bool
	ProfileSSIDs                                    map[string]string
}

type view struct {
	status
	Token, Error, Message, SelectedSSID string
	Networks                            []network
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	ca, err := os.ReadFile(env("ROUTER_CA_FILE", "/run/secrets/router-ca.pem"))
	if err != nil {
		log.Fatal("Cannot read router CA")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		log.Fatal("Invalid router CA")
	}
	password, err := os.ReadFile(env("ROUTER_PASSWORD_FILE", "/run/secrets/router-password"))
	if err != nil || len(password) == 0 {
		log.Fatal("Cannot read router password")
	}
	var trusted []netip.Prefix
	for _, value := range strings.Split(os.Getenv("TRUSTED_PREFIX"), ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			log.Fatal("Invalid Trusted prefix")
		}
		trusted = append(trusted, prefix)
	}
	address := os.Getenv("ROUTER_ADDRESS")
	tlsConfig := &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12,
		ServerName: os.Getenv("ROUTER_SERVER_NAME")}
	dial := func(ctx context.Context) (runAPI, func(), error) {
		conn, err := (&tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: tlsConfig}).DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, nil, err
		}
		_ = conn.SetDeadline(time.Now().Add(40 * time.Second))
		stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
		closeConn := func() { stop(); _ = conn.Close() }
		client, err := routeros.NewClient(conn)
		if err == nil {
			err = client.Login(os.Getenv("ROUTER_USER"), strings.TrimSpace(string(password)))
		}
		if err != nil {
			closeConn()
			return nil, nil, err
		}
		return func(words ...string) ([]record, error) {
			reply, err := client.Run(words...)
			if err != nil {
				return nil, err
			}
			rows := make([]record, 0, len(reply.Re))
			for _, row := range reply.Re {
				rows = append(rows, row.Map)
			}
			return rows, nil
		}, closeConn, nil
	}
	a := &app{host: os.Getenv("UI_HOST"), token: rand.Text(), trusted: trusted, config: cfg, dial: dial}
	server := &http.Server{Addr: env("LISTEN_ADDRESS", ":8080"), Handler: a,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 50 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	log.Print("gateway-ui listening")
	log.Fatal(server.ListenAndServe())
}

func unique(values ...string) []string {
	var result []string
	seen := map[string]bool{}
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" && !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		}
	}
	return result
}

func networks(rows []record) []network {
	best := map[string]network{}
	for _, row := range rows {
		ssid := row["ssid"]
		if ssid == "" {
			continue
		}
		security := strings.ToLower(row["security"])
		method := "unsupported"
		if security == "" || security == "open" {
			method = "open"
		} else if strings.Contains(security, "psk") && !strings.Contains(security, "eap") {
			method = "personal"
		}
		signal, _ := strconv.Atoi(row["signal"])
		n := network{ssid, method, signal}
		key := ssid + "\x00" + method
		if old, exists := best[key]; !exists || signal > old.Signal {
			best[key] = n
		}
	}
	result := make([]network, 0, len(best))
	for _, n := range best {
		result = append(result, n)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Signal == result[j].Signal {
			return result[i].SSID < result[j].SSID
		}
		return result[i].Signal > result[j].Signal
	})
	return result
}
