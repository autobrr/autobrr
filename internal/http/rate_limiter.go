// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/autobrr/go-cache/ttlcache"
	"golang.org/x/time/rate"
)

// clientRateLimiter keeps a token bucket per client, so one noisy client runs
// out of attempts without locking out anyone else. Callers key it with
// Server.limiterKey: the TCP peer, or the forwarded client behind a trusted
// proxy, never an address a client could pick through forwarded headers.
type clientRateLimiter struct {
	limit    rate.Limit
	burst    int
	limiters *ttlcache.Cache[string, *rate.Limiter]
}

// newClientRateLimiter expires a bucket once it has gone untouched for as long
// as a full refill takes, when it is indistinguishable from a fresh one. That
// bounds memory by the clients seen per refill window instead of per hour.
func newClientRateLimiter(interval time.Duration, burst int) *clientRateLimiter {
	return &clientRateLimiter{
		limit:    rate.Every(interval),
		burst:    burst,
		limiters: ttlcache.New[string, *rate.Limiter](ttlcache.SetDefaultTTL(time.Duration(burst) * interval)),
	}
}

// limiter looks the bucket up with Get first because only Get pushes the
// expiration forward; a GetOrSet hit would let a drained bucket expire early.
func (l *clientRateLimiter) limiter(ip string) *rate.Limiter {
	if limiter, ok := l.limiters.Get(ip); ok {
		return limiter
	}

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

// Middleware rejects requests with a 429 once the client identified by key is
// out of tokens.
func (l *clientRateLimiter) Middleware(key func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wait := l.take(key(r), time.Now()); wait > 0 {
				setRetryAfter(w, wait)
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// limiterBucket groups IPv6 clients by /64, the smallest block an ISP assigns,
// so one client cannot mint a fresh bucket for every address in its prefix.
func limiterBucket(addr netip.Addr) string {
	if addr.Is6() {
		return netip.PrefixFrom(addr, 64).Masked().String()
	}

	return addr.String()
}

// clientIP strips the port from a remote address, since every new connection
// from the same client gets a fresh source port.
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
