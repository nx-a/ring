.DEFAULT_GOAL := build
.PHONY: run build prod clear

include .env
export

clear:
	rm -f app

run: clear
	go run ./cmd/web/main.go

build: clear
	CGO_ENABLED=0 go build -x -o ./app ./cmd/web/main.go

prod: clear
	CGO_ENABLED=0 go build -ldflags="-s -w" -buildvcs=false -o ./app ./cmd/web/main.go