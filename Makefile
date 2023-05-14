.PHONY: test
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

all: clean build

deps:
	cd web && pnpm install
	go mod download

test:
	go test $(go list ./... | grep -v test/integration)

build: deps build/web build/app

build/app:
	go build -ldflags $(GOFLAGS) -o bin/$(SERVICE) cmd/$(SERVICE)/main.go

build/ctl:
	go build -ldflags $(GOFLAGS) -o bin/autobrrctl cmd/autobrrctl/main.go

build/web:
	cd web && pnpm run build

build/docker:
	docker build -t autobrr:dev -f Dockerfile . --build-arg GIT_TAG=$(GIT_TAG) --build-arg GIT_COMMIT=$(GIT_COMMIT)

clean:
	$(RM) -rf bin

install: all
	echo $(DESTDIR)$(PREFIX)/$(BINDIR)
	mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	cp -f bin/$(SERVICE) $(DESTDIR)$(PREFIX)/$(BINDIR)
