.DEFAULT_GOAL := build
.PHONY: run build prod clear ui ui-dev lint test

include .env
export

clear:
	rm -f app

ui:
	cd web && npm install && npm run build

ui-dev:
	cd web && npm run dev

run: clear
	go run ./cmd/web/main.go

build: ui clear
	CGO_ENABLED=0 go build -x -o ./app ./cmd/web/main.go

prod: ui clear
	CGO_ENABLED=0 go build -ldflags="-s -w" -buildvcs=false -o ./app ./cmd/web/main.go

test:
	go test ./...

lint:
	golangci-lint run ./...
