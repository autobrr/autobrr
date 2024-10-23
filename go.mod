module github.com/autobrr/autobrr

go 1.23.2

replace github.com/r3labs/sse/v2 => github.com/autobrr/sse/v2 v2.0.0-20230520125637-530e06346d7d

require (
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/Masterminds/squirrel v1.5.4
	github.com/anacrolix/torrent v1.57.1
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/autobrr/go-deluge v1.2.0
	github.com/autobrr/go-qbittorrent v1.10.0
	github.com/autobrr/go-rtorrent v1.11.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/avast/retry-go/v4 v4.6.0
	github.com/containrrr/shoutrrr v0.8.0
	github.com/dcarbone/zadapters/zstdlog v1.0.0
	github.com/dustin/go-humanize v1.0.1
	github.com/ergochat/irc-go v0.4.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-andiamo/splitter v1.2.5
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/render v1.0.3
	github.com/gorilla/sessions v1.2.2
	github.com/gosimple/slug v1.14.0
	github.com/hashicorp/go-version v1.7.0
	github.com/hekmon/transmissionrpc/v3 v3.0.0
	github.com/icholy/digest v0.1.23
	github.com/jellydator/ttlcache/v3 v3.3.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-shellwords v1.0.12
	github.com/mmcdole/gofeed v1.3.0
	github.com/moistari/rls v0.5.12
	github.com/pkg/errors v0.9.1
	github.com/r3labs/sse/v2 v2.10.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/cors v1.11.1
	github.com/rs/zerolog v1.33.0
	github.com/sasha-s/go-deadlock v0.3.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	go.uber.org/automaxprocs v1.6.0
	golang.org/x/crypto v0.28.0
	golang.org/x/net v0.30.0
	golang.org/x/sync v0.8.0
	golang.org/x/term v0.25.0
	golang.org/x/time v0.6.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.33.1
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/anacrolix/dht/v2 v2.21.1 // indirect
	github.com/anacrolix/generics v0.0.3-0.20240902042256-7fb2702ef0ca // indirect
	github.com/anacrolix/missinggo v1.3.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.7.4 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/bradfitz/iter v0.0.0-20191230175014-e8f45d346db8 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/gdm85/go-rencode v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hekmon/cunits/v2 v2.1.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mmcdole/goxpp v1.1.1-0.20240225020742-a0c311522b23 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/petermattis/goid v0.0.0-20240813172612-4fcff4a6cae7 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	lukechampine.com/blake3 v1.1.6 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)
