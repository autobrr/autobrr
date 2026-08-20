package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/utils"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog"
)

// Per-field limits. Exceeding any of them makes Discord reject the whole
// message with 400 rather than truncating it.
// https://docs.discord.com/developers/resources/message#embed-object-embed-limits
const (
	EmbedTitleLimit       = 256
	EmbedDescriptionLimit = 4096
	EmbedFieldNameLimit   = 256
	EmbedFieldValueLimit  = 1024
)

type Config struct {
	WebHookURL string
	Name       string
}

type Message struct {
	Content   string   `json:"content,omitempty"`
	TTS       bool     `json:"tts,omitempty"`
	Embeds    []*Embed `json:"embeds,omitempty"`
	Username  string   `json:"username,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
}

func NewMessage() *Message {
	return &Message{
		Embeds: []*Embed{},
	}
}

func (m *Message) AddEmbed(embeds ...*Embed) {
	m.Embeds = append(m.Embeds, embeds...)
}

type Author struct {
	Name    string `json:"name,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
	URL     string `json:"url,omitempty"`
}

type Footer struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type Embeds []*Embed

type Embed struct {
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Color       int         `json:"color,omitempty"`
	Fields      EmbedFields `json:"fields,omitempty"`
	Timestamp   time.Time   `json:"timestamp,omitempty"`
	Author      *Author     `json:"author,omitempty"`
	Thumbnail   *Image      `json:"thumbnail,omitempty"`
	Image       *Image      `json:"image,omitempty"`
	URL         string      `json:"url,omitempty"`
	Footer      *Footer     `json:"footer,omitempty"`
}

func NewEmbed() *Embed {
	return &Embed{
		Fields:    EmbedFields{},
		Timestamp: time.Now(),
	}
}

type EmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

func (e *Embed) SetTitle(title string) {
	if title == "" {
		return
	}
	e.Title = utils.TruncateStr(title, EmbedTitleLimit)
}
func (e *Embed) SetDescription(description string) {
	if description == "" {
		return
	}
	e.Description = utils.TruncateStr(description, EmbedDescriptionLimit)
}

func (e *Embed) SetColor(color EmbedColors) {
	if color == 0 {
		return
	}
	e.Color = int(color)
}

func (e *Embed) SetTimestamp(timestamp time.Time) {
	if timestamp.IsZero() {
		return
	}
	e.Timestamp = timestamp
}

func (e *Embed) SetURL(url string) {
	if url == "" {
		return
	}
	e.URL = url
}

func (e *Embed) SetFooter(footer *Footer) {
	if footer == nil {
		return
	}
	e.Footer = footer
}

func (e *Embed) SetThumbnail(thumbnail *Image) {
	if thumbnail == nil {
		return
	}
	e.Thumbnail = thumbnail
}

func (e *Embed) SetImage(image *Image) {
	if image == nil {
		return
	}
	e.Image = image
}

func (e *Embed) SetAuthor(name, iconURL, url string) {
	if name == "" {
		return
	}
	e.Author = &Author{
		Name:    name,
		IconURL: iconURL,
		URL:     url,
	}
}

func (e *Embed) AddAuthor(author *Author) {
	if author == nil {
		return
	}
	e.Author = author
}

func (e *Embed) AddField(field *EmbedField) {
	if field == nil {
		return
	}

	e.Fields = append(e.Fields, field)
}

func (e *Embed) AddFields(fields ...*EmbedField) {
	for _, field := range fields {
		if field == nil {
			continue
		}

		e.Fields = append(e.Fields, field)
	}
}

type EmbedFields []*EmbedField

func (e EmbedFields) Add(fields ...*EmbedField) EmbedFields {
	vals := make([]*EmbedField, 0, len(e)+len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}

		vals = append(vals, field)
	}
	return vals
}

func TextField(name, value string, inline bool) *EmbedField {
	if value == "" {
		return nil
	}
	return &EmbedField{
		Name:   name,
		Value:  utils.TruncateStr(value, EmbedFieldValueLimit),
		Inline: inline,
	}
}

func SizeField(name string, value uint64, inline bool) *EmbedField {
	if value == 0 {
		return nil
	}
	return &EmbedField{
		Name:   name,
		Value:  humanize.IBytes(value),
		Inline: inline,
	}
}

func CodeField(name, value string, inline bool) *EmbedField {
	if value == "" {
		return nil
	}
	return &EmbedField{
		Name:   name,
		Value:  utils.TruncateStr("```\n"+value+"\n```", EmbedFieldValueLimit-12),
		Inline: inline,
	}
}

type Link struct {
	Label string
	URL   string
}

func NewLink(label, url string) Link {
	return Link{
		Label: label,
		URL:   url,
	}
}

func NewLinkF(label, url string, args ...any) Link {
	return Link{
		Label: label,
		URL:   fmt.Sprintf(url, args...),
	}
}

func LinksField(name string, inline bool, links ...Link) *EmbedField {
	if len(links) == 0 {
		return nil
	}
	l := make([]string, 0, len(links))
	for _, link := range links {
		l = append(l, fmt.Sprintf("[%s](%s)", link.Label, link.URL))
	}
	return &EmbedField{
		Name:   name,
		Value:  strings.Join(l, " ∙ "),
		Inline: inline,
	}
}

type Image struct {
	URL string `json:"url,omitempty"`
}

type EmbedColors int

const (
	LIGHT_BLUE EmbedColors = 5814783  // 58b9ff
	RED        EmbedColors = 16711680 // ff0000
	ORANGE     EmbedColors = 15105570 // e67e22
	GREEN      EmbedColors = 38912    // 009800
	GRAY       EmbedColors = 10070709 // 99aab5
	YELLOW     EmbedColors = 15844367 // f1c40f
	TEAL       EmbedColors = 1752220  // 1abc9c
	PURPLE     EmbedColors = 10181046 // 9b59b6
	BLURPLE    EmbedColors = 5793266  // 5865f2
)

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func (s *Client) Name() string {
	return "discord"
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "discord").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *Client) SendMessage(ctx context.Context, message *Message) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "could not marshal message to json: %v", message)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.WebHookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request for message: %v", jsonData)
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	// TODO retryable http on status 429

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for message: %v", jsonData)
	}

	defer sharedhttp.DrainAndClose(res)

	s.log.Trace().Msgf("discord response status: %d", res.StatusCode)

	// discord responds with 204, Notifiarr with 204 so lets take all 200 as ok
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		// Limit error body reading to prevent memory issues
		limitedReader := io.LimitReader(res.Body, 4096) // 4KB limit
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return errors.Wrap(err, "could not read body for discord: %v", jsonData)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Str("message", string(jsonData)).Msg("notification successfully sent to discord")

	return nil
}
