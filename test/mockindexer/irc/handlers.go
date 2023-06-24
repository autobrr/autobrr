// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"log"
	"strings"
)

func RegistrationHandler(c *Client, cmd []string) {
	switch cmd[0] {
	case "NICK":
		c.nick = cmd[1]
		break
	case "USER":
		c.user = cmd[1]
	}

	if c.nick != "" && c.user != "" {
		log.Printf("Logged in\n")

		c.handler = CommandHandler

		c.writer <- fmt.Sprintf(
			"001 %s :\r\n002 %s :\r\n003 %s :\r\n004 %s n n-d o o\r\n251 %s :\r\n422 %s :",
			c.nick, c.nick, c.nick, c.nick, c.nick, c.nick)
	}
}

func CommandHandler(c *Client, cmd []string) {
	switch cmd[0] {
	case "JOIN":
		c.writer <- fmt.Sprintf("331 %s %s :No topic", c.nick, c.channelName)
		c.writer <- fmt.Sprintf("353 %s = %s :%s %s", c.nick, c.channelName, c.nick, c.botName)
		c.writer <- fmt.Sprintf("366 %s %s :End", c.nick, c.channelName)
	case "PING":
		c.writer <- fmt.Sprintf("PONG n %s", strings.Join(cmd[1:], " "))
	case "PRIVMSG":
		c.writer <- fmt.Sprintf("%s PRIVMSG %s %s", fmt.Sprintf(":%s", c.nick), cmd[1], fmt.Sprintf("%s", strings.Join(cmd[2:], " ")))
	}
}
