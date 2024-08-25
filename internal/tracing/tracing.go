// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tracing

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func New(host string) {
	go func() {
		log.Println(http.ListenAndServe(host+":6060", nil))
	}()
}
