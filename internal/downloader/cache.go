// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package downloader

import (
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

type Instance struct {
	config *domain.Downloader
	client any
}

func NewInstance(config *domain.Downloader, client any) *Instance {
	return &Instance{
		config: config,
		client: client,
	}
}

func (i *Instance) Config() *domain.Downloader {
	return i.config
}

func (i *Instance) ClientAs[T any]() (T, error) {
	var zero T

	if i == nil {
		return zero, errors.New("nil download client instance")
	}

	client, ok := i.client.(T)
	if !ok {
		return zero, errors.New(
			"download client %d (%s) has runtime type %T",
			i.config.ID,
			i.config.Type,
			i.client,
		)
	}

	return client, nil
}

type Cache struct {
	mu        sync.RWMutex
	instances map[int32]*Instance
}

func NewCache() *Cache {
	return &Cache{
		instances: make(map[int32]*Instance),
	}
}

func (c *Cache) Set(id int32, instance *Instance) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.instances[id] = instance
}

func (c *Cache) Get(id int32) (*Instance, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instance, ok := c.instances[id]
	return instance, ok
}

func (c *Cache) Delete(id int32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.instances, id)
}
