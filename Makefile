.PHONY: generate generate-sqlc generate-jsonschema build test lint clean

SQLC_CONFIG := resources/sqlc.yaml

# Version metadata, overridable so CI or packagers can supply their own values.
# git describe falls back to "dev" outside a repo; "dirty" marks uncommitted trees.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -X github.com/fakeapate/pry/cmd.version=$(VERSION) \
           -X github.com/fakeapate/pry/cmd.commit=$(COMMIT) \
           -X github.com/fakeapate/pry/cmd.date=$(DATE)

generate: generate-sqlc generate-jsonschema

generate-sqlc:
	sqlc generate -f $(SQLC_CONFIG)

generate-jsonschema:
	go-jsonschema -p mullvad -o internal/mullvad/relay.go resources/mullvad_relay_schema.json
	go-jsonschema -p mullvad -o internal/mullvad/ami.go resources/mullvad_ami_schema.json

build:
	go build -ldflags '$(LDFLAGS)' -o bin/pry .

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
