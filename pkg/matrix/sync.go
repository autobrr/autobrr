// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

// Event types and message types the announce listener reads.
const (
	EventTypeRoomMessage = "m.room.message"
	EventTypeRoomMember  = "m.room.member"
	MsgTypeText          = "m.text"
	MsgTypeNotice        = "m.notice"
	RelationTypeReplace  = "m.replace"
)

// SyncRequest is one /sync call.
type SyncRequest struct {
	// Since is the next_batch token of the previous response; empty for an initial sync.
	Since string
	// Timeout is how long the homeserver may hold the request open waiting for events.
	Timeout time.Duration
	// Rooms limits the response to these room ids.
	Rooms []string
	// TimelineLimit caps the events returned per room.
	TimelineLimit int
}

// SyncResponse is the subset of the /sync response for joined room timelines.
type SyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]JoinedRoom `json:"join"`
	} `json:"rooms"`
}

// JoinedRoom holds the new timeline events of a joined room.
type JoinedRoom struct {
	Timeline struct {
		Events  []Event `json:"events"`
		Limited bool    `json:"limited"`
	} `json:"timeline"`
}

// Event is a room event.
type Event struct {
	Type           string          `json:"type"`
	EventID        string          `json:"event_id"`
	Sender         string          `json:"sender"`
	OriginServerTS int64           `json:"origin_server_ts"`
	StateKey       *string         `json:"state_key,omitempty"`
	Content        json.RawMessage `json:"content"`
	Unsigned       struct {
		PrevContent json.RawMessage `json:"prev_content"`
	} `json:"unsigned"`
}

// IsJoinOf reports whether the event is userID joining the room. A profile change is
// also a join membership event, so an event whose previous membership already was join
// does not count.
func (e Event) IsJoinOf(userID string) bool {
	if e.Type != EventTypeRoomMember || e.StateKey == nil || *e.StateKey != userID {
		return false
	}

	return membership(e.Content) == "join" && membership(e.Unsigned.PrevContent) != "join"
}

func membership(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}

	var member struct {
		Membership string `json:"membership"`
	}

	if err := json.Unmarshal(content, &member); err != nil {
		return ""
	}

	return member.Membership
}

// MessageContent is the content of an m.room.message event.
type MessageContent struct {
	MsgType   string `json:"msgtype"`
	Body      string `json:"body"`
	RelatesTo *struct {
		RelType string `json:"rel_type"`
	} `json:"m.relates_to,omitempty"`
}

// IsEdit reports whether the message replaces an earlier one.
func (m MessageContent) IsEdit() bool {
	return m.RelatesTo != nil && m.RelatesTo.RelType == RelationTypeReplace
}

// Message decodes the content of an m.room.message event.
func (e Event) Message() (*MessageContent, error) {
	if e.Type != EventTypeRoomMessage {
		return nil, errors.New("event %s is not a message: %s", e.EventID, e.Type)
	}

	var content MessageContent
	if err := json.Unmarshal(e.Content, &content); err != nil {
		return nil, errors.Wrap(err, "could not decode message content")
	}

	return &content, nil
}

type syncFilter struct {
	AccountData eventFilter `json:"account_data"`
	Presence    eventFilter `json:"presence"`
	Room        roomFilter  `json:"room"`
}

type roomFilter struct {
	Rooms       []string    `json:"rooms"`
	AccountData eventFilter `json:"account_data"`
	Ephemeral   eventFilter `json:"ephemeral"`
	State       eventFilter `json:"state"`
	Timeline    eventFilter `json:"timeline"`
}

type eventFilter struct {
	Limit *int     `json:"limit,omitempty"`
	Types []string `json:"types"`
}

// Sync long-polls the homeserver for new events in the requested rooms.
func (c *Client) Sync(ctx context.Context, req SyncRequest) (*SyncResponse, error) {
	filter := syncFilter{
		AccountData: eventFilter{Types: []string{}},
		Presence:    eventFilter{Types: []string{}},
		Room: roomFilter{
			Rooms:       req.Rooms,
			AccountData: eventFilter{Types: []string{}},
			Ephemeral:   eventFilter{Types: []string{}},
			State:       eventFilter{Types: []string{}},
			// member events mark where a room joined mid-session stops being history
			Timeline: eventFilter{Limit: new(req.TimelineLimit), Types: []string{EventTypeRoomMessage, EventTypeRoomMember}},
		},
	}

	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal sync filter")
	}

	query := url.Values{
		"filter":       {string(filterJSON)},
		"timeout":      {strconv.FormatInt(req.Timeout.Milliseconds(), 10)},
		"set_presence": {"offline"},
	}

	if req.Since != "" {
		query.Set("since", req.Since)
	}

	var resp SyncResponse
	if err := c.do(ctx, http.MethodGet, "/_matrix/client/v3/sync", query, nil, &resp); err != nil {
		return nil, err
	}

	// without next_batch the stream position is lost and the next session would skip announces as backlog
	if resp.NextBatch == "" {
		return nil, errors.New("sync response has no next_batch")
	}

	return &resp, nil
}
