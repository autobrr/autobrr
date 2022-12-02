# autobrr SelfIndexer

This is a simple IRC announcer and torrent indexer rolled into one. It is built
as a tool for testing IRC announces and actions.

## Getting started

 * Put a torrent file in `./files`, name it `1.torrent`.
 * Run the SelfIndexer with `go run main.go`.

For autobrr, uncomment the `customDefinitions` line in `config.toml` to load the
extra indexer definitions. Then start autobrr as usual.

 * Add an instance of the SelfIndexer in autobrr UI. Pick any nickname,
   _don't set any auth_.
 * Set up an action - for example the watchdir action which will make autobrr
   actually download the announced torrent file from the SelfIndexer.

Posting announces.

 * Open `http://localhost:3999` in your browser. A simple input will allow you to
   post announces to the channel. For example, to announce the `1.torrent` file
   added to the `./files` dir, send this,

```
New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' freeleech -  http://localhost:3999/torrent/1
```

It is the `1` in the end of the announce line that should match the file name.
