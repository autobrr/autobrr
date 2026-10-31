// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/autobrr/go-cache/ttlcache"
	"golang.org/x/time/rate"
)

// clientRateLimiter keeps a token bucket per client IP, so one noisy client
// runs out of attempts without locking out anyone else.
type clientRateLimiter struct {
	limit    rate.Limit
	burst    int
	limiters *ttlcache.Cache[string, *rate.Limiter]
}

func newClientRateLimiter(interval time.Duration, burst int) *clientRateLimiter {
	return &clientRateLimiter{
		limit:    rate.Every(interval),
		burst:    burst,
		limiters: ttlcache.New[string, *rate.Limiter](ttlcache.SetDefaultTTL(time.Hour)),
	}
}

func (l *clientRateLimiter) limiter(ip string) *rate.Limiter {
	limiter, _ := l.limiters.GetOrSet(ip, rate.NewLimiter(l.limit, l.burst), ttlcache.DefaultTTL)

	return limiter
}

// retryAfter returns how long ip has to wait for a token without consuming
// one, zero if it may proceed.
func (l *clientRateLimiter) retryAfter(ip string, now time.Time) time.Duration {
	limiter, ok := l.limiters.Get(ip)
	if !ok {
		return 0
	}

	tokens := limiter.TokensAt(now)
	if tokens >= 1 {
		return 0
	}

	return time.Duration((1 - tokens) / float64(l.limit) * float64(time.Second))
}

// take consumes a token for ip, or returns how long to wait if none is left.
func (l *clientRateLimiter) take(ip string, now time.Time) time.Duration {
	reservation := l.limiter(ip).ReserveN(now, 1)

	if delay := reservation.DelayFrom(now); delay > 0 {
		reservation.CancelAt(now)
		return delay
	}

	return 0
}

// record consumes a token for ip regardless of how many are left.
func (l *clientRateLimiter) record(ip string, now time.Time) {
	l.limiter(ip).AllowN(now, 1)
}

func (l *clientRateLimiter) reset(ip string) {
	l.limiters.Delete(ip)
}

// Middleware rejects requests with a 429 once the client IP is out of tokens.
func (l *clientRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wait := l.take(clientIP(r.RemoteAddr), time.Now()); wait > 0 {
			setRetryAfter(w, wait)
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP keys limiters on the address without its port, since every new
// connection from the same client gets a fresh source port.
func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return host
}

func setRetryAfter(w http.ResponseWriter, wait time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
}
