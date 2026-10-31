package main

import (
	"errors"
	"net/netip"
	"sort"
	"strings"
)

func readStatus(run runAPI, c config) (status, error) {
	s := status{Link: "Disconnected", Uplink: "None", WiFiReady: true}
	ids, err := run("/system/identity/print", "=.proplist=name")
	if err != nil || len(ids) != 1 {
		return s, errors.New("identity unavailable")
	}
	s.Hostname = ids[0]["name"]
	rows, err := run("/interface/wifi/print", "=.proplist=.id,name,configuration,master-interface,channel.band,disabled,running")
	if err != nil {
		return s, err
	}
	for _, r := range rows {
		s.Radios = append(s.Radios, radioState{ID: r[".id"], Name: r["name"], Profile: r["configuration"], Master: r["master-interface"], Band: r["channel.band"], Disabled: r["disabled"] == "true", Running: r["running"] == "true"})
	}
	profiles, err := run("/interface/wifi/configuration/print", "=.proplist=name,ssid")
	if err != nil {
		return s, err
	}
	s.ProfileSSIDs = map[string]string{}
	for _, p := range profiles {
		s.ProfileSSIDs[p["name"]] = p["ssid"]
	}
	for _, r := range s.Radios {
		if r.Profile == c.StationProfile && r.Master == "" {
			s.Radio = r
			s.Band = bandOf(r.Band)
		}
	}
	if s.Radio.ID == "" {
		return s, errors.New("station not provisioned")
	}
	s.SSID = s.ProfileSSIDs[c.StationProfile]
	for _, r := range c.Radios {
		if s.radio(r.Interface).ID == "" {
			s.WiFiReady = false
		}
		for _, ap := range r.APs {
			ssid := s.ProfileSSIDs[ap.Profile]
			if ssid == "" {
				s.WiFiReady = false
				continue
			}
			if ap.VirtualInterface != "" && s.radio(ap.VirtualInterface).ID == "" {
				s.WiFiReady = false
			}
			index := -1
			for i, n := range s.WiFi {
				if n.SSID == ssid {
					index = i
				}
			}
			if index < 0 {
				s.WiFi = append(s.WiFi, accessPoint{SSID: ssid})
				index = len(s.WiFi) - 1
			}
			master := s.radio(r.Interface)
			child := s.radio(ap.VirtualInterface)
			if master.Profile != c.StationProfile && !master.Disabled && (master.Profile == ap.Profile || child.ID != "" && !child.Disabled && child.Profile == ap.Profile) {
				s.WiFi[index].Enabled = true
			}
		}
	}
	clients, err := run("/ip/dhcp-client/print", "=.proplist=.id,interface,status,address")
	if err != nil {
		return s, err
	}
	for _, client := range clients {
		if client["interface"] == s.Radio.Name {
			s.DHCP = dhcpState{ID: client[".id"], Status: client["status"]}
		}
	}
	for _, r := range c.Radios {
		if r.Uplink {
			found := false
			for _, client := range clients {
				if client["interface"] == r.Interface {
					found = true
				}
			}
			if !found {
				s.WiFiReady = false
			}
		}
	}
	if !s.Radio.Disabled {
		s.Link = "Connecting"
		if s.Radio.Running {
			s.Link = "Connected"
			if s.DHCP.Status != "bound" {
				s.Link = "Getting an IP address"
			}
		}
	}
	if !s.Radio.Disabled && s.Radio.Running {
		peers, e := run("/interface/wifi/registration-table/print", "?interface="+s.Radio.Name, "=.proplist=signal")
		if e != nil {
			return s, e
		}
		if len(peers) > 0 {
			s.Signal = peers[0]["signal"]
		}
	}
	routes, err := run("/ip/route/print", "?dst-address=0.0.0.0/0", "?routing-table=main", "=.proplist=active,immediate-gw,gateway")
	if err != nil {
		return s, err
	}
	addresses, err := run("/ip/address/print", "?disabled=false", "=.proplist=interface,address")
	if err != nil {
		return s, err
	}
	dns, err := run("/ip/dns/print", "=.proplist=servers,dynamic-servers")
	if err != nil {
		return s, err
	}
	if len(dns) > 0 {
		s.DNS = strings.Join(unique(dns[0]["servers"], dns[0]["dynamic-servers"]), ", ")
	}
	paths := map[string][]string{}
	for _, route := range routes {
		if route["active"] != "true" {
			continue
		}
		hops := route["immediate-gw"]
		if hops == "" {
			hops = route["gateway"]
		}
		for _, hop := range unique(hops) {
			gateway, iface, qualified := strings.Cut(hop, "%")
			if !qualified {
				if _, err := netip.ParseAddr(hop); err == nil {
					iface = "Unknown"
				} else {
					iface, gateway = hop, ""
				}
			}
			paths[iface] = append(paths[iface], gateway)
		}
	}
	ifaces := make([]string, 0, len(paths))
	for iface := range paths {
		ifaces = append(ifaces, iface)
	}
	sort.Strings(ifaces)
	var names []string
	for _, iface := range ifaces {
		name := iface
		var ips []string
		for _, address := range addresses {
			if prefix, err := netip.ParsePrefix(address["address"]); err == nil && prefix.Addr().Is4() && address["interface"] == iface {
				ips = append(ips, address["address"])
			}
		}
		s.Connections = append(s.Connections, connection{name, strings.Join(unique(ips...), ", "), strings.Join(unique(paths[iface]...), ", ")})
		names = append(names, name)
	}
	if len(names) > 0 {
		s.Uplink = strings.Join(names, " and ")
	}
	return s, nil
}
