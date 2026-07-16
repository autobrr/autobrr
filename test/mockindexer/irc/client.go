// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bufio"
	"log"
	"net"
	"strings"
	"time"
)

type Client struct {
	conn   net.Conn
	writer chan string

	botName     string
	channelName string
	nick        string
	user        string

	users map[string]struct{}

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
		users:       make(map[string]struct{}),
	}

	client.handler = RegistrationHandler

	go client.readerLoop()
	go client.writerLoop()
	go client.pingLoop()

	return client
}

func (c *Client) readerLoop() {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := scanner.Text()
		cmd := strings.Split(line, " ")

		log.Printf("--> %s", scanner.Text())

		c.handler(c, cmd)
	}
}

func (c *Client) writerLoop() {
	for cmd := range c.writer {
		log.Printf("<-- %s", []byte(cmd+"\r\n"))
		c.conn.Write([]byte(cmd + "\r\n"))
	}
}

func (c *Client) pingLoop() {
	for {
		for user, _ := range c.users {
			c.conn.Write([]byte("PING " + user + "\r\n"))
		}
		time.Sleep(60 * time.Second)
	}
}
