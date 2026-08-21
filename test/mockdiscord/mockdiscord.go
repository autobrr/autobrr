// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package mockdiscord implements a minimal Discord webhook endpoint for
// testing notifications: it validates payloads against the documented Discord
// embed limits, records everything it accepts, and can simulate rate limiting.
package mockdiscord

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Documented Discord limits, see
// https://discord.com/developers/docs/resources/message#create-message and
// https://discord.com/developers/docs/resources/message#embed-object-embed-limits
const (
	LimitContent     = 2000
	LimitEmbeds      = 10
	LimitTitle       = 256
	LimitDescription = 4096
	LimitFields      = 25
	LimitFieldName   = 256
	LimitFieldValue  = 1024
	LimitFooterText  = 2048
	LimitAuthorName  = 256
	LimitEmbedTotal  = 6000
)

type Message struct {
	Content *string `json:"content"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
}

type EmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	Text string `json:"text,omitempty"`
}

type EmbedAuthor struct {
	Name string `json:"name,omitempty"`
}

// Received is one accepted webhook execution.
type Received struct {
	WebhookID    string    `json:"webhook_id"`
	WebhookToken string    `json:"webhook_token"`
	Message      Message   `json:"message"`
	ReceivedAt   time.Time `json:"received_at"`
	Raw          string    `json:"raw"`
}

// Server records webhook executions and serves the mock endpoints. The zero
// value is usable; RateLimitEvery > 0 makes every Nth request return 429.
type Server struct {
	RateLimitEvery int
	OnMessage      func(Received)

	mu       sync.Mutex
	messages []Received
	requests int
}

// Messages returns a copy of all accepted webhook executions.
func (s *Server) Messages() []Received {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Received, len(s.messages))
	copy(out, s.messages)
	return out
}

// Reset drops all recorded messages and the rate-limit counter.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = nil
	s.requests = 0
}

// Handler serves POST /api/webhooks/{id}/{token} like Discord does, plus
// GET /messages to fetch everything recorded so far as JSON.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/webhooks/", s.executeWebhook)
	mux.HandleFunc("/messages", s.listMessages)
	return mux
}

func (s *Server) executeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		discordError(w, http.StatusMethodNotAllowed, 0, "405: Method Not Allowed")
		return
	}

	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/webhooks/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		discordError(w, http.StatusNotFound, 10015, "Unknown Webhook")
		return
	}

	s.mu.Lock()
	s.requests++
	limited := s.RateLimitEvery > 0 && s.requests%s.RateLimitEvery == 0
	s.mu.Unlock()

	if limited {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"message":     "You are being rate limited.",
			"retry_after": 1.0,
			"global":      false,
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		discordError(w, http.StatusBadRequest, 50109, "The request body contains invalid JSON.")
		return
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		discordError(w, http.StatusBadRequest, 50109, "The request body contains invalid JSON.")
		return
	}

	if err := Validate(msg); err != nil {
		discordError(w, http.StatusBadRequest, 50035, err.Error())
		return
	}

	received := Received{
		WebhookID:    parts[0],
		WebhookToken: parts[1],
		Message:      msg,
		ReceivedAt:   time.Now(),
		Raw:          string(body),
	}

	s.mu.Lock()
	s.messages = append(s.messages, received)
	s.mu.Unlock()

	if s.OnMessage != nil {
		s.OnMessage(received)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMessages(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Messages())
}

// Validate checks a webhook message against the documented Discord limits.
// Discord counts characters, not bytes, so all lengths are in runes.
func Validate(msg Message) error {
	hasContent := msg.Content != nil && strings.TrimSpace(*msg.Content) != ""
	if !hasContent && len(msg.Embeds) == 0 {
		return fmt.Errorf("one of content or embeds is required")
	}

	if hasContent && runeLen(*msg.Content) > LimitContent {
		return fmt.Errorf("content: must be %d or fewer in length", LimitContent)
	}

	if len(msg.Embeds) > LimitEmbeds {
		return fmt.Errorf("embeds: must be %d or fewer in length", LimitEmbeds)
	}

	total := 0
	for i, embed := range msg.Embeds {
		if runeLen(embed.Title) > LimitTitle {
			return fmt.Errorf("embeds.%d.title: must be %d or fewer in length", i, LimitTitle)
		}
		if runeLen(embed.Description) > LimitDescription {
			return fmt.Errorf("embeds.%d.description: must be %d or fewer in length", i, LimitDescription)
		}
		if len(embed.Fields) > LimitFields {
			return fmt.Errorf("embeds.%d.fields: must be %d or fewer in length", i, LimitFields)
		}

		total += runeLen(embed.Title) + runeLen(embed.Description)

		for j, field := range embed.Fields {
			// Discord trims required strings, so whitespace-only is rejected too
			if strings.TrimSpace(field.Name) == "" {
				return fmt.Errorf("embeds.%d.fields.%d.name: this field is required", i, j)
			}
			if strings.TrimSpace(field.Value) == "" {
				return fmt.Errorf("embeds.%d.fields.%d.value: this field is required", i, j)
			}
			if runeLen(field.Name) > LimitFieldName {
				return fmt.Errorf("embeds.%d.fields.%d.name: must be %d or fewer in length", i, j, LimitFieldName)
			}
			if runeLen(field.Value) > LimitFieldValue {
				return fmt.Errorf("embeds.%d.fields.%d.value: must be %d or fewer in length", i, j, LimitFieldValue)
			}

			total += runeLen(field.Name) + runeLen(field.Value)
		}

		if embed.Footer != nil {
			if runeLen(embed.Footer.Text) > LimitFooterText {
				return fmt.Errorf("embeds.%d.footer.text: must be %d or fewer in length", i, LimitFooterText)
			}
			total += runeLen(embed.Footer.Text)
		}

		if embed.Author != nil {
			if runeLen(embed.Author.Name) > LimitAuthorName {
				return fmt.Errorf("embeds.%d.author.name: must be %d or fewer in length", i, LimitAuthorName)
			}
			total += runeLen(embed.Author.Name)
		}
	}

	if total > LimitEmbedTotal {
		return fmt.Errorf("embeds: combined length must be %d or fewer, got %d", LimitEmbedTotal, total)
	}

	return nil
}

func runeLen(s string) int {
	return len([]rune(s))
}

func discordError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": message,
	})
}
