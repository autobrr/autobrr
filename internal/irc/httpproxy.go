// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// httpProxyDialer implements proxy.ContextDialer for HTTP CONNECT proxies
type httpProxyDialer struct {
	proxyURL *url.URL
	forward  proxy.Dialer
}

// newHTTPProxyDialer creates a new HTTP CONNECT proxy dialer
func newHTTPProxyDialer(proxyURL *url.URL, forward proxy.Dialer) *httpProxyDialer {
	return &httpProxyDialer{
		proxyURL: proxyURL,
		forward:  forward,
	}
}

// DialContext implements proxy.ContextDialer
func (d *httpProxyDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Only support TCP connections
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	// Connect to the proxy server
	proxyAddr := d.proxyURL.Host
	if d.proxyURL.Port() == "" {
		// Default HTTP proxy port
		proxyAddr = net.JoinHostPort(d.proxyURL.Hostname(), "8080")
	}

	// Use a dialer with timeout from context
	dialer := &net.Dialer{}
	proxyConn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy %s: %w", proxyAddr, err)
	}

	// Handle HTTPS proxies (HTTP CONNECT over TLS)
	if d.proxyURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			ServerName:         d.proxyURL.Hostname(),
			InsecureSkipVerify: true,
		}
		proxyConn = tls.Client(proxyConn, tlsConfig)
	}

	// Send CONNECT request with additional headers for better compatibility
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\n", addr)
	connectReq += fmt.Sprintf("Host: %s\r\n", addr)
	connectReq += "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n"
	connectReq += "Proxy-Connection: Keep-Alive\r\n"
	connectReq += "Connection: Keep-Alive\r\n"

	// Add proxy authentication if provided
	if d.proxyURL.User != nil {
		password, _ := d.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
			d.proxyURL.User.Username(), password)))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}

	connectReq += "\r\n"

	// Set a deadline for the CONNECT handshake
	deadline := time.Now().Add(30 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := proxyConn.SetDeadline(deadline); err != nil {
		proxyConn.Close()
		return nil, err
	}

	// Send the CONNECT request
	if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	reader := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Only read error body on failure
		var errorBody string
		if resp.Body != nil {
			bodyBytes := make([]byte, 1024)
			n, _ := resp.Body.Read(bodyBytes)
			if n > 0 {
				errorBody = string(bodyBytes[:n])
			}
			resp.Body.Close()
		}

		proxyConn.Close()
		errMsg := fmt.Sprintf("proxy CONNECT to %s failed with status: %s", addr, resp.Status)
		if errorBody != "" {
			errMsg += fmt.Sprintf(" (body: %s)", errorBody)
		}

		switch resp.StatusCode {
		case http.StatusForbidden:
			errMsg += " - The proxy may be blocking IRC ports (6667/6697) or the destination host"
		case http.StatusProxyAuthRequired:
			errMsg += " - Proxy authentication required"
		case http.StatusUnauthorized:
			errMsg += " - Invalid proxy credentials"
		}

		return nil, errors.New(errMsg)
	}

	// Close the response body for successful responses (should be empty for CONNECT)
	if resp.Body != nil {
		resp.Body.Close()
	}

	// Reset the deadline
	if err := proxyConn.SetDeadline(time.Time{}); err != nil {
		proxyConn.Close()
		return nil, err
	}

	return proxyConn, nil
}

// Dial implements proxy.Dialer (for compatibility)
func (d *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}
