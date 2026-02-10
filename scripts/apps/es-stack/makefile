IK_VERSION := 8.11.1
PLUGINS_ROOT := ./plugins
IK_DIR := $(PLUGINS_ROOT)/analysis-ik
ZIP_FILE := $(PLUGINS_ROOT)/elasticsearch-analysis-ik-$(IK_VERSION).zip
# 这是一个更稳的下载源
DOWNLOAD_URL := https://release.infinilabs.com/analysis-ik/stable/elasticsearch-analysis-ik-$(IK_VERSION).zip
# 数据目录
DATA_DIR := ./es_data

.PHONY: all setup up down reset

all: setup up

setup:
	@echo ">>> 1. 准备插件目录..."
	@mkdir -p $(PLUGINS_ROOT)
	
	@if [ ! -d "$(IK_DIR)" ]; then \
		echo ">>> 2. 未检测到插件，开始下载..."; \
		curl -L -f -o $(ZIP_FILE) $(DOWNLOAD_URL); \
		if [ $$? -ne 0 ]; then \
			echo "❌ 下载失败！请检查网络。"; \
			exit 1; \
		fi; \
		echo ">>> 3. 下载成功，正在解压..."; \
		unzip -q $(ZIP_FILE) -d $(IK_DIR); \
		rm -f $(ZIP_FILE); \
	else \
		echo "✅ 插件已存在。"; \
	fi
	@chmod -R 777 $(PLUGINS_ROOT)

	@echo ">>> 4. 准备数据目录 (防止权限报错)..."
	@mkdir -p $(DATA_DIR)
	@chmod 777 $(DATA_DIR)

up:
	@echo ">>> 启动服务..."
	docker compose up -d
	@echo "✅ 服务启动中... http://localhost:5601"

down:
	docker compose down

reset: down
	@echo ">>> 清理所有环境..."
	rm -rf $(PLUGINS_ROOT)
	rm -f *.zip
	sudo rm -rf $(DATA_DIR)