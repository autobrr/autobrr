// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bufio"
	"log"
	"net"
	"strings"
)

type Client struct {
	conn   net.Conn
	writer chan string

	botName     string
	channelName string
	nick        string
	user        string

	handler func(c *Client, cmd []string)
}

type ClientHandler interface {
	Handle(c Client, cmd []string)
}

func NewClient(conn net.Conn, botName, channelName string) *Client {
	client := &Client{
		botName:     botName,
		channelName: channelName,
		conn:        conn,
		writer:      make(chan string),
	}

	client.handler = RegistrationHandler

	go client.readerLoop()
	go client.writerLoop()

	return client
}

func (c *Client) readerLoop() {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := scanner.Text()
		cmd := strings.Split(line, " ")

		log.Printf("%s", scanner.Text())

		c.handler(c, cmd)
	}
}

func (c *Client) writerLoop() {
	for cmd := range c.writer {
		c.conn.Write([]byte(cmd + "\r\n"))
	}
}
