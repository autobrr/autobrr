// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

// fakeIRCServer speaks just enough of the protocol for ircevent.Connection to
// finish registration: it answers USER with the welcome numerics and closes the
// link when the client QUITs. The full in-process ircd lives in test/irc; this
// one exists so unit tests can hold a genuine registered session without the
// integration build tag.
type fakeIRCServer struct {
	ln net.Listener

	// The knobs below are read from serve() without locking: set them before
	// the client's first connection, never mid-session.
	//
	// onRegister runs after USER arrives, before the welcome numerics are sent
	onRegister func()
	// refuseWith, when set, answers the client's first line with an ERROR and
	// closes the link, so registration never completes
	refuseWith string
	// ignoreQuit keeps the link open when the client QUITs, the way a real ircd
	// that has not processed the QUIT yet does; the QUIT is still counted
	ignoreQuit bool

	mu      sync.Mutex
	conns   []net.Conn
	accepts int
	quits   int
}

func newFakeIRCServer(t *testing.T) *fakeIRCServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	s := &fakeIRCServer{ln: ln}
	go s.acceptLoop()

	t.Cleanup(func() { ln.Close(); s.Drop() })

	return s
}

func (s *fakeIRCServer) Addr() (string, int) {
	addr := s.ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

func (s *fakeIRCServer) AddrString() string {
	return s.ln.Addr().String()
}

func (s *fakeIRCServer) Accepts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accepts
}

func (s *fakeIRCServer) Quits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quits
}

// Broadcast writes a raw line to every live connection and returns how many
// received it, so a test cannot pass vacuously by broadcasting to nobody.
func (s *fakeIRCServer) Broadcast(line string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	delivered := 0
	for _, conn := range s.conns {
		if _, err := conn.Write([]byte(line + "\r\n")); err == nil {
			delivered++
		}
	}

	return delivered
}

// Drop severs every live connection the way a netsplit does: no QUIT, no ERROR.
func (s *fakeIRCServer) Drop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.conns {
		conn.Close()
	}
	s.conns = nil
}

func (s *fakeIRCServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}

		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.accepts++
		s.mu.Unlock()

		go s.serve(conn)
	}
}

func (s *fakeIRCServer) serve(conn net.Conn) {
	defer conn.Close()

	nick := "tester"
	r := bufio.NewReader(conn)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if s.refuseWith != "" {
			conn.Write([]byte("ERROR :" + s.refuseWith + "\r\n"))
			return
		}

		switch {
		case strings.HasPrefix(line, "NICK "):
			nick = strings.TrimPrefix(line, "NICK ")

		case strings.HasPrefix(line, "USER "):
			if s.onRegister != nil {
				s.onRegister()
			}

			conn.Write([]byte(
				":fake.test 001 " + nick + " :welcome\r\n" +
					":fake.test 376 " + nick + " :end of motd\r\n"))

		case strings.HasPrefix(line, "PING"):
			conn.Write([]byte(":fake.test PONG fake.test :" + strings.TrimPrefix(line, "PING ") + "\r\n"))

		case strings.HasPrefix(line, "QUIT"):
			s.mu.Lock()
			s.quits++
			s.mu.Unlock()

			if !s.ignoreQuit {
				return
			}
		}
	}
}

// TestIrcGoLoopExitsAfterQuitFromDisconnectCallback pins the irc-go behaviour
// the whole reconnect design stands on: disconnect callbacks run before Loop
// wakes from a finished session, so a Quit() issued inside the callback
// guarantees Loop tears the socket down and exits instead of reconnecting on
// its own. The ordering is library internals, not API contract - if an upgrade
// ever changes it, the supervisor and the library would reconnect the same
// network in parallel, and nothing else in the suite would notice.
func TestIrcGoLoopExitsAfterQuitFromDisconnectCallback(t *testing.T) {
	srv := newFakeIRCServer(t)

	var disconnects atomic.Int32

	client := &ircevent.Connection{
		Server:    srv.AddrString(),
		Nick:      "pin",
		User:      "pin",
		RealName:  "pin",
		KeepAlive: time.Minute,
		Timeout:   5 * time.Second,
		// short enough that a library-driven reconnect shows up within the test
		ReconnectFreq: 50 * time.Millisecond,
		QuitMessage:   "bye",
		Log:           log.New(io.Discard, "", 0),
	}

	client.AddDisconnectCallback(func(_ ircmsg.Message) {
		disconnects.Add(1)
		client.Quit()
	})

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	loopDone := make(chan struct{})
	go func() {
		client.Loop()
		close(loopDone)
	}()

	srv.Drop()

	select {
	case <-loopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Loop did not exit after the disconnect callback quit the connection")
	}

	if got := disconnects.Load(); got != 1 {
		t.Errorf("disconnect callback ran %d times for one session, want exactly 1", got)
	}

	// a violation surfaces as a second connection within a few ReconnectFreqs
	time.Sleep(time.Second)
	if got := srv.Accepts(); got != 1 {
		t.Errorf("the library opened %d connections for one session: it reconnected on its own despite the quit", got)
	}
}

// TestIrcGoDisconnectCallbacksRequireRegistration pins the other library rule
// the breaker's accounting splits on: a connection refused before registration
// produces no disconnect callback, only its ERROR line, which is why
// handleServerError counts those and leaves registered sessions to
// onDisconnect. If a future irc-go version fires disconnect callbacks for
// unregistered connections too, every refusal counts twice and the breaker
// trips at half its threshold.
func TestIrcGoDisconnectCallbacksRequireRegistration(t *testing.T) {
	srv := newFakeIRCServer(t)
	srv.refuseWith = "Trying to reconnect too fast."

	var disconnects atomic.Int32

	client := &ircevent.Connection{
		Server:      srv.AddrString(),
		Nick:        "pin",
		User:        "pin",
		RealName:    "pin",
		KeepAlive:   time.Minute,
		Timeout:     5 * time.Second,
		QuitMessage: "bye",
		Log:         log.New(io.Discard, "", 0),
	}

	client.AddDisconnectCallback(func(_ ircmsg.Message) {
		disconnects.Add(1)
	})

	if err := client.Connect(); err == nil {
		client.Quit()
		t.Fatal("expected a refused registration to fail Connect")
	}

	// Connect's error path tears the connection down before returning, so the
	// callbacks have already run - or correctly have not
	if got := disconnects.Load(); got != 0 {
		t.Errorf("disconnect callback ran %d times for an unregistered connection, want 0", got)
	}
}
