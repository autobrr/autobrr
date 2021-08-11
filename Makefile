.PHONY: test
.POSIX:
.SUFFIXES:

SERVICE = autobrr
GO = go
RM = rm
GOFLAGS =
PREFIX = /usr/local
BINDIR = bin

all: clean build

deps:
	cd web && yarn install
	go mod download

test:
	go test $(go list ./... | grep -v test/integration)

build: deps build/web build/app

build/app:
	go build -o bin/$(SERVICE) cmd/$(SERVICE)/main.go

build/web:
	cd web && yarn build

build/docker:
	docker build -t autobrr:dev -f Dockerfile .

clean:
	$(RM) -rf bin

install: all
	echo $(DESTDIR)$(PREFIX)/$(BINDIR)
	mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	cp -f bin/$(SERVICE) $(DESTDIR)$(PREFIX)/$(BINDIR)
