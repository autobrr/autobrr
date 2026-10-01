module github.com/autobrr/autobrr

go 1.27.0

replace github.com/r3labs/sse/v2 => github.com/autobrr/sse/v2 v2.0.0-20230520125637-530e06346d7d

replace github.com/moistari/rls => github.com/autobrr/rls v0.8.1

require (
	github.com/Hellseher/go-shellquote v0.1.0
	github.com/KimMachineGun/automemlimit v1.0.0
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/Masterminds/squirrel v1.5.4
	github.com/alexedwards/scs/postgresstore v0.0.0-20250417082927-ab20b3feb5e9
	github.com/alexedwards/scs/v2 v2.9.0
	github.com/alphadose/haxmap v1.4.1
	github.com/autobrr/go-cache v1.0.0-rc1
	github.com/autobrr/go-deluge v1.4.0
	github.com/autobrr/go-qbittorrent v1.19.0
	github.com/autobrr/go-rtorrent v1.12.0
	github.com/autobrr/go-torrent v1.1.1
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/avast/retry-go/v4 v4.7.0
	github.com/coreos/go-oidc/v3 v3.21.0
	github.com/dcarbone/zadapters/zstdlog v1.1.0
	github.com/dustin/go-humanize v1.1.0
	github.com/ergochat/irc-go v0.7.0
	github.com/fergusstrange/embedded-postgres v1.34.0
	github.com/fsnotify/fsnotify v1.10.1
	github.com/go-andiamo/splitter v1.2.5
	github.com/go-chi/chi/v5 v5.3.2
	github.com/go-chi/render v1.0.3
	github.com/gosimple/slug v1.15.0
	github.com/hashicorp/go-version v1.9.0
	github.com/hekmon/transmissionrpc/v3 v3.0.0
	github.com/icholy/digest v1.2.0
	github.com/lib/pq v1.12.3
	github.com/magiconair/properties v1.18.12
	github.com/maniartech/signals v1.3.1
	github.com/mmcdole/gofeed v1.5.0
	github.com/moistari/rls v0.6.0
	github.com/mxschmitt/playwright-go v0.6201.1
	github.com/nicholas-fedor/shoutrrr v0.21.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.24.1
	github.com/r3labs/sse/v2 v2.10.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/cors v1.11.1
	github.com/rs/xid v1.6.0
	github.com/rs/zerolog v1.35.1
	github.com/sasha-s/go-deadlock v0.3.9
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.12.1
	go.uber.org/automaxprocs v1.6.0
	golang.org/x/crypto v0.57.0
	golang.org/x/net v0.59.0
	golang.org/x/oauth2 v0.37.0
	golang.org/x/sync v0.23.0
	golang.org/x/term v0.46.0
	golang.org/x/text v0.42.0
	golang.org/x/time v0.16.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.59.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/eclipse/paho.golang v0.23.0 // indirect
	github.com/gdm85/go-rencode v0.1.8 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hekmon/cunits/v2 v2.1.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mmcdole/goxpp/v2 v2.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/petermattis/goid v0.0.0-20250813065127-a731cc31b4fe // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp v0.0.0-20260508232706-74f9aab9d74a // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/tools v0.50.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	mellium.im/reader v0.1.0 // indirect
	mellium.im/sasl v0.3.2 // indirect
	mellium.im/xmlstream v0.15.4 // indirect
	mellium.im/xmpp v0.23.0 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)
