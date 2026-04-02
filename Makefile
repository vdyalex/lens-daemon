.PHONY: build clean generate validate format lint vulnerabilities test coverage check tools daemon start stop status restart logs test-integration

-include .env
export

BINARY_NAME ?= lensd
BINARY_PATH ?= ./bin/$(BINARY_NAME)
XPATH = github.com/vdyalex/lens-daemon/src/utils/buildinfo.BinaryName

build:
	CGO_LDFLAGS=-Wl,-no_warn_duplicate_libraries go build -ldflags "-X '$(XPATH)=$(BINARY_NAME)'" -o $(BINARY_PATH) ./src

clean:
	find bin -type f ! -name '.gitignore' -delete 2>/dev/null || true

generate:
	export PATH=$$PATH:$$(go env GOPATH)/bin && go generate ./...

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
	unset ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; go test -count=1 -p 1 -coverprofile=coverage/coverage.out ./... && go tool cover -html=coverage/coverage.out -o coverage/coverage.html && echo "Coverage report generated: coverage/coverage.html"

test-integration:
	unset ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; go test -count=1 -p 1 -run Integration ./src/ipc/... ./src/daemon/...

check: format validate lint vulnerabilities test test-integration

# Daemon management convenience targets
daemon: build
	$(BINARY_PATH) daemon

start: build
	$(BINARY_PATH) start

stop:
	$(BINARY_PATH) stop

status:
	$(BINARY_PATH) status

restart: build
	$(BINARY_PATH) restart

logs:
	$(BINARY_PATH) logs

tools:
	@echo "Installing analysis and code generation tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install go.uber.org/mock/mockgen@latest
