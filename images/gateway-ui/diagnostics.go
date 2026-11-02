package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type diagnosticResult struct {
	Tool, Target, Server, Address, Replies, Loss, Average, Elapsed, Error string
}

func ipv4(value string) bool {
	address, err := netip.ParseAddr(value)
	return err == nil && address.Is4()
}

func hostname(value string) bool {
	if len(value) == 0 || len(value) > 253 {
		return false
	}
	if _, err := netip.ParseAddr(value); err == nil {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(value, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

func diagnosticFailure(err error, tool string) string {
	var network net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &network) && network.Timeout() {
		return tool + " timed out."
	}
	if strings.Contains(err.Error(), "dns name does not exist") {
		return "Hostname not found."
	}
	return tool + " failed."
}

func (a *app) diagnose(w http.ResponseWriter, r *http.Request) {
	result := diagnosticResult{Tool: "Ping", Target: strings.TrimSpace(r.PostForm.Get("target"))}
	if r.URL.Path == "/dns" {
		result.Tool = "DNS lookup"
		result.Server = strings.TrimSpace(r.PostForm.Get("server"))
	}
	switch {
	case !hostname(result.Target) && (result.Tool != "Ping" || !ipv4(result.Target)):
		result.Error = "Enter a hostname."
		if result.Tool == "Ping" {
			result.Error = "Enter an IPv4 address or hostname."
		}
	case result.Server != "" && !ipv4(result.Server):
		result.Error = "Enter an IPv4 DNS server address."
	default:
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		run, closeConn, err := a.dial(ctx)
		if err != nil {
			result.Error = "Cannot reach the router."
		} else {
			defer closeConn()
			result = runDiagnostic(run, result)
			if ctx.Err() != nil {
				result.Error = diagnosticFailure(ctx.Err(), result.Tool)
			}
		}
	}
	a.diagnostic.Store(&result)
	http.Redirect(w, r, "/?diagnostic=1#diagnostics", http.StatusSeeOther)
}

func runDiagnostic(run runAPI, result diagnosticResult) diagnosticResult {
	words := []string{"/tool/ping", "=address=" + result.Target, "=count=5", "=interval=1s"}
	if result.Tool == "DNS lookup" {
		words = []string{"/resolve", "=domain-name=" + result.Target, "=type=ipv4"}
		if result.Server != "" {
			words = append(words, "=server="+result.Server)
		}
	}
	start := time.Now()
	rows, err := run(words...)
	if err != nil {
		result.Error = diagnosticFailure(err, result.Tool)
		return result
	}
	if result.Tool == "DNS lookup" {
		result.Elapsed = fmt.Sprintf("%.0f ms", time.Since(start).Seconds()*1000)
		if len(rows) == 1 && ipv4(rows[0]["ret"]) {
			result.Address = rows[0]["ret"]
		} else {
			result.Error = "DNS lookup failed."
		}
		return result
	}
	if len(rows) == 0 {
		result.Error = "Ping failed."
		return result
	}
	last := rows[len(rows)-1]
	received, e1 := strconv.Atoi(last["received"])
	sent, e2 := strconv.Atoi(last["sent"])
	if e1 != nil || e2 != nil || sent != 5 || received < 0 || received > sent {
		result.Error = "Ping failed."
		return result
	}
	result.Replies = strconv.Itoa(received)
	result.Loss = strconv.Itoa((sent - received) * 100 / sent)
	for _, row := range rows {
		if ipv4(row["host"]) {
			result.Address = row["host"]
			break
		}
	}
	if received == 0 {
		result.Error = "No replies received."
	} else if average, err := time.ParseDuration(last["avg-rtt"]); err == nil {
		result.Average = fmt.Sprintf("%.1f ms", average.Seconds()*1000)
	}
	return result
}
