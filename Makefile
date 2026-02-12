# ---------------------------------------------------------
#  Config (cross-platform: Linux, macOS, Windows)
# ---------------------------------------------------------
BUILD_DIR   := bin
HISTORY_DIR := history/shell

# OS detection: Windows sets OS=Windows_NT
ifeq ($(OS),Windows_NT)
  BINARY_NAME := cli.exe
  # Windows: use cmd.exe for recipes that need it
  _IS_WIN := 1
else
  BINARY_NAME := cli
  _IS_WIN := 0
endif

# Git Info (use git for date - cross-platform, avoids `date` format differences)
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
# Use git for date (cross-platform; avoids Unix `date` format differences)
DATE        := $(shell git log -1 --format=%cI 2>/dev/null || echo "unknown")

LDFLAGS     := -X '$(shell go list -m)/cmd.version=$(VERSION)' \
               -X '$(shell go list -m)/cmd.commit=$(COMMIT)' \
               -X '$(shell go list -m)/cmd.date=$(DATE)' \
               -s -w

# 静态链接：禁用 CGO 避免依赖 glibc，可在老系统运行
CGO_ENV     := CGO_ENABLED=0

# Colors (printf compatible, skip on Windows if terminal doesn't support)
CC_GREEN  := \033[0;32m
CC_CYAN   := \033[1;36m
CC_RED    := \033[0;31m
CC_YELLOW := \033[1;33m
CC_RESET  := \033[0m

# Cross-platform mkdir (used by build-windows, build-all)
ifeq ($(_IS_WIN),1)
  MKDIR_P = if not exist $(1) mkdir $(1)
else
  MKDIR_P = mkdir -p $(1)
endif

.PHONY: all build clean test help register run install build-windows build-all deps

all: build

deps:
	@go mod tidy
	@printf "$(CC_GREEN)✔  Dependencies updated$(CC_RESET)\n"

# --- Unix/Linux/macOS build ---
ifneq ($(_IS_WIN),1)
build:
	@mkdir -p $(BUILD_DIR)
	@printf "$(CC_CYAN)➜  Compiling...$(CC_RESET)\n"
	@$(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@printf "$(CC_GREEN)✔  Build ready: $(BUILD_DIR)/$(BINARY_NAME)$(CC_RESET)\n"

clean:
	@rm -rf $(BUILD_DIR)

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

install:
	@printf "$(CC_CYAN)➜  Installing to \$$(go env GOPATH)/bin ...$(CC_RESET)\n"
	@go install -ldflags "$(LDFLAGS)"
	@if command -v $(BINARY_NAME) >/dev/null 2>&1; then \
		printf "$(CC_GREEN)✔  Successfully installed!$(CC_RESET)\n"; \
		printf "   Location: $$(which $(BINARY_NAME))\n"; \
		printf "   You can now run '$(BINARY_NAME)' directly.\n"; \
	else \
		printf "$(CC_RED)✘  Installation successful, but '$(BINARY_NAME)' was NOT found in your PATH.$(CC_RESET)\n"; \
		printf "$(CC_YELLOW)⚠️  Please add the following to your shell profile (~/.bashrc or ~/.zshrc):$(CC_RESET)\n"; \
		printf "   export PATH=\$$PATH:$$(go env GOPATH)/bin\n"; \
	fi
endif

# --- Windows native build (when running make on Windows) ---
ifeq ($(_IS_WIN),1)
build:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	@echo Compiling...
	@$(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@echo Build ready: $(BUILD_DIR)/$(BINARY_NAME)

clean:
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)

register: build
	@echo Error: register uses Unix tools (date, basename, mv). Use Git Bash or WSL on Windows.
	@exit /b 1

run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

install:
	@echo Installing to GOPATH\bin ...
	@go install -ldflags "$(LDFLAGS)"
	@where $(BINARY_NAME) >nul 2>&1 && echo Successfully installed! || echo Add GOPATH\bin to PATH if needed.
endif

# --- Cross-compilation (from any host) ---
ifneq ($(_IS_WIN),1)
build-windows:
	@$(call MKDIR_P,$(BUILD_DIR))
	@echo Cross-compiling for Windows amd64...
	@$(CGO_ENV) GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cli.exe main.go
	@echo Build ready: $(BUILD_DIR)/cli.exe

build-all: build
	@echo Cross-compiling for Windows amd64...
	@$(CGO_ENV) GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cli-windows-amd64.exe main.go
	@echo All builds in $(BUILD_DIR)/
else
build-windows:
	@$(call MKDIR_P,$(BUILD_DIR))
	@echo Cross-compiling for Windows amd64 (already on Windows)...
	@set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cli.exe main.go
	@echo Build ready: $(BUILD_DIR)/cli.exe

build-all: build
	@echo Same-OS build already done. Use build for Windows binary.
endif

test:
	@go test -v ./...

help:
	@printf "$(CC_CYAN)Available commands:$(CC_RESET)\n"
	@printf "  $(CC_GREEN)build$(CC_RESET)           - 编译项目 (当前平台)\n"
	@printf "  $(CC_GREEN)build-windows$(CC_RESET)   - 交叉编译 Windows amd64 版本\n"
	@printf "  $(CC_GREEN)build-all$(CC_RESET)      - 编译当前平台 + Windows amd64\n"
	@printf "  $(CC_GREEN)install$(CC_RESET)        - 安装到 GOPATH/bin\n"
	@printf "  $(CC_GREEN)test$(CC_RESET)            - 运行测试\n"
	@printf "  $(CC_GREEN)clean$(CC_RESET)           - 清理构建文件\n"
	@printf "  $(CC_GREEN)run$(CC_RESET)             - 运行命令 (使用 ARGS=...)\n"
	@printf "  $(CC_GREEN)register$(CC_RESET)       - 注册脚本到历史 (使用 FILE=...)\n"
	@printf "\n"
	@printf "$(CC_CYAN)Printer子命令示例:$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --setup\"$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --file document.pdf --printer MyPrinter\"$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --url https://example.com/doc.pdf --auto\"$(CC_RESET)\n"
	@printf "  $(CC_YELLOW)make run ARGS=\"printer --scan --scan-source adf\"$(CC_RESET)\n"
