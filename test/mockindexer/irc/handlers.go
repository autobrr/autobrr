// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"log"
	"strings"
	"time"
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

		c.writer <- fmt.Sprintf(":localhost 001 %s :Welcome %s", c.nick, c.nick)
		c.writer <- fmt.Sprintf(":localhost 002 %s :Your host is localhost, running mock-irc-0.0.1", c.nick)
		c.writer <- fmt.Sprintf(":localhost 003 %s :This server was created %s", c.nick, time.Now().String())
		c.writer <- fmt.Sprintf(":localhost 004 %s localhost mock-irc-0.0.1 o o o", c.nick)
		c.writer <- fmt.Sprintf(":localhost 251 %s :there are 1 users on 1 server", c.nick)
		c.writer <- fmt.Sprintf(":localhost 422 %s :MOTD File is missing", c.nick)
	}
}

func CommandHandler(c *Client, cmd []string) {
	log.Printf("cmd: %+v", cmd)
	switch cmd[0] {
	case "CAP":
		log.Printf("caps: %+v", cmd)
	case "JOIN":
		c.writer <- fmt.Sprintf(":localhost 221 %s +Zi", c.nick)
		c.writer <- fmt.Sprintf(":localhost 331 %s %s :No topic", c.nick, c.channelName)
		c.writer <- fmt.Sprintf(":localhost 353 %s = %s :%s %s", c.nick, c.channelName, c.nick, c.botName)
		c.writer <- fmt.Sprintf(":localhost 366 %s %s :End of NAMES list", c.nick, c.channelName)
	case "PING":
		c.writer <- fmt.Sprintf(":localhost PONG localhost %s", strings.Join(cmd[1:], " "))
	case "PRIVMSG":
		c.writer <- fmt.Sprintf(":localhost %s PRIVMSG %s %s", fmt.Sprintf(":%s", c.nick), cmd[1], fmt.Sprintf("%s", strings.Join(cmd[2:], " ")))
	case "QUIT":
		c.writer <- fmt.Sprintf(":%s!%s@localhost QUIT :Quit%s", c.nick, c.nick, strings.Join(cmd[1:], " "))
		//c.writer <- fmt.Sprintf(":localhost %s!%s@localhost QUIT :Quit%s", c.nick, c.nick, strings.Join(cmd[1:], " "))
		//c.writer <- fmt.Sprintf(":localhost :%s@localhost QUIT :Quit%s", c.nick, strings.Join(cmd[1:], " "))
		c.writer <- fmt.Sprintf("ERROR :Quit%s", strings.Join(cmd[1:], " "))
	case "ERROR":
		c.writer <- fmt.Sprintf("ERROR :Quit%s", strings.Join(cmd[1:], " "))
	}
}
