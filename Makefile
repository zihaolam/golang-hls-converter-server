# Go compiler
GO := go

# Binary name
BINARY_NAME := bin/app

.PHONY: all build run clean

all: build

build:
	$(GO) build -o $(bin/app) cmd/app/main.go

run: build
	./$(BINARY_NAME)

dev:
	air --build.cmd "$(GO) build -o bin/app cmd/app/main.go" --build.bin "./bin/app"

clean:
	rm -f $(BINARY_NAME)