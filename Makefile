APP_NAME   := stock-ticker
BINARY_DIR := bin
BINARY     := $(BINARY_DIR)/$(APP_NAME)
MAIN       := ./cmd/app

.PHONY: all build run test lint clean docker-build docker-run

all: build

## build: compile the binary
build:
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY) $(MAIN)

## run: run the service (requires FINNHUB_API_KEY to be set)
run:
	go run $(MAIN)

## test: run all tests
test:
	go test ./... -v -race -count=1

## lint: run golangci-lint (install: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## clean: remove build artifacts
clean:
	rm -rf $(BINARY_DIR)

## docker-build: build Docker image
docker-build:
	docker build -t $(APP_NAME):latest .

## docker-run: run Docker container (requires FINNHUB_API_KEY env var)
docker-run:
	docker run --rm \
		-e FINNHUB_API_KEY=$(FINNHUB_API_KEY) \
		-p 8080:8080 \
		$(APP_NAME):latest
