# ---------------------------------------------------------
#  配置区域
# ---------------------------------------------------------
BINARY_NAME := cli
BUILD_DIR   := bin
# 提取 git 信息用于注入变量
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        := $(shell date +%Y-%m-%dT%H:%M:%S%z)

# 链接参数: 注入 Version/Commit/Date 到 cmd 包的变量中
LDFLAGS     := -X '$(shell go list -m)/cmd.version=$(VERSION)' \
               -X '$(shell go list -m)/cmd.commit=$(COMMIT)' \
               -X '$(shell go list -m)/cmd.date=$(DATE)' \
               -s -w

# ---------------------------------------------------------
#  颜色定义 (使用 printf 兼容格式)
# ---------------------------------------------------------
# \033 是八进制的 ESC
CC_RED    := \033[0;31m
CC_GREEN  := \033[0;32m
CC_YELLOW := \033[0;33m
CC_BLUE   := \033[0;34m
CC_CYAN   := \033[1;36m
CC_RESET  := \033[0m

# ---------------------------------------------------------
#  构建任务
# ---------------------------------------------------------
.PHONY: all build clean test help

all: build

# build: 编译二进制文件到 bin 目录
build:
	@mkdir -p $(BUILD_DIR)
	@printf "$(CC_CYAN)➜  Starting build process...$(CC_RESET)\n"
	@printf "   $(CC_YELLOW)Version:$(CC_RESET) %s\n" "$(VERSION)"
	@printf "   $(CC_YELLOW)Commit: $(CC_RESET) %s\n" "$(COMMIT)"
	@printf "$(CC_BLUE)➜  Compiling Go binary to $(BUILD_DIR)/$(BINARY_NAME)...$(CC_RESET)\n"
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@printf "$(CC_GREEN)✔  Build successful!$(CC_RESET)\n"
	@printf "   Run: $(CC_CYAN)./$(BUILD_DIR)/$(BINARY_NAME) version$(CC_RESET)\n"

# clean: 清理构建产物
clean:
	@printf "$(CC_RED)➜  Cleaning up...$(CC_RESET)\n"
	@rm -rf $(BUILD_DIR)
	@rm -f archive_*.tar.gz
	@printf "$(CC_GREEN)✔  Done.$(CC_RESET)\n"

# test: 运行测试
test:
	@printf "$(CC_CYAN)➜  Running tests...$(CC_RESET)\n"
	@go test -v ./...

# help: 显示帮助
help:
	@printf "$(CC_CYAN)Available Targets:$(CC_RESET)\n"
	@printf "  $(CC_GREEN)make build$(CC_RESET)  - Compile binary to bin/\n"
	@printf "  $(CC_GREEN)make clean$(CC_RESET)  - Remove binaries and archives\n"
	@printf "  $(CC_GREEN)make test$(CC_RESET)   - Run unit tests\n"