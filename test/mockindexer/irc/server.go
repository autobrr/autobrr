// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"log"
	"net"
)

type Server struct {
	listener net.Listener
	clients  []*Client
	options  *ServerOptions
}

type ServerOptions struct {
	BotName string
	Channel string
}

func NewServer(options *ServerOptions) (*Server, error) {
	listener, err := net.Listen("tcp", "localhost:6697")

	if err != nil {
		return nil, err
	}

	return &Server{
		listener: listener,
		options:  options,
	}, nil
}

func (s *Server) Run() {
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			log.Printf("Failed accept: %v", err)
			continue
		}

		s.clients = append(s.clients, NewClient(conn, s.options.BotName, s.options.Channel))
	}
}

func (s *Server) SendAll(line string) {
	for _, client := range s.clients {
		client.writer <- fmt.Sprintf(":%s PRIVMSG %s :%s", s.options.BotName, s.options.Channel, line)
	}
}
