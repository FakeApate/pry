.PHONY: generate generate-sqlc generate-jsonschema build test lint clean

SQLC_CONFIG := resources/sqlc.yaml

generate: generate-sqlc generate-jsonschema

generate-sqlc:
	sqlc generate -f $(SQLC_CONFIG)

generate-jsonschema:
	go-jsonschema -p mullvad -o internal/mullvad/relay.go resources/mullvad_relay_schema.json
	go-jsonschema -p mullvad -o internal/mullvad/ami.go resources/mullvad_ami_schema.json

build:
	go build -o bin/pry .

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
