module github.com/autobrr/autobrr

go 1.25

replace github.com/r3labs/sse/v2 => github.com/autobrr/sse/v2 v2.0.0-20230520125637-530e06346d7d

require (
	github.com/Hellseher/go-shellquote v0.1.0
	github.com/KimMachineGun/automemlimit v0.7.4
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/Masterminds/squirrel v1.5.4
	github.com/alexedwards/scs/postgresstore v0.0.0-20250417082927-ab20b3feb5e9
	github.com/alexedwards/scs/v2 v2.9.0
	github.com/anacrolix/torrent v1.59.1
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/autobrr/go-deluge v1.3.0
	github.com/autobrr/go-qbittorrent v1.15.0-rc1
	github.com/autobrr/go-rtorrent v1.12.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/avast/retry-go/v4 v4.6.1
	github.com/containrrr/shoutrrr v0.8.0
	github.com/coreos/go-oidc/v3 v3.16.0
	github.com/dcarbone/zadapters/zstdlog v1.1.0
	github.com/dustin/go-humanize v1.0.1
	github.com/ergochat/irc-go v0.4.0
	github.com/fergusstrange/embedded-postgres v1.32.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-andiamo/splitter v1.2.5
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/render v1.0.3
	github.com/gosimple/slug v1.15.0
	github.com/hashicorp/go-version v1.7.0
	github.com/hekmon/transmissionrpc/v3 v3.0.0
	github.com/icholy/digest v1.1.0
	github.com/lib/pq v1.10.9
	github.com/magiconair/properties v1.8.10
	github.com/mmcdole/gofeed v1.3.0
	github.com/moistari/rls v0.6.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.23.2
	github.com/r3labs/sse/v2 v2.10.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/cors v1.11.1
	github.com/rs/zerolog v1.34.0
	github.com/sasha-s/go-deadlock v0.3.6
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/automaxprocs v1.6.0
	golang.org/x/crypto v0.42.0
	golang.org/x/net v0.44.0
	golang.org/x/oauth2 v0.31.0
	golang.org/x/sync v0.17.0
	golang.org/x/term v0.35.0
	golang.org/x/text v0.29.0
	golang.org/x/time v0.13.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.39.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/anacrolix/generics v0.1.0 // indirect
	github.com/anacrolix/missinggo v1.3.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.10.0 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/gdm85/go-rencode v0.1.8 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hekmon/cunits/v2 v2.1.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mmcdole/goxpp v1.1.1-0.20240225020742-a0c311522b23 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/petermattis/goid v0.0.0-20250813065127-a731cc31b4fe // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20250819193227-8b4c13bb791b // indirect
	golang.org/x/sys v0.36.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	lukechampine.com/blake3 v1.1.6 // indirect
	modernc.org/libc v1.66.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
