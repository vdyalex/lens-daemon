.PHONY: build run clean vet fmt check service-install service-uninstall service-start service-stop service-logs

build:
	go build -o networkd ./src

run: build
	./networkd

clean:
	rm -f networkd

vet:
	go vet ./...

fmt:
	gofmt -w .

check: fmt vet

# Service management targets
SCRIPTS_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))scripts
SERVICE_PLIST := $(HOME)/Library/LaunchAgents/com.vdyalex.assistant.plist
SERVICE_LOG_DIR := $(HOME)/Library/Logs/ccat

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
