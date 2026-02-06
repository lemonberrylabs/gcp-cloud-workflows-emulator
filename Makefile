.PHONY: build run test docker-build docker-run docker-up docker-down

IMAGE_NAME ?= gcw-emulator
IMAGE_TAG  ?= latest

build:
	go build -o bin/gcw-emulator ./cmd/gcw-emulator

run: build
	./bin/gcw-emulator

test:
	go test ./...

docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-run: docker-build
	docker run --rm -p 8787:8787 -p 8788:8788 $(IMAGE_NAME):$(IMAGE_TAG)

docker-up:
	docker compose up --build

docker-down:
	docker compose down
