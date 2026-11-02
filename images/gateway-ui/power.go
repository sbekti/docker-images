package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/go-routeros/routeros/v3"
)

type powerResult struct{ Hostname, Title, Message string }

func (a *app) power(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Query().Has("result") {
		result := a.powerResult.Load()
		if result == nil {
			http.Redirect(w, r, "/#maintenance", http.StatusSeeOther)
			return
		}
		a.render(w, http.StatusOK, view{status: status{Hostname: result.Hostname}, PowerResult: result})
		return
	}
	action := r.URL.Query().Get("action")
	if r.Method == http.MethodPost {
		action = r.PostForm.Get("action")
	}
	command, label, message := "/system/reboot", "Restart", "Connections will drop while the router restarts. Reopen this page after reconnecting."
	if action == "shutdown" {
		command, label, message = "/system/shutdown", "Shut down", "Connections will stop. Cycle power to start the router again. Wait 30 seconds before removing power."
	} else if action != "restart" {
		http.Error(w, "Invalid power action.", http.StatusBadRequest)
		return
	}
	var confirm *confirmation
	if r.Method == http.MethodPost {
		confirm = a.powerConfirmation.Load()
		if confirm == nil || confirm.Action != action || subtle.ConstantTimeCompare([]byte(confirm.Nonce), []byte(r.PostForm.Get("confirmation"))) != 1 || !a.powerConfirmation.CompareAndSwap(confirm, nil) {
			http.Error(w, "Confirmation expired. Open Maintenance and confirm again.", http.StatusForbidden)
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 15*time.Second)
	defer cancel()
	run, closeConn, err := a.dial(ctx)
	if err != nil {
		http.Error(w, "Cannot reach the router. Return to Maintenance.", http.StatusServiceUnavailable)
		return
	}
	defer closeConn()
	if r.Method == http.MethodGet {
		rows, err := run("/system/identity/print", "=.proplist=name")
		if err != nil || len(rows) != 1 {
			http.Error(w, "Router identity unavailable.", http.StatusServiceUnavailable)
			return
		}
		confirm = &confirmation{Name: rows[0]["name"], Action: action, Path: "/power", Title: label + " router?", Message: message, Label: label, Nonce: rand.Text(), Danger: action == "shutdown"}
		a.powerConfirmation.Store(confirm)
		a.render(w, http.StatusOK, view{status: status{Hostname: confirm.Name}, Token: a.token, Confirmation: confirm})
		return
	}
	result := &powerResult{Hostname: confirm.Name, Title: label + " requested", Message: message}
	// Consume confirmation before sending exactly one native command. Never retry.
	if _, err := run(command); err != nil {
		var rejected *routeros.DeviceError
		if errors.As(err, &rejected) && rejected.Sentence.Word == "!trap" {
			result.Title = "Request rejected"
			result.Message = "The router rejected the request. Return to Maintenance before trying again."
		} else {
			result.Title = "Result unconfirmed"
			result.Message = "The connection closed before confirmation. Reconnect and check the router before trying again."
		}
	}
	a.powerResult.Store(result)
	http.Redirect(w, r, "/power?result=1", http.StatusSeeOther)
}
