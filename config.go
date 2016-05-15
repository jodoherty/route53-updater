package main

import (
	"encoding/json"
)

type Config interface {
	IsValidToken(string, string) bool
	Region(string) string
	TrustsProxy(string) bool
	ZoneId(string) string
}

type HostnameConfig struct {
	Region string
	Token  string
	ZoneId string
}

type JsonConfig struct {
	Hostnames      map[string]HostnameConfig
	TrustedProxies []string
}

func LoadJsonConfig(jsonInput []byte) (Config, error) {
	var config JsonConfig
	err := json.Unmarshal(jsonInput, &config)
	return &config, err
}

func (cfg *JsonConfig) IsValidToken(hostname string, token string) bool {
	return token != "" && cfg.Hostnames[hostname].Token == token
}

func (cfg *JsonConfig) Region(hostname string) string {
	return cfg.Hostnames[hostname].Region
}

func (cfg *JsonConfig) TrustsProxy(proxyIp string) bool {
	for _, trustedProxy := range cfg.TrustedProxies {
		if proxyIp == trustedProxy {
			return true
		}
	}
	return false
}

func (cfg *JsonConfig) ZoneId(hostname string) string {
	return cfg.Hostnames[hostname].ZoneId
}
