.PHONY: build run clean vet fmt lint vuln test coverage check tools service-install service-uninstall service-start service-stop service-logs

-include .env
export

BINARY_NAME ?= lensd

build:
	CGO_LDFLAGS=-Wl,-no_warn_duplicate_libraries go build -o $(BINARY_NAME) ./src

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	$(shell go env GOPATH)/bin/golangci-lint run

vuln:
	$(shell go env GOPATH)/bin/govulncheck ./...

test:
	unset ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; go test -count=1 -p 1 ./src/adapters/ai ./src/adapters/ocr ./src/adapters/im ./src/adapters/im/poller ./src/adapters/im/helpers ./src/adapters/im/store ./src/modules/extractor ./src/modules/capturer ./src/utils/config ./src/pipeline

coverage:
	unset ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; go test -count=1 -p 1 -coverprofile=coverage.out ./src/adapters/ai ./src/adapters/ocr ./src/adapters/im ./src/adapters/im/poller ./src/adapters/im/helpers ./src/adapters/im/store ./src/modules/extractor ./src/modules/capturer ./src/utils/config ./src/pipeline && go tool cover -html=coverage.out -o coverage.html && echo "Coverage report generated: coverage.html"

check: fmt vet lint vuln test

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
