// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bufio"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectRequestsChathistoryOnlyThroughBouncer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		useBouncer bool
		want       bool
	}{
		{name: "bouncer", useBouncer: true, want: true},
		{name: "direct", useBouncer: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { ln.Close() })

			network := domain.IrcNetwork{
				ID:      1,
				Name:    "caps",
				Nick:    "bot",
				Enabled: true,
			}
			if tt.useBouncer {
				network.Server = "irc.example.test"
				network.Port = 6667
				network.UseBouncer = true
				network.BouncerAddr = ln.Addr().String()
			} else {
				host, port, err := net.SplitHostPort(ln.Addr().String())
				require.NoError(t, err)
				network.Server = host
				network.Port, err = strconv.Atoi(port)
				require.NoError(t, err)
			}

			h := NewHandler(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, network, nil, nil, nil)
			go func() { _ = h.Run() }()
			t.Cleanup(h.Stop)

			caps := readCapRequests(t, ln)
			assert.Equal(t, tt.want, caps["draft/chathistory"])
		})
	}
}

// readCapRequests accepts one client and returns the caps it requests before it sends USER.
func readCapRequests(t *testing.T, ln net.Listener) map[string]bool {
	t.Helper()

	require.NoError(t, ln.(*net.TCPListener).SetDeadline(time.Now().Add(5*time.Second)))
	conn, err := ln.Accept()
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

	caps := map[string]bool{}
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg, err := ircmsg.ParseLine(scanner.Text())
		require.NoError(t, err)

		switch msg.Command {
		case "CAP":
			if len(msg.Params) >= 2 && msg.Params[0] == "REQ" {
				caps[msg.Params[1]] = true
			}
		case "USER":
			return caps
		}
	}
	require.NoError(t, scanner.Err())
	t.Fatal("client closed the connection before USER")
	return nil
}

func TestConnectRegistersWhetherBouncerAcksOrNaksChathistory(t *testing.T) {
	t.Parallel()

	for _, reply := range []string{"ACK", "NAK"} {
		t.Run(reply, func(t *testing.T) {
			t.Parallel()

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { ln.Close() })

			network := domain.IrcNetwork{
				ID:          1,
				Name:        "caps",
				Server:      "irc.example.test",
				Port:        6667,
				Nick:        "bot",
				Enabled:     true,
				UseBouncer:  true,
				BouncerAddr: ln.Addr().String(),
			}

			h := NewHandler(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, network, nil, nil, nil)
			go func() { _ = h.Run() }()
			t.Cleanup(h.Stop)

			go serveBouncerRegistration(ln, reply)

			live := func() bool {
				h.m.RLock()
				defer h.m.RUnlock()
				return h.clientState == ircLive
			}
			assert.True(t, waitFor(live, 5*time.Second), "registration did not finish after the bouncer sent %s", reply)
		})
	}
}

// serveBouncerRegistration answers each CAP REQ with reply and ends registration after CAP END.
func serveBouncerRegistration(ln net.Listener, reply string) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg, err := ircmsg.ParseLine(scanner.Text())
		if err != nil || msg.Command != "CAP" || len(msg.Params) == 0 {
			continue
		}

		var out string
		switch msg.Params[0] {
		case "LS":
			out = ":bnc CAP * LS :draft/chathistory"
			if reply == "NAK" {
				out = ":bnc CAP * LS :"
			}
		case "REQ":
			out = ":bnc CAP * " + reply + " :" + msg.Params[1]
		case "END":
			out = ":bnc 001 bot :welcome\r\n:bnc 376 bot :End of /MOTD command"
		}
		if _, err := conn.Write([]byte(out + "\r\n")); err != nil {
			return
		}
	}
}
