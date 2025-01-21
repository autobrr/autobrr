.PHONY: all deps test build build/app build/ctl build/web build/docker clean install install-man dev
.POSIX:
.SUFFIXES:

GIT_COMMIT := $(shell git rev-parse HEAD 2> /dev/null)
GIT_TAG := $(shell git describe --abbrev=0 --tags)

SERVICE = autobrr
GO = go
RM = rm
GOFLAGS = "-X main.commit=$(GIT_COMMIT) -X main.version=$(GIT_TAG)"
PREFIX = /usr/local
BINDIR = bin
MANDIR = share/man

all: clean build

deps:
	pnpm --dir web install --frozen-lockfile
	go mod download

test:
	go test $(go list ./... | grep -v test/integration)

build: deps build/web build/app

build/app:
	go build -ldflags $(GOFLAGS) -o bin/$(SERVICE) cmd/$(SERVICE)/main.go

build/ctl:
	go build -ldflags $(GOFLAGS) -o bin/autobrrctl cmd/autobrrctl/main.go

build/web:
	pnpm --dir web run build
	@touch web/dist/.gitkeep 2>/dev/null # To avoid accidental commit of the deletionn

build/docker:
	docker build -t autobrr:dev -f Dockerfile . --build-arg GIT_TAG=$(GIT_TAG) --build-arg GIT_COMMIT=$(GIT_COMMIT)

clean:
	$(RM) -rf bin web/dist/*

install-man:
	mkdir -p $(DESTDIR)$(PREFIX)/$(MANDIR)/man1
	cp -f docs/man/autobrr.1 $(DESTDIR)$(PREFIX)/$(MANDIR)/man1/

install: all install-man
	mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	cp -f bin/$(SERVICE) $(DESTDIR)$(PREFIX)/$(BINDIR)

dev:
	@if ! command -v tmux >/dev/null 2>&1; then \
		echo "tmux is not installed. Please install it to use dev mode."; \
		echo "On Ubuntu/Debian: sudo apt install tmux"; \
		echo "On macOS: brew install tmux"; \
		exit 1; \
	fi
	@tmux new-session -d -s autobrr-dev 'pnpm --dir web dev'
	@tmux split-window -h 'go run cmd/$(SERVICE)/main.go'
	@tmux -2 attach-session -d
