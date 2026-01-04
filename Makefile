BINARY_NAME := cli
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        := $(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS     := -X '$(shell go list -m)/cmd.version=$(VERSION)' \
               -X '$(shell go list -m)/cmd.commit=$(COMMIT)' \
               -X '$(shell go list -m)/cmd.date=$(DATE)' -s -w

# Colors
B_GREEN  := \033[1;32m
B_CYAN   := \033[1;36m
RESET    := \033[0m

all: build

build:
	@echo "$(B_CYAN)➜ Building Binary...$(RESET)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) main.go
	@echo "$(B_GREEN)✔ Build Success: ./$(BINARY_NAME)$(RESET)"

clean:
	@rm -f $(BINARY_NAME) archive_*.tar.gz
	@echo "Cleaned."

test:
	@go test -v ./...
