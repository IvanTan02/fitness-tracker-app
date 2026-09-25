.PHONY: run build fmt vet test tidy

run:
	set -a && . ./.env && set +a && go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

fmt:
	gofmt -l .

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy
