# Contributing to autobrr

Thanks for taking interest in contribution! We welcome anyone who wants to contribute.

If you have an idea for a bigger feature or a change then we are happy to discuss it before you start working on it.  
It is usually a good idea to make sure it aligns with the project and is a good fit.  
Open an issue or post in #dev-general on [Discord](https://discord.gg/WQ2eUycxyT).

This document is a guide to help you through the process of contributing to autobrr.

## Become a contributor

* Code: new features, bug fixes, improvements
* Report bugs
* Documentation: The docs repo can be found here: [github.com/autobrr/autobrr.com](https://github.com/autobrr/autobrr.com)

## Developer guide

This guide helps you get started developing autobrr.

## Dependencies

Make sure you have the following dependencies installed before setting up your developer environment:

- [Git](https://git-scm.com/)
- [Go](https://golang.org/dl/) (see [go.mod](go.mod#L3) for minimum required version)
- [Node.js](https://nodejs.org) (we usually use the latest Node LTS version - for further information see `@types/node` major version in [package.json](web/package.json))
- [pnpm](https://pnpm.io/installation)

## How to contribute

- **Fork and Clone:** [Fork the autobrr repository](https://github.com/autobrr/autobrr/fork) and clone it to start working on your changes.
- **Branching:** Create a new branch for your changes. Use a descriptive name for easy understanding.
  - Checkout a new branch for your fix or feature `git checkout -b fix/filters-issue`
- **Coding:** Ensure your code is well-commented for clarity. With go use `go fmt`
- **Commit Guidelines:** We appreciate the use of [Conventional Commit Guidelines](https://www.conventionalcommits.org/en/v1.0.0/#summary) when writing your commits.
  - Examples: `fix(indexers): Mock improve parsing`, `feat(notifications): add NewService`
  - There is no need for force pushing or rebasing. We squash commits on merge to keep the history clean and manageable.
- **Pull Requests:** Submit a pull request from your Fork with a clear description of your changes. Reference any related issues.
  - Mark it as Draft if it's still in progress.
- **Code Review:** Be open to feedback during the code review process.

## Development environment

The backend is written in Go and the frontend is written in TypeScript using React.

You need to have the Go toolchain installed and Node.js with `pnpm` as the package manager.

Clone the project and change dir:

```shell
git clone github.com/YOURNAME/autobrr && cd autobrr
```

## Frontend

First install the web dependencies:

```shell
cd web && pnpm install
```

Run the project:

```shell
pnpm dev
```

This should make the frontend available at [http://localhost:3000](http://localhost:3000). It's setup to communicate with the API at [http://localhost:7474](http://localhost:7474).

### Build

In order to build binaries of the full application you need to first build the frontend.

To build the frontend, run:

```shell
pnpm --dir web run build
```

## Backend

Install Go dependencies:

```shell
go mod tidy
```

Run the project:

```shell
go run cmd/autobrr/main.go
```

This uses the default `config.toml` and runs the API on [http://localhost:7474](http://localhost:7474).

### Build

To build the backend, run:

```shell
make build/app
```

This will output a binary in `./bin/autobrr`

You can also build the frontend and the backend at once with:

```shell
make build
```

### Build cross-platform binaries

You can optionally build it with [GoReleaser](https://goreleaser.com/) which makes it easy to build cross-platform binaries.

Install it with `go install` or check the [docs for alternatives](https://goreleaser.com/install/):

```shell
go install github.com/goreleaser/goreleaser/v2@latest
```

Then to build binaries, run:

```shell
goreleaser build --snapshot --clean
```

## Tests

The test suite consists of only backend tests at this point. All tests run per commit with GitHub Actions.

### Run backend tests

We have a mix of unit and integration tests.

Run all non-integration tests:

```shell
go test -v ./...
```

### Run SQLite and PostgreSQL integration tests

The integration tests runs against an in memory SQLite database and currently requires Docker for the Postgres tests.

If you have docker setup then run the `test_postgres` container with:

```shell
docker compose up -d test_postgres
```

Then run all tests:

```shell
TZ=UTC go test ./... -tags=integration
```

## Build Docker image

To build a Docker image, run:

```shell
make build/docker
```

The image will be tagged as `autobrr:dev`

To build a cross platform Docker image (for instance if you're running on arm64 or Apple ARM):

```shell
make build/dockerx
```

[Docker multi-platform docs](https://docs.docker.com/build/building/multi-platform/)

## Mock indexer

We have a mock indexer you can run locally that features:

* Built in IRC server that can send announces
* Mock indexer for downloads
* RSS feed mock
* Webhook mock for External Filters

See the documentation [here](./test/mockindexer/README.md). Add the `customDefinitions` to the `config.toml` and then run it with:

```shell
go run test/mockindexer/main.go
```

* Restart the backend API for it to load the new mock.yaml definition
* Then add it via Settings -> Indexers -> Add, and select Mock Indexer in the list
* Go to Settings -> IRC and toggle the IRC network `Mock Indexer`
* Add a new Filter or add the indexer to an existing filter
* Open a new tab and navigate to [http://localhost:3999](http://localhost:3999) and put the example announce in the input then hit enter
