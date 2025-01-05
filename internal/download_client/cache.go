// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package download_client

import (
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
)

type ClientCacheStore interface {
	Set(id int32, client *domain.DownloadClient)
	Get(id int32) *domain.DownloadClient
	Pop(id int32)
}

type ClientCache struct {
	mu      sync.RWMutex
	clients map[int32]*domain.DownloadClient
}

func NewClientCache() *ClientCache {
	return &ClientCache{
		clients: make(map[int32]*domain.DownloadClient),
	}
}

func (c *ClientCache) Set(id int32, client *domain.DownloadClient) {
	if client != nil {
		c.mu.Lock()
		c.clients[id] = client
		c.mu.Unlock()
	}
}

func (c *ClientCache) Get(id int32) *domain.DownloadClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.clients[id]
	if ok {
		return v
	}
	return nil
}

func (c *ClientCache) Pop(id int32) {
	c.mu.Lock()
	delete(c.clients, id)
	c.mu.Unlock()
}
