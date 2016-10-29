package main

// Config holds global configuration, defaults are provided in main.
// GOC-Proxy config is populated from startup flag.
type Config struct {
	Environment         string
	LogLevel            string
	Port                int
	ElectionKeyPrefix   string
	HttpScheme          string
	MaxIdleConnsPerHost int
	DisableKeepAlives   bool
}
