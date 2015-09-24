package main

import (
	"code.google.com/p/gcfg"
	"io/ioutil"
)

type Config struct {
	Backend map[string]*struct {
		URI     []string
		Index   string
		Workers int
		Bulk    int
		Queue   string
		Limit   int
	}
	General struct {
		Primary  string
		Backup   string
		Geoip2db string
		TailFile string
		Buffer   int
	}
}

var defaultConfig = `
[backend "es"]
index="ulog2queue-2006.01.02"
bulk=5000
workers=1

[general]
geoip2db="/usr/share/ulog2queue/GeoLite2-City.mmdb"
tailfile="/var/log/ulogd.json"
buffer=10000
`

func NewConfig(cfgfile string) *Config {
	var cfg Config
	gcfg.ReadStringInto(&cfg, defaultConfig)
	content, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		log.Fatal(err)
	}
	err = gcfg.ReadStringInto(&cfg, string(content))
	if err != nil {
		log.Fatal("Failed to parse "+cfgfile+":", err)
	}
	return &cfg
}
