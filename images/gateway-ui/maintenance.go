package main

import (
	"net/http"
	"net/url"
	"slices"
)

type maintenance struct {
	Version, Uptime string
	Containers      []containerState
}
type containerState struct{ Name, State, Action string }
type confirmation struct {
	Name, Action, Path, Title, Message, Label, Nonce string
	Danger                                           bool
}

func containerStatus(row record) containerState {
	c := containerState{Name: row["name"], State: "Unavailable"}
	for _, state := range []string{"stopping", "starting", "starting-with-healthcheck", "downloading", "extracting", "unhealthy", "stopped", "running"} {
		if row[state] == "true" {
			c.State = state
			break
		}
	}
	if c.State == "Unavailable" && row["healthy"] == "true" {
		c.State = "running"
	}
	if c.State == "running" {
		c.Action = "restart"
	}
	if c.State == "stopped" {
		c.Action = "start"
	}
	if c.State == "starting-with-healthcheck" {
		c.State = "starting"
	}
	if c.State == "downloading" || c.State == "extracting" {
		c.State = "updating"
	}
	return c
}

func readMaintenance(run runAPI, cfg config) maintenance {
	m := maintenance{}
	if rows, err := run("/system/resource/print", "=.proplist=version,uptime"); err == nil && len(rows) == 1 {
		m.Version, m.Uptime = rows[0]["version"], rows[0]["uptime"]
	}
	if len(cfg.RestartContainers) == 0 {
		return m
	}
	rows, err := run("/container/print", "=.proplist=.id,name,running,healthy,stopped,starting,stopping,starting-with-healthcheck,downloading,extracting,unhealthy")
	for _, name := range cfg.RestartContainers {
		c := containerState{Name: name, State: "Unavailable"}
		if err == nil {
			for _, row := range rows {
				if row["name"] == name {
					c = containerStatus(row)
					break
				}
			}
		}
		m.Containers = append(m.Containers, c)
	}
	return m
}

func (a *app) container(w http.ResponseWriter, r *http.Request, run runAPI) {
	values := r.URL.Query()
	if r.Method == http.MethodPost {
		values = r.PostForm
	}
	name, action := values.Get("container"), values.Get("action")
	if !slices.Contains(a.config.RestartContainers, name) || (action != "start" && action != "restart") {
		http.Error(w, "Invalid container action.", http.StatusBadRequest)
		return
	}
	rows, err := run("/container/print", "?name="+name, "=.proplist=.id,name,running,healthy,stopped")
	if err != nil || len(rows) != 1 || rows[0][".id"] == "" || containerStatus(rows[0]).Action != action {
		a.respond(w, r, http.StatusConflict, view{Token: a.token, Error: "container"})
		return
	}
	if r.Method == http.MethodGet {
		ids, _ := run("/system/identity/print", "=.proplist=name")
		label, message := "Start", ""
		if action == "restart" {
			label, message = "Restart", "This briefly interrupts the service."
		}
		v := view{Token: a.token, Confirmation: &confirmation{Name: name, Action: action, Path: "/container", Title: label + " " + name + "?", Message: message, Label: label}}
		if len(ids) == 1 {
			v.Hostname = ids[0]["name"]
		}
		a.render(w, http.StatusOK, v)
		return
	}
	// RouterOS owns the restart lifecycle; never use a script or a shell command.
	arg := "=.id=" + rows[0][".id"]
	if action == "restart" {
		arg = "=number=" + rows[0][".id"]
	}
	if _, err := run("/container/"+action, arg); err != nil {
		a.respond(w, r, http.StatusBadGateway, view{Token: a.token, Error: "container"})
		return
	}
	http.Redirect(w, r, "/?"+url.Values{"result": {action}}.Encode()+"#maintenance", http.StatusSeeOther)
}
