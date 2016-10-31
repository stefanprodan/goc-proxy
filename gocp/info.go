package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var (
	Version     = "0.0.1-alpha.0"
	BuildDate   = "No date provided."
	Revision    = "No revision provided."
	Branch      = "master"
	GoVersion   = runtime.Version()
	Uptime      = time.Now().UTC()
	Hostname, _ = os.Hostname()
	WorkDir, _  = os.Getwd()
)

// Info runtime and build information
func Info() map[string]string {
	return map[string]string{
		"Uptime":    fmt.Sprintf("%v", Uptime),
		"Version":   Version,
		"BuildDate": BuildDate,
		"Revision":  Revision,
		"Branch":    Branch,
		"GoVersion": GoVersion,
		"Hostname":  Hostname,
		"WorkDir":   WorkDir,
	}
}
