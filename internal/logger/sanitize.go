// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

var (
	regexReplacements = []struct {
		pattern *regexp.Regexp
		repl    string
	}{
		{
			pattern: regexp.MustCompile(`("apikey\\":\s?\\"|"host\\":\s?\\"|"password\\":\s?\\"|"user\\":\s?\\"|ExternalWebhookHost:)(\S+)(\\"|\sExternalWebhookData:)`),
			repl:    "${1}REDACTED${3}",
		},
		{
			pattern: regexp.MustCompile(`(torrent_pass|passkey|authkey|auth|secret_key|api|apikey)=([a-zA-Z0-9]+)`),
			repl:    "${1}=REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(https?://[^\s]+/((rss/download/[a-zA-Z0-9]+/)|torrent/download/((auto\.[a-zA-Z0-9]+\.|[a-zA-Z0-9]+\.))))([a-zA-Z0-9]+)`),
			repl:    "${1}REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(https?://)(.*?):(.*?)@`),
			repl:    "${1}REDACTED_USER:REDACTED_PW@",
		},
		{
			pattern: regexp.MustCompile(`(NickServ IDENTIFY )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`),
			repl:    "${1}REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(AUTHENTICATE )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`),
			repl:    "${1}REDACTED",
		},
		{
			pattern: regexp.MustCompile(
				`(?m)(` +
					`(?:Voyager autobot\s+\w+|Satsuki enter #announce\s+\w+|Sauron bot #ant-announce\s+\w+|Millie announce|DBBot announce|PT-BOT invite|midgards announce|HeBoT !invite|NBOT !invite|PS-Info pass|Synd1c4t3 invite|UHDBot invite|ENDOR !invite(\s+)\w+|immortal invite(\s+)\w+|Muffit bot #nbl-announce\s+\w+|hermes enter #announce\s+\w+|Drone enter #red-announce\s+\w+|RevoTT !invite\s+\w+|erica letmeinannounce\s+\w+|Cerberus identify\s+\w+)` +
					`)(?:\s+[a-zA-Z0-9]+)`),
			repl: "$1 REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(LiMEY_ !invite\s+)([a-zA-Z0-9]+)(\s+\w+)`),
			repl:    "${1}REDACTED${3}",
		},
		{
			pattern: regexp.MustCompile(`(Vertigo ENTER #GGn-Announce\s+)(\w+).([a-zA-Z0-9]+)`),
			repl:    "$1$2 REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(Hummingbird ENTER\s+\w+).([a-zA-Z0-9]+)(\s+#ptp-announce-dev)`),
			repl:    "$1 REDACTED$3",
		},
		{
			pattern: regexp.MustCompile(`(SceneHD..invite).([a-zA-Z0-9]+)(\s+#announce)`),
			repl:    "$1 REDACTED$3",
		},
	}
)

func SanitizeLogFile(filePath string, output io.Writer) error {
	inFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	reader := bufio.NewReader(inFile)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	for {
		// Read the next line from the file
		line, err := reader.ReadString('\n')

		if err != nil {
			if err != io.EOF {
				log.Error().Msgf("Error reading line from input file: %v", err)
			}
			break
		}

		// Sanitize the line using regexReplacements array
		bIRC := strings.Contains(line, `"module":"irc"`)
		bFilter := (strings.Contains(line, `"module":"feed"`) ||
			strings.Contains(line, `"module":"filter"`)) ||
			strings.Contains(line, `"repo":"release"`) ||
			strings.Contains(line, `"module":"action"`)

		for i := 0; i < len(regexReplacements); i++ {
			// Apply the first three patterns only if the line contains "module":"feed",
			// "module":"filter", "repo":"release", or "module":"action"
			if i < 4 {
				if bFilter {
					line = regexReplacements[i].pattern.ReplaceAllString(line, regexReplacements[i].repl)
				}
			} else if bIRC {
				// Check for "module":"irc" before applying other patterns
				line = regexReplacements[i].pattern.ReplaceAllString(line, regexReplacements[i].repl)
			}
		}

		// Write the sanitized line to the writer
		if _, err = writer.WriteString(line); err != nil {
			log.Error().Msgf("Error writing line to output: %v", err)
			return err
		}
	}

	return nil
}
