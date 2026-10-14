.PHONY: up down run build test clean lint

up:
	docker-compose up -d

down:
	docker-compose down

run:
	go run ./cmd/cleantrace

build:
	go build -ldflags="-w -s" -o bin/cleantrace ./cmd/cleantrace

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/