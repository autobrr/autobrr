// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/autobrr/autobrr/test/mockdiscord"

	"github.com/spf13/pflag"
)

func main() {
	var (
		port           int
		rateLimitEvery int
	)

	pflag.IntVar(&port, "port", 8095, "port to listen on")
	pflag.IntVar(&rateLimitEvery, "rate-limit-every", 0, "respond 429 to every Nth request (0 disables)")
	pflag.Parse()

	server := &mockdiscord.Server{
		RateLimitEvery: rateLimitEvery,
		OnMessage: func(received mockdiscord.Received) {
			fmt.Printf("--- webhook %s at %s\n", received.WebhookID, received.ReceivedAt.Format("15:04:05"))
			if received.Message.Content != nil {
				fmt.Printf("content: %s\n", *received.Message.Content)
			}
			for _, embed := range received.Message.Embeds {
				fmt.Printf("embed: %s\n", embed.Title)
				if embed.Description != "" {
					fmt.Printf("  %s\n", embed.Description)
				}
				for _, field := range embed.Fields {
					fmt.Printf("  %s: %s\n", field.Name, field.Value)
				}
			}
		},
	}

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("mock discord listening on %s\n", addr)
	fmt.Printf("point a Discord notification at http://localhost:%d/api/webhooks/1/mock-token\n", port)
	fmt.Printf("recorded messages: http://localhost:%d/messages\n", port)

	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
