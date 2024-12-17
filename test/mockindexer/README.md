# autobrr MockIndexer

This is a simple IRC announcer and torrent indexer rolled into one. It is built
as a tool for testing IRC announces and actions.

## Getting started

 * Put a torrent file in `./files`, name it `1.torrent`.
 * Run the MockIndexer with `go run main.go`.

For autobrr, uncomment the `customDefinitions` line in `config.toml` to load the
extra indexer definitions. Then start autobrr as usual.

 * Add an instance of the MockIndexer in autobrr UI. Pick any nickname,
   _don't set any auth_.
 * Set up an action - for example the Watchdir action which will make autobrr
   actually download the announced torrent file from the MockIndexer.

## Post announce

 * Open `http://localhost:3999` in your browser. A simple input will allow you to
   post announces to the channel. For example, to announce the `1.torrent` file
   added to the `./files` dir, send this,

```
New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' freeleech -  http://localhost:3999/torrent/1
```

It is the `1` at the end of the announce line that should match the file name.

## RSS Feed

You can use the mockindexer as an RSS feed as well. Place a complete XML feed in `./feeds` and name it something like `mock.xml`.

In autobrr to set up the feed you use the url like `http://localhost:3999/feeds/mock` where the last part is the name of the xml file without extension.

## Webhook

The mockindexer can also be used as a simple webhook endpoint. Use it with a method `POST` to `http://localhost:3999/webhook`.

You can trigger different behavior by appending the following URL parameters.

- `timeout=2` - wait for 2 seconds to respond
- `status=500` - respond with status 500

Use it like `http://localhost:3999/webhook?timeout=2&status=500`.