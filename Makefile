# ---------------------------------------------------------
#  Config
# ---------------------------------------------------------
BINARY_NAME := cli
BUILD_DIR   := bin
HISTORY_DIR := history/shell

# Git Info
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        := $(shell date +%Y-%m-%dT%H:%M:%S%z)

LDFLAGS     := -X '$(shell go list -m)/cmd.version=$(VERSION)' \
               -X '$(shell go list -m)/cmd.commit=$(COMMIT)' \
               -X '$(shell go list -m)/cmd.date=$(DATE)' \
               -s -w

# Colors (printf compatible)
CC_GREEN  := \033[0;32m
CC_CYAN   := \033[1;36m
CC_RED    := \033[0;31m
CC_YELLOW := \033[1;33m
CC_RESET  := \033[0m

.PHONY: all build clean test help register run install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@printf "$(CC_CYAN)➜  Compiling...$(CC_RESET)\n"
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@printf "$(CC_GREEN)✔  Build ready: $(BUILD_DIR)/$(BINARY_NAME)$(CC_RESET)\n"

# install: 安装到 GOPATH/bin 并检查 PATH
install:
	@printf "$(CC_CYAN)➜  Installing to \$$(go env GOPATH)/bin ...$(CC_RESET)\n"
	@go install -ldflags "$(LDFLAGS)"
	@# 检测安装后的命令是否在 PATH 中可用
	@if command -v $(BINARY_NAME) >/dev/null 2>&1; then \
		printf "$(CC_GREEN)✔  Successfully installed!$(CC_RESET)\n"; \
		printf "   Location: $$(which $(BINARY_NAME))\n"; \
		printf "   You can now run '$(BINARY_NAME)' directly.\n"; \
	else \
		printf "$(CC_RED)✘  Installation successful, but '$(BINARY_NAME)' was NOT found in your PATH.$(CC_RESET)\n"; \
		printf "$(CC_YELLOW)⚠️  Please add the following to your shell profile (~/.bashrc or ~/.zshrc):$(CC_RESET)\n"; \
		printf "   export PATH=\$$PATH:$$(go env GOPATH)/bin\n"; \
	fi

register: build
	@if [ -z "$(FILE)" ]; then \
		printf "$(CC_RED)Error: FILE argument is missing. Usage: make register FILE=script.sh$(CC_RESET)\n"; \
		exit 1; \
	fi
	@printf "$(CC_CYAN)➜  Adding to Project History...$(CC_RESET)\n"
	@$(BUILD_DIR)/$(BINARY_NAME) history add $(FILE)
	@printf "$(CC_CYAN)➜  Archiving file...$(CC_RESET)\n"
	@mkdir -p $(HISTORY_DIR)
	@TS=$$(date +%Y%m%d_%H%M%S); \
	mv $(FILE) $(HISTORY_DIR)/$${TS}_$$(basename $(FILE)); \
	printf "$(CC_GREEN)✔  Registered & Archived to $(HISTORY_DIR)/$${TS}_$$(basename $(FILE))$(CC_RESET)\n"

run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

clean:
	@rm -rf $(BUILD_DIR)

test:
	@go test -v ./...

help:
	@printf "$(CC_CYAN)Available commands:$(CC_RESET)\n"
	@printf "  $(CC_GREEN)build$(CC_RESET)      - 编译项目\n"
	@printf "  $(CC_GREEN)install$(CC_RESET)    - 安装到 GOPATH/bin\n"
	@printf "  $(CC_GREEN)test$(CC_RESET)       - 运行测试\n"
	@printf "  $(CC_GREEN)clean$(CC_RESET)      - 清理构建文件\n"
	@printf "  $(CC_GREEN)run$(CC_RESET)        - 运行命令 (使用 ARGS=...)\n"
	@printf "  $(CC_GREEN)register$(CC_RESET)  - 注册脚本到历史 (使用 FILE=...)\n"
	@printf "\n"
	@printf "$(CC_CYAN)Printer子命令示例:$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --setup\"$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --file document.pdf --printer MyPrinter\"$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --url https://example.com/doc.pdf --auto\"$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --scan --scan-source adf\"$(CC_RESET)\n"
