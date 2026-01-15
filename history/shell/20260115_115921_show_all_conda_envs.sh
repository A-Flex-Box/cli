#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 23:10:00
# original_prompt: 我现在希望的是虚拟环境会显示目前所有的虚拟环境以及当前选择的虚拟环境
# summary: 升级 setup 命令，支持列出所有 Conda 虚拟环境并高亮当前环境
# action: 修改 cmd/ai.go，在 setup 步骤 [3/3] 中，检测 conda 命令，如果存在则输出 conda env list 的结果，否则回退到仅显示当前激活环境。
# expected_outcome: bin/cli ai setup 将列出所有可用环境，并清晰指出当前处于哪个环境中。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在升级 AI Setup (集成 Conda 环境列表)...${NC}"

cat << 'GO_EOF' > cmd/ai.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI 工程化辅助工具",
}

// -----------------------------------------------------------
// 1. Setup: 环境自检 (支持列表显示)
// -----------------------------------------------------------
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "检查 GPU、CUDA 及虚拟环境列表",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔌 AI Environment Diagnostic")
		fmt.Println("--------------------------------------------------")

		// Step 1: NVIDIA Driver
		fmt.Print("[1/3] NVIDIA Driver: ")
		if cmdPath, err := exec.LookPath("nvidia-smi"); err == nil {
			fmt.Printf("✅ Detected (%s)\n", cmdPath)
			out, _ := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader").Output()
			gpuInfo := strings.TrimSpace(string(out))
			fmt.Printf("      GPU: %s\n", gpuInfo)
		} else {
			fmt.Println("❌ Not Found (Running on CPU mode)")
		}

		// Step 2: PyTorch CUDA
		fmt.Print("[2/3] PyTorch Stack: ")
		checkCuda := exec.Command("python3", "-c", "import torch; print(f'{torch.__version__}|{torch.cuda.is_available()}|{torch.version.cuda}')")
		if out, err := checkCuda.CombinedOutput(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), "|")
			if len(parts) == 3 {
				ver, avail, cudaVer := parts[0], parts[1], parts[2]
				if avail == "True" {
					fmt.Printf("✅ Ready (Torch v%s + CUDA v%s)\n", ver, cudaVer)
				} else {
					fmt.Printf("⚠️  Torch v%s Installed (No CUDA)\n", ver)
				}
			}
		} else {
			fmt.Println("❌ Python/PyTorch not working.")
		}

		// Step 3: Virtual Environments (List All)
		fmt.Println("[3/3] Virtual Environments:")
		
		// 尝试检测 Conda
		if _, err := exec.LookPath("conda"); err == nil {
			// 如果有 conda，列出所有环境
			cmd := exec.Command("conda", "env", "list")
			stdout, _ := cmd.StdoutPipe()
			cmd.Start()

			scanner := bufio.NewScanner(stdout)
			foundEnvs := false
			for scanner.Scan() {
				line := scanner.Text()
				// 跳过注释行
				if strings.HasPrefix(line, "#") {
					continue
				}
				if strings.TrimSpace(line) == "" {
					continue
				}
				
				foundEnvs = true
				// 简单的格式化：给 active 环境加绿色箭头，其他的缩进
				if strings.Contains(line, "*") {
					// Conda 输出里当前环境带星号
					// 替换星号为更显眼的标记，或者保持原样但加颜色
					fmt.Printf("      👉 \033[1;32m%s\033[0m\n", line) // Green Highlight
				} else {
					fmt.Printf("         %s\n", line)
				}
			}
			cmd.Wait()

			if !foundEnvs {
				fmt.Println("      (Conda installed but no environments found?)")
			}

		} else {
			// 如果没有 Conda，回退到原来的逻辑 (只显示当前 Active 的)
			fmt.Println("      (Conda not found, checking active VENV only)")
			if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
				envName := filepath.Base(venv)
				fmt.Printf("      👉 Active Venv: \033[1;32m%s\033[0m (%s)\n", envName, venv)
			} else {
				fmt.Println("      ⚠️  No Active Virtual Environment")
			}
		}

		// 显示当前 Python 解释器路径 (Double Check)
		checkPath := exec.Command("python3", "-c", "import sys; print(sys.executable)")
		if out, err := checkPath.CombinedOutput(); err == nil {
			realPath := strings.TrimSpace(string(out))
			fmt.Printf("\n      Interpreter: %s\n", realPath)
		}

		fmt.Println("--------------------------------------------------")
	},
}

// 2. Init: 标准化目录结构
var initCmd = &cobra.Command{
	Use:   "init [project_name]",
	Short: "生成 AI 项目标准目录结构",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pName := args[0]
		structure := map[string]string{
			"data/raw":        "原始不可变数据",
			"data/processed":  "清洗后的特征数据",
			"models":          "模型权重 checkpoints",
			"notebooks":       "Jupyter Notebooks",
			"src":             "源代码",
			"src/utils":       "工具函数",
			"logs":            "Training Logs",
			"configs":         "Hyperparameters",
		}
		fmt.Printf("🏗  Initializing Project: %s\n", pName)
		for path, desc := range structure {
			fullPath := filepath.Join(pName, path)
			os.MkdirAll(fullPath, 0755)
			os.WriteFile(filepath.Join(fullPath, "README.md"), []byte(desc), 0644)
		}
		fmt.Println("✅ Done.")
	},
}

// 3. Template: 最小训练闭环
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "生成最小训练闭环代码 (train.py)",
	Run: func(cmd *cobra.Command, args []string) {
		code := `# Minimal PyTorch Training Loop
import torch
import torch.nn as nn
import torch.optim as optim

X = torch.tensor([[1.0], [2.0], [3.0]], device='cpu')
y = torch.tensor([[2.0], [4.0], [6.0]], device='cpu')

model = nn.Linear(1, 1) 
criterion = nn.MSELoss()
optimizer = optim.SGD(model.parameters(), lr=0.01)

print("Start Training...")
for epoch in range(100):
    preds = model(X)
    loss = criterion(preds, y)
    optimizer.zero_grad()
    loss.backward()
    optimizer.step()

print(f"Result: y = {model.weight.item():.2f}x + {model.bias.item():.2f}")
`
		os.WriteFile("train.py", []byte(code), 0644)
		fmt.Println("✅ train.py created.")
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(setupCmd)
	aiCmd.AddCommand(initCmd)
	aiCmd.AddCommand(templateCmd)
}
GO_EOF

echo -e "${GREEN}-> 重新编译...${NC}"
make build

echo -e "${GREEN}=== 升级完成 ===${NC}"
echo -e "请运行: ${GREEN}bin/cli ai setup${NC} 查看环境列表。"
