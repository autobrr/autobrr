// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package diagnostics

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/rs/zerolog/log"
)

// SetupProfiling pprof profiling
func SetupProfiling(enabled bool, host string, port int) {
	if enabled {
		go func() {
			// pprof has an init method which adds the following endpoints
			// http.HandleFunc(prefix+"/debug/pprof/", Index)
			// http.HandleFunc(prefix+"/debug/pprof/cmdline", Cmdline)
			// http.HandleFunc(prefix+"/debug/pprof/profile", Profile)
			// http.HandleFunc(prefix+"/debug/pprof/symbol", Symbol)
			// http.HandleFunc(prefix+"/debug/pprof/trace", Trace)
			err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
			if err != nil {
				log.Printf("Error starting profiling server: %v", err)
			}
		}()
	}
}
