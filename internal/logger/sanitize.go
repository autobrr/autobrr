// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

const redactedValue = "REDACTED"

// secretFields are json keys whose value is a secret wherever it appears, at
// any depth. Redacting by key is exact, unlike matching the value, so anything
// that is only ever a credential belongs here rather than in a rule.
var secretFields = map[string]struct{}{
	"api_key":           {},
	"apikey":            {},
	"access_token":      {},
	"auth_key":          {},
	"authkey":           {},
	"client_secret":     {},
	"cookie":            {},
	"invite_command":    {},
	"irc_key":           {},
	"nickserv_password": {},
	"pass":              {},
	"passkey":           {},
	"password":          {},
	"raw_cookie":        {},
	"rawcookie":         {},
	"rsskey":            {},
	"secret_key":        {},
	"token":             {},
	"torrent_pass":      {},
}

// rule redacts a secret embedded inside a string, such as a passkey in a
// download url or a credential in an irc command.
//
// scopes limits a rule to the modules whose lines can contain that data, taken
// from the module or repo field of the line. A rule with no scopes runs
// everywhere, which is the default for anything credential shaped so that a new
// module cannot silently start leaking. Only rules whose trigger is
// module-specific by construction, like the irc bot commands, are scoped.
type rule struct {
	name   string
	scopes []string
	// prefilter is a literal the pattern cannot match without. Checking it is
	// far cheaper than running the regex, and most values match no rule at all.
	prefilter string
	pattern   *regexp.Regexp
	repl      string
}

// appliesTo reports whether the rule should run. A nil scopes means the line
// could not be parsed, so the module is unknown and every rule runs rather than
// risk letting a credential through.
func (r rule) appliesTo(scopes []string) bool {
	if len(r.scopes) == 0 || scopes == nil {
		return true
	}

	for _, want := range r.scopes {
		if slices.Contains(scopes, want) {
			return true
		}
	}

	return false
}

// ircInviteCommand mirrors an invite_command default from
// internal/indexer/definitions. skip is how many tokens sit between the command
// and the credential, which is 1 for the trackers that want a username first.
type ircInviteCommand struct {
	command string
	skip    int
}

var ircInviteCommands = []ircInviteCommand{
	{"BJAnnounce invite", 0},
	{"Cerberus identify", 1},
	{"DBBot announce", 0},
	{"Drone enter #red-announce", 1},
	{"ENDOR !invite", 1},
	{"Fapster !auth", 0},
	{"HD-Bot enter #announce", 1},
	{"HeBoT !invite", 0},
	{"Hummingbird ENTER", 1},
	{"LiMEY_ !invite", 0},
	{"Millie announce", 0},
	{"Muffit bot #nbl-announce", 1},
	{"NBOT !invite", 0},
	{"OldSchool invite", 1},
	{"PS-Info pass", 0},
	{"PT-BOT invite", 0},
	{"RevoTT !invite", 1},
	{"Satsuki enter #announce", 1},
	{"Sauron bot #ant-announce", 1},
	{"SceneHD .invite", 0},
	{"Synd1c4t3 invite", 0},
	{"THR key", 0},
	{"UHDBot invite", 0},
	{"Vertigo ENTER #GGn-Announce", 1},
	{"Voyager autobot", 1},
	{"Yuki enter #sugoi-announce", 1},
	{"bithumen !invite", 0},
	{"erica letmeinannounce", 1},
	{"hermes enter #announce", 1},
	{"immortal invite", 1},
	{"midgards announce", 0},
	{"mmt !invite", 0},
	{"stonekeeper enter", 1},
	{"vulcan enter #announce", 1},
}

var rules = buildRules()

func buildRules() []rule {
	rules := []rule{
		{
			name:      "url_query_credential",
			prefilter: "=",
			pattern:   regexp.MustCompile(`\b(torrent_pass|passkey|authkey|auth_key|auth|secret_key|api_key|apikey|api|rsskey|rss_key|access_token|token)=([^\s&"'\\]+)`),
			repl:      "${1}=" + redactedValue,
		},
		{
			name:      "url_path_credential",
			prefilter: "/download/",
			pattern:   regexp.MustCompile(`(https?://[^\s]+/((rss/download/[a-zA-Z0-9]+/)|torrent/download/((auto\.[a-zA-Z0-9]+\.|[a-zA-Z0-9]+\.))))([a-zA-Z0-9]+)`),
			repl:      "${1}" + redactedValue,
		},
		{
			name:      "url_basic_auth",
			prefilter: "@",
			pattern:   regexp.MustCompile(`(https?://)([^\s:/@]+):([^\s:/@]+)@`),
			repl:      "${1}REDACTED_USER:REDACTED_PW@",
		},
		{
			// a config or payload dump embedded in a message, which reaches the
			// log as escaped json inside a string value
			name:      "embedded_json_secret",
			prefilter: "\"",
			pattern:   regexp.MustCompile(`(\\?"(?:apikey|api_key|host|user|username|password|passkey|authkey|token|secret)\\?"\s*:\s*\\?")([^"\\]*)`),
			repl:      "${1}" + redactedValue,
		},
		{
			name:      "embedded_torrent_data",
			prefilter: "torrentData",
			pattern:   regexp.MustCompile(`(\\?"torrentData\\?"\s*:\s*\\?")([^"\\]*)`),
			repl:      "${1}" + redactedValue,
		},
		{
			name:      "external_webhook_host",
			prefilter: "ExternalWebhookHost:",
			pattern:   regexp.MustCompile(`(ExternalWebhookHost:)(\S+)`),
			repl:      "${1}" + redactedValue,
		},
		{
			name:      "irc_nickserv_identify",
			scopes:    []string{"irc"},
			prefilter: "NickServ",
			pattern:   regexp.MustCompile(`(NickServ :?IDENTIFY )\S+(?:\s+\S+)?`),
			repl:      "${1}" + redactedValue,
		},
		{
			name:      "irc_sasl_authenticate",
			scopes:    []string{"irc"},
			prefilter: "AUTHENTICATE",
			pattern:   regexp.MustCompile(`(AUTHENTICATE )\S+`),
			repl:      "${1}" + redactedValue,
		},
	}

	for _, cmd := range ircInviteCommands {
		skip := ""
		if cmd.skip > 0 {
			skip = `(?:\s+\S+){` + strconv.Itoa(cmd.skip) + `}`
		}

		rules = append(rules, rule{
			name:      "irc_invite_" + cmd.command,
			scopes:    []string{"irc"},
			prefilter: cmd.command,
			pattern:   regexp.MustCompile(`(` + regexp.QuoteMeta(cmd.command) + skip + `\s+)\S+`),
			repl:      "${1}" + redactedValue,
		})
	}

	return rules
}

// SanitizeLogFile strips credentials from a log file and writes the result to
// output. Lines are handled as the json zerolog writes; anything that does not
// parse falls back to matching the raw text.
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
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, wErr := writer.WriteString(sanitizeLine(line)); wErr != nil {
				log.Error().Err(wErr).Msg("error writing line to output")
				return wErr
			}
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Error().Err(err).Msg("error reading line from input file")
			}
			break
		}
	}

	return nil
}

func sanitizeLine(line string) string {
	trimmed := strings.TrimRight(line, "\r\n")
	suffix := line[len(trimmed):]

	if sanitized, ok := sanitizeJSONLine(trimmed); ok {
		return sanitized + suffix
	}

	// not json, so the module is unknown and every rule has to be considered
	return applyRules(trimmed, nil) + suffix
}

func applyRules(s string, scopes []string) string {
	for _, r := range rules {
		if r.prefilter != "" && !strings.Contains(s, r.prefilter) {
			continue
		}

		if !r.appliesTo(scopes) {
			continue
		}

		s = r.pattern.ReplaceAllString(s, r.repl)
	}

	return s
}

// sanitizeJSONLine rewrites a zerolog line, redacting secret fields by name and
// running the rules over the remaining string values. Key order is preserved,
// which decoding into a map would not do. It reports false for anything it will
// not rewrite faithfully, leaving the caller to fall back to the text rules.
func sanitizeJSONLine(line string) (string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(line), "{") {
		return "", false
	}

	// Unmarshal reads the module and repo, but it is also the guard: it rejects
	// the whole line unless it is exactly one well formed object, so trailing
	// junk, a second object, or nesting deeper than encoding/json allows never
	// reaches the rewriter below. Do not swap this for a cheaper scan of the
	// module field, or those lines get rewritten to whatever parsed and the
	// remainder is dropped.
	var meta struct {
		Module string `json:"module"`
		Repo   string `json:"repo"`
	}
	if err := json.Unmarshal([]byte(line), &meta); err != nil {
		return "", false
	}

	scopes := make([]string, 0, 2)
	if meta.Module != "" {
		scopes = append(scopes, meta.Module)
	}
	if meta.Repo != "" {
		scopes = append(scopes, meta.Repo)
	}

	dec := json.NewDecoder(strings.NewReader(line))
	dec.UseNumber()

	var buf bytes.Buffer

	// one encoder per line rather than per string, it only exists to escape
	// exactly the way zerolog does
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	w := &jsonWriter{buf: &buf, enc: enc}
	if err := rewriteValue(dec, w, scopes); err != nil {
		return "", false
	}

	return buf.String(), true
}

func rewriteValue(dec *json.Decoder, w *jsonWriter, scopes []string) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			w.buf.WriteByte('{')

			first := true
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return err
				}

				key, ok := keyTok.(string)
				if !ok {
					return errUnexpectedToken
				}

				if !first {
					w.buf.WriteByte(',')
				}
				first = false

				w.writeString(key)
				w.buf.WriteByte(':')

				if isSecretField(key) {
					// consume the value whatever its shape, so an object or an
					// array under a secret key is dropped whole
					var discard json.RawMessage
					if err := dec.Decode(&discard); err != nil {
						return err
					}

					w.writeString(redactedValue)

					continue
				}

				if err := rewriteValue(dec, w, scopes); err != nil {
					return err
				}
			}

			if _, err := dec.Token(); err != nil {
				return err
			}

			w.buf.WriteByte('}')

		case '[':
			w.buf.WriteByte('[')

			first := true
			for dec.More() {
				if !first {
					w.buf.WriteByte(',')
				}
				first = false

				if err := rewriteValue(dec, w, scopes); err != nil {
					return err
				}
			}

			if _, err := dec.Token(); err != nil {
				return err
			}

			w.buf.WriteByte(']')

		default:
			return errUnexpectedToken
		}

	case string:
		w.writeString(applyRules(t, scopes))

	case json.Number:
		w.buf.WriteString(t.String())

	case bool:
		w.buf.WriteString(strconv.FormatBool(t))

	case nil:
		w.buf.WriteString("null")

	default:
		return errUnexpectedToken
	}

	return nil
}

func isSecretField(key string) bool {
	_, ok := secretFields[strings.ToLower(key)]
	return ok
}

// jsonWriter emits sanitized json, escaping strings the way zerolog does
// rather than the way json.Marshal would, so urls keep their ampersands.
type jsonWriter struct {
	buf *bytes.Buffer
	enc *json.Encoder
}

func (w *jsonWriter) writeString(s string) {
	start := w.buf.Len()

	if err := w.enc.Encode(s); err != nil {
		w.buf.Truncate(start)
		w.buf.WriteString(`""`)

		return
	}

	// Encode appends a newline that we do not want here
	w.buf.Truncate(w.buf.Len() - 1)
}

var errUnexpectedToken = errUnexpected("unexpected json token")

type errUnexpected string

func (e errUnexpected) Error() string { return string(e) }
