package server

import (
	"time"

	"github.com/bmeg/grip/util"
)

// Config describes configuration for the server.
type Config struct {
	HostName         string
	HTTPPort         string
	RPCPort          string
	WorkDir          string
	ContentDir       string
	ReadOnly         bool
	BasicAuth        []BasicCredential
	DisableHTTPCache bool
	// How often the server should rebuild the graph schemas
	SchemaRefreshInterval time.Duration
	// How many vertices/edges to inspect to infer the schema
	SchemaInspectN uint32
	// Strategy to use for selecting the vertices/edges to inspect.
	// Random if True; first N otherwise
	SchemaRandomSample bool
}

// HTTPAddress returns the HTTP address based on HostName and HTTPPort
func (c Config) HTTPAddress() string {
	http := ""
	if c.HostName != "" {
		http = "http://" + c.HostName
	}
	if c.HTTPPort != "" {
		http = http + ":" + c.HTTPPort
	}
	return http
}

// RPCAddress returns the RPC address based on HostName and RPCPort
func (c *Config) RPCAddress() string {
	rpc := c.HostName
	if c.RPCPort != "" {
		rpc = rpc + ":" + c.RPCPort
	}
	return rpc
}

// BasicCredential describes a username and password for use with Funnel's basic auth.
type BasicCredential struct {
	User     string
	Password string
}

func testConfig() Config {
	c := Config{}
	c.HostName = "localhost"
	c.HTTPPort = util.RandomPort()
	c.RPCPort = util.RandomPort()
	c.WorkDir = "grip.work." + util.RandomString(6)
	c.DisableHTTPCache = true
	c.SchemaRefreshInterval = 1 * time.Minute
	c.SchemaInspectN = 100
	return c
}
