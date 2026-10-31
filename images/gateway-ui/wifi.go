package main

import (
	"errors"
	"strconv"
	"strings"
)

type dhcpState struct{ ID, Status string }

type radioState struct {
	ID, Name, Profile, Master, Band string
	Disabled, Running               bool
}
type accessPoint struct {
	SSID    string
	Enabled bool
}

func bandOf(band string) string {
	if strings.HasPrefix(band, "2ghz") {
		return "2g"
	}
	if strings.HasPrefix(band, "5ghz") {
		return "5g"
	}
	return ""
}
func (s status) radio(name string) radioState {
	for _, r := range s.Radios {
		if r.Name == name {
			return r
		}
	}
	return radioState{}
}
func (s status) bandRadio(c config, band string) radioState {
	for _, r := range c.Radios {
		if r.Uplink {
			native := s.radio(r.Interface)
			if bandOf(native.Band) == band {
				return native
			}
		}
	}
	return radioState{}
}
func setNamed(run runAPI, path, name string, args ...string) error {
	rows, err := run(path+"/print", "?name="+name, "=.proplist=.id")
	if err != nil || len(rows) != 1 {
		return errors.New("named object unavailable")
	}
	_, err = run(append([]string{path + "/set", "=.id=" + rows[0][".id"]}, args...)...)
	return err
}
func bindRadio(run runAPI, c config, r radioConfig, profile string, disabled bool) error {
	for _, ap := range r.APs {
		if ap.VirtualInterface != "" {
			if err := setNamed(run, "/interface/wifi", ap.VirtualInterface, "=disabled=yes"); err != nil {
				return err
			}
		}
	}
	args := []string{"=configuration=" + profile, "=disabled=" + strconv.FormatBool(disabled)}
	// Clear direct overrides so the selected configuration profile is authoritative.
	for _, key := range []string{"configuration.ssid", "configuration.mode", "security", "security.authentication-types", "security.passphrase", "security.management-protection", "security.wps", "datapath", "datapath.bridge", "datapath.vlan-id", "datapath.interface-list", "aaa"} {
		args = append(args, "=!"+key+"=")
	}
	return setNamed(run, "/interface/wifi", r.Interface, args...)
}
func configureAP(run runAPI, c config, r radioConfig, s status, selected map[string]bool) error {
	primary := -1
	for i, ap := range r.APs {
		if selected[s.ProfileSSIDs[ap.Profile]] {
			primary = i
			break
		}
	}
	profile := r.APs[0].Profile
	if primary >= 0 {
		profile = r.APs[primary].Profile
	}
	if err := bindRadio(run, c, r, profile, primary < 0); err != nil {
		return err
	}
	for i, ap := range r.APs {
		if i != primary && selected[s.ProfileSSIDs[ap.Profile]] {
			if ap.VirtualInterface == "" {
				return errors.New("virtual interface unavailable")
			}
			if err := setNamed(run, "/interface/wifi", ap.VirtualInterface, "=disabled=no"); err != nil {
				return err
			}
		}
	}
	return nil
}
func applyAPs(run runAPI, c config, s status, selected map[string]bool) error {
	for _, r := range c.Radios {
		if s.radio(r.Interface).Profile != c.StationProfile {
			if err := configureAP(run, c, r, s, selected); err != nil {
				return err
			}
		}
	}
	return nil
}
func selectBand(run runAPI, c config, s status, band string) error {
	target := s.bandRadio(c, band)
	if target.ID == "" {
		return errors.New("band unavailable")
	}
	if target.Name == s.Radio.Name {
		return nil
	}
	if s.DHCP.ID == "" {
		return errors.New("DHCP client unavailable")
	}
	if _, err := run("/ip/dhcp-client/set", "=.id="+s.DHCP.ID, "=disabled=yes"); err != nil {
		return err
	}
	rows, err := run("/interface/list/member/print", "?list="+c.WANList, "?interface="+s.Radio.Name, "=.proplist=.id")
	if err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := run("/interface/list/member/remove", "=.id="+r[".id"]); err != nil {
			return err
		}
	}
	selected := map[string]bool{}
	for _, ap := range s.WiFi {
		selected[ap.SSID] = ap.Enabled
	}
	for _, r := range c.Radios {
		if r.Interface == s.Radio.Name {
			if err := configureAP(run, c, r, s, selected); err != nil {
				return err
			}
		}
	}
	for _, r := range c.Radios {
		if r.Interface == target.Name {
			return bindRadio(run, c, r, c.StationProfile, true)
		}
	}
	return errors.New("radio unavailable")
}
func connect(run runAPI, c config, radio radioState, ssid, password string) error {
	if radio.ID == "" {
		return errors.New("radio unavailable")
	}
	auth, protection := "wpa2-psk,wpa3-psk", "allowed"
	if password == "" {
		auth, protection = "", "disabled"
	}
	if err := setNamed(run, "/interface/wifi/configuration", c.StationProfile, "=mode=station", "=ssid="+ssid, "=security.authentication-types="+auth, "=security.passphrase="+password, "=security.management-protection="+protection, "=security.wps=disable"); err != nil {
		return err
	}
	for _, r := range c.Radios {
		if r.Interface == radio.Name {
			if err := bindRadio(run, c, r, c.StationProfile, false); err != nil {
				return err
			}
		}
	}
	members, err := run("/interface/list/member/print", "?list="+c.WANList, "?interface="+radio.Name, "=.proplist=.id")
	if err != nil {
		return err
	}
	if len(members) == 0 {
		if _, err := run("/interface/list/member/add", "=list="+c.WANList, "=interface="+radio.Name); err != nil {
			return err
		}
	}
	clients, err := run("/ip/dhcp-client/print", "?interface="+radio.Name, "=.proplist=.id")
	if err != nil || len(clients) != 1 {
		return errors.New("prepared DHCP client unavailable")
	}
	if _, err := run("/ip/dhcp-client/set", "=.id="+clients[0][".id"], "=disabled=no"); err != nil {
		return err
	}
	_, err = run("/ip/dns/cache/flush")
	return err
}
