// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"slices"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
)

const maxRoomMessages = 1000

// messageBuffer keeps the latest messages of a room for the UI log view.
type messageBuffer struct {
	m        sync.Mutex
	messages []domain.MatrixMessage
}

func (b *messageBuffer) add(msg domain.MatrixMessage) {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.messages) >= maxRoomMessages {
		b.messages = slices.Delete(b.messages, 0, len(b.messages)-maxRoomMessages+1)
	}

	b.messages = append(b.messages, msg)
}

func (b *messageBuffer) list() []domain.MatrixMessage {
	b.m.Lock()
	defer b.m.Unlock()

	return slices.Clone(b.messages)
}
