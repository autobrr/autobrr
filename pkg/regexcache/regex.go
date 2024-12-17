// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package regexcache

import (
	"regexp"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

var cache = ttlcache.New[string, *regexp.Regexp](
	ttlcache.WithTTL[string, *regexp.Regexp](5 * time.Minute),
)

func init() {
	go cache.Start()
}

func MustCompilePOSIX(pattern string) *regexp.Regexp {
	item := cache.Get(pattern)
	if item != nil {
		return item.Value()
	}

	reg := regexp.MustCompilePOSIX(pattern)
	cache.Set(pattern, reg, ttlcache.NoTTL)
	return reg
}

func MustCompile(pattern string) *regexp.Regexp {
	item := cache.Get(pattern)
	if item != nil {
		return item.Value()
	}

	reg := regexp.MustCompile(pattern)
	cache.Set(pattern, reg, ttlcache.NoTTL)
	return reg
}

func CompilePOSIX(pattern string) (*regexp.Regexp, error) {
	item := cache.Get(pattern)
	if item != nil {
		return item.Value(), nil
	}

	reg, err := regexp.CompilePOSIX(pattern)
	if err != nil {
		return nil, err
	}

	cache.Set(pattern, reg, ttlcache.DefaultTTL)
	return reg, nil
}

func Compile(pattern string) (*regexp.Regexp, error) {
	item := cache.Get(pattern)
	if item != nil {
		return item.Value(), nil
	}

	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	cache.Set(pattern, reg, ttlcache.DefaultTTL)
	return reg, nil
}

func SubmitOriginal(plain string, reg *regexp.Regexp) {
	cache.Set(plain, reg, ttlcache.DefaultTTL)
}

func FindOriginal(plain string) (*regexp.Regexp, bool) {
	item := cache.Get(plain)
	if item != nil {
		return item.Value(), true
	}

	return nil, false
}
