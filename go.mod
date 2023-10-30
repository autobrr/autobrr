module github.com/autobrr/autobrr

go 1.20

replace github.com/r3labs/sse/v2 => github.com/autobrr/sse/v2 v2.0.0-20230520125637-530e06346d7d

require (
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/Masterminds/squirrel v1.5.4
	github.com/anacrolix/torrent v1.53.1
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/autobrr/go-deluge v1.1.0
	github.com/autobrr/go-qbittorrent v1.7.2-0.20231029234932-67580aa0e42a
	github.com/autobrr/go-rtorrent v1.10.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/avast/retry-go/v4 v4.5.0
	github.com/dcarbone/zadapters/zstdlog v1.0.0
	github.com/dustin/go-humanize v1.0.1
	github.com/ergochat/irc-go v0.4.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-chi/chi/v5 v5.0.10
	github.com/go-chi/render v1.0.3
	github.com/gorilla/sessions v1.2.1
	github.com/gosimple/slug v1.13.1
	github.com/hashicorp/go-version v1.6.0
	github.com/hekmon/transmissionrpc/v2 v2.0.1
	github.com/lib/pq v1.10.9
	github.com/mattn/go-shellwords v1.0.12
	github.com/mmcdole/gofeed v1.2.1
	github.com/moistari/rls v0.5.10
	github.com/pkg/errors v0.9.1
	github.com/r3labs/sse/v2 v2.10.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/cors v1.10.1
	github.com/rs/zerolog v1.31.0
	github.com/sasha-s/go-deadlock v0.3.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.17.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/crypto v0.14.0
	golang.org/x/net v0.17.0
	golang.org/x/sync v0.4.0
	golang.org/x/term v0.13.0
	golang.org/x/time v0.3.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.26.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/anacrolix/dht/v2 v2.20.0 // indirect
	github.com/anacrolix/missinggo v1.3.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.7.2 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/bradfitz/iter v0.0.0-20191230175014-e8f45d346db8 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gdm85/go-rencode v0.1.8 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hekmon/cunits/v2 v2.1.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mmcdole/goxpp v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/petermattis/goid v0.0.0-20230518223814-80aa455d8761 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.3.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.10.0 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.41.0 // indirect
	modernc.org/ccgo/v3 v3.16.14 // indirect
	modernc.org/libc v1.24.1 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.6.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.1.0 // indirect
)
