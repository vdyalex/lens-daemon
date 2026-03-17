.PHONY: build run clean validate format lint vulnerabilities test coverage check tools service-install service-uninstall service-start service-stop service-logs

-include .env
export

BINARY_NAME ?= lensd
BINARY_PATH ?= ./bin/$(BINARY_NAME)

build:
	CGO_LDFLAGS=-Wl,-no_warn_duplicate_libraries go build -o $(BINARY_PATH) ./src

run: build
	./$(BINARY_PATH)

clean:
	find bin -type f ! -name '.gitignore' -delete 2>/dev/null || true

validate:
	go vet ./...

format:
	gofmt -w .

lint:
	$(shell go env GOPATH)/bin/golangci-lint run

vulnerabilities:
	$(shell go env GOPATH)/bin/govulncheck ./...

test:
	unset ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; go test -count=1 -p 1 ./...

coverage:
	unset ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; go test -count=1 -p 1 -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html && echo "Coverage report generated: coverage.html"

check: format validate lint vulnerabilities test

tools:
	@echo "Installing analysis tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

# Service management targets
SCRIPTS_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))scripts
SERVICE_PLIST := $(HOME)/Library/LaunchAgents/com.vdyalex.lensd.plist
SERVICE_LOG_DIR := $(HOME)/Library/Logs/lens

service-install:
	@bash $(SCRIPTS_DIR)/service-install.sh

service-uninstall:
	@bash $(SCRIPTS_DIR)/service-uninstall.sh

service-start:
	launchctl load -w "$(SERVICE_PLIST)"

service-stop:
	launchctl unload "$(SERVICE_PLIST)"

service-logs:
	tail -f "$(SERVICE_LOG_DIR)/stdout.log" "$(SERVICE_LOG_DIR)/stderr.log"
