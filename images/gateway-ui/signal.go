package main

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

var templateFuncs = template.FuncMap{"signal": signalStrength}

type signalView struct {
	Level      int
	Color, DBm string
}

func signalStrength(value any) signalView {
	text := fmt.Sprint(value)
	dbm, err := strconv.Atoi(text)
	if err != nil || dbm >= 0 || dbm < -127 {
		return signalView{}
	}
	s := signalView{Level: 1, Color: "weak", DBm: text}
	switch {
	case dbm >= -55:
		s.Level, s.Color = 4, "strong"
	case dbm >= -65:
		s.Level, s.Color = 3, "strong"
	case dbm >= -75:
		s.Level, s.Color = 2, "fair"
	}
	return s
}
func securityLabel(raw string) string {
	if raw == "" || raw == "open" {
		return "Open"
	}
	label := "Other"
	switch {
	case strings.Contains(raw, "wpa2") && strings.Contains(raw, "wpa3"):
		label = "WPA2/3"
	case strings.Contains(raw, "wpa3"):
		label = "WPA3"
	case strings.Contains(raw, "wpa2"):
		label = "WPA2"
	case strings.Contains(raw, "wpa-"):
		label = "WPA"
	case strings.Contains(raw, "owe"):
		label = "OWE"
	case strings.Contains(raw, "dpp"):
		label = "DPP"
	}
	if strings.Contains(raw, "eap") {
		label += "-EAP"
	}
	return label
}
