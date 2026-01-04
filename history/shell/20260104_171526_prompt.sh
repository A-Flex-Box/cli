#!/bin/bash
set -e

# ==========================================
# 0. 配置与颜色
# ==========================================
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}=== Shell 脚本归档整理工具 ===${NC}"
echo -e "${YELLOW}正在将所有脚本迁移至 history/shell/ ...${NC}"

# 目标目录
TARGET_DIR="history/shell"
mkdir -p "$TARGET_DIR"

# 获取当前脚本名，防止在循环中被过早移动
CURRENT_SCRIPT=$(basename "$0")

# ==========================================
# 1. 执行移动操作
# ==========================================
count=0

# 遍历所有 .sh 文件
for file in *.sh; do
    # 检查文件是否存在（处理没有 .sh 文件的情况）
    [ -e "$file" ] || continue
    
    # 跳过当前正在运行的脚本（最后再移动）
    if [ "$file" == "$CURRENT_SCRIPT" ]; then
        continue
    fi

    # 获取文件修改时间作为时间戳
    # 尝试 macOS stat 语法 (-f), 失败则尝试 Linux date 语法 (-r)
    if stat -f "%Sm" -t "%Y%m%d_%H%M%S" "$file" >/dev/null 2>&1; then
        # macOS
        TS=$(stat -f "%Sm" -t "%Y%m%d_%H%M%S" "$file")
    else
        # Linux (GNU date)
        TS=$(date -r "$file" +"%Y%m%d_%H%M%S")
    fi

    # 移动并重命名
    NEW_NAME="${TS}_${file}"
    echo -e "-> 归档: ${file} \t=> ${TARGET_DIR}/${NEW_NAME}"
    mv "$file" "$TARGET_DIR/$NEW_NAME"
    ((count++))
done

echo -e "${GREEN}成功归档了 $count 个脚本。${NC}"

# ==========================================
# 2. 更新 history.json
# ==========================================
echo -e "${BLUE}-> 更新 history.json 记录...${NC}"

# 去掉末尾的 "]"
sed -i.bak '$ s/]$//' history/history.json 2>/dev/null || true

# 准备新记录
CURRENT_TIME=$(date "+%Y-%m-%d %H:%M:%S")
cat << EOF >> history/history.json
  ,
  {
    "timestamp": "${CURRENT_TIME}",
    "original_prompt": "注意其实现在所有的shell应该在cli/history/shell目录下并且附带上创建时间戳在文件名里",
    "summary": "项目目录重构 (Script Organization)",
    "action": "创建 history/shell 目录，将根目录所有 .sh 脚本按创建时间戳重命名并归档",
    "expected_outcome": "根目录保持整洁，所有历史脚本有序存放在 history/shell/ 中"
  }
]
EOF

# 清理 sed 备份
rm -f history/history.json.bak

# ==========================================
# 3. 自我归档
# ==========================================
echo -e "${YELLOW}-> 最后一步: 将本脚本也归档...${NC}"

# 获取当前时间作为本脚本的时间戳
NOW_TS=$(date "+%Y%m%d_%H%M%S")
mv "$CURRENT_SCRIPT" "$TARGET_DIR/${NOW_TS}_${CURRENT_SCRIPT}"

echo -e "${GREEN}=== 全部完成！ ===${NC}"
echo -e "脚本已移动至: ${TARGET_DIR}/${NOW_TS}_${CURRENT_SCRIPT}"
echo -e "现在的根目录应该非常干净了。"