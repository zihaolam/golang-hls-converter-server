# Go compiler
GO := go

# Binary name
BINARY_NAME := bin/app

.PHONY: all build run clean

all: build

build:
	$(GO) build -o $(BINARY_NAME) cmd/app/main.go

run: build
	./$(BINARY_NAME)

dev:
	air --build.cmd "$(GO) build --race -o $(BINARY_NAME) cmd/app/main.go" --build.bin "$(BINARY_NAME)"

clean:
	rm -f $(BINARY_NAME)