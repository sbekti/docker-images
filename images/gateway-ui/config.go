package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type config struct {
	StationProfile string        `json:"station_profile"`
	WANList        string        `json:"wan_list"`
	Radios         []radioConfig `json:"radios"`
}
type radioConfig struct {
	Interface string      `json:"interface"`
	Uplink    bool        `json:"uplink"`
	APs       []apBinding `json:"aps"`
}
type apBinding struct {
	Profile          string `json:"profile"`
	VirtualInterface string `json:"virtual_interface,omitempty"`
}

func loadConfig() (config, error) {
	var c config
	for _, name := range []string{"CONFIG_FILE", "ROUTER_ADDRESS", "ROUTER_SERVER_NAME", "ROUTER_USER", "UI_HOST", "TRUSTED_PREFIX"} {
		if os.Getenv(name) == "" {
			return c, fmt.Errorf("%s is required", name)
		}
	}
	f, err := os.Open(os.Getenv("CONFIG_FILE"))
	if err != nil {
		return c, errors.New("cannot open configuration")
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return c, errors.New("invalid configuration")
	}
	if d.Decode(new(any)) != io.EOF {
		return c, errors.New("configuration must contain one object")
	}
	return c, c.validate()
}
func (c config) validate() error {
	if c.StationProfile == "" || c.WANList == "" || len(c.Radios) == 0 {
		return errors.New("incomplete configuration")
	}
	names := map[string]bool{}
	uplinks := 0
	for _, r := range c.Radios {
		if r.Interface == "" || names[r.Interface] || len(r.APs) == 0 {
			return errors.New("invalid radio binding")
		}
		names[r.Interface] = true
		if r.Uplink {
			uplinks++
		}
		profiles := map[string]bool{}
		for i, p := range r.APs {
			if p.Profile == "" || p.Profile == c.StationProfile || profiles[p.Profile] || i > 0 && p.VirtualInterface == "" {
				return errors.New("invalid AP binding")
			}
			profiles[p.Profile] = true
			if p.VirtualInterface != "" {
				if names[p.VirtualInterface] {
					return errors.New("duplicate interface")
				}
				names[p.VirtualInterface] = true
			}
		}
	}
	if uplinks == 0 {
		return errors.New("no eligible uplink radio")
	}
	return nil
}
