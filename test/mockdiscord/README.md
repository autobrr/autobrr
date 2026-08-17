# mockdiscord

A minimal live mock of the Discord webhook endpoint for testing autobrr
notifications without spamming a real channel. It validates payloads against
the documented Discord embed limits, records accepted messages, and can
simulate rate limiting.

## Run it

```sh
go run ./test/mockdiscord/cmd --port 8095
```

Then create a Discord notification in autobrr with the webhook URL:

```
http://localhost:8095/api/webhooks/1/mock-token
```

Every accepted execution is printed to stdout and kept in memory; fetch the
full log as JSON from `http://localhost:8095/messages`.

Flags:

- `--port` - listen port (default 8095)
- `--rate-limit-every N` - respond `429 Too Many Requests` to every Nth
  request, for exercising retry handling (default off)

## Use it in tests

```go
server := &mockdiscord.Server{}
ts := httptest.NewServer(server.Handler())
defer ts.Close()

sender := NewDiscordSender(log, &domain.Notification{
	Type:    domain.NotificationTypeDiscord,
	Enabled: true,
	Webhook: ts.URL + "/api/webhooks/1/mock-token",
	Events:  []string{"PUSH_APPROVED"},
})
```

Payloads that break a Discord limit (embed count, title/description/field
lengths, the 6000-character combined total) are rejected with a Discord-style
`400` error body, so a test failure mirrors what production Discord would do.
