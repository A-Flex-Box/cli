#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 22:55:00
# original_prompt: 这个setup怎么就一个2/3还有就是我希望将目前的虚拟环境可以检测出来还有对应虚拟环境的解释器
# summary: 修复 AI Setup 步骤计数并增加虚拟环境检测功能
# action: 重写 cmd/ai.go，完善 setup 子命令。新增步骤 [3/3] 用于检测 CONDA_DEFAULT_ENV/VIRTUAL_ENV 及 python 解释器绝对路径。
# expected_outcome: cli ai setup 输出完整的 3/3 步骤，并准确显示当前激活的虚拟环境名称和解释器位置。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在升级 AI Setup 命令 (增加虚拟环境透视)...${NC}"

cat << 'GO_EOF' > cmd/ai.go
package cmd

import (
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
// 1. Setup: 环境自检 (升级版)
// -----------------------------------------------------------
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "检查 GPU、CUDA 及当前虚拟环境详情",
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

		// Step 3: Virtual Env & Interpreter (新增功能)
		fmt.Print("[3/3] Virtual Env:   ")
		
		// 1. 检测环境类型
		envType := "System"
		envName := "None"
		
		// 检查 Conda
		if condaEnv := os.Getenv("CONDA_DEFAULT_ENV"); condaEnv != "" {
			envType = "Conda"
			envName = condaEnv
			fmt.Printf("✅ Active (%s: %s)\n", envType, envName)
		} else if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
			// 检查标准 venv
			envType = "Venv"
			envName = filepath.Base(venv)
			fmt.Printf("✅ Active (%s: %s)\n", envType, envName)
		} else {
			fmt.Printf("⚠️  No Virtual Environment Detected (Using System Python)\n")
		}

		// 2. 获取解释器绝对路径 (Double Check)
		// 有时候环境变量激活了，但 path 没刷新，这一步能通过 python 自身确认真实路径
		checkPath := exec.Command("python3", "-c", "import sys; print(sys.executable)")
		if out, err := checkPath.CombinedOutput(); err == nil {
			realPath := strings.TrimSpace(string(out))
			fmt.Printf("      Interpreter: %s\n", realPath)
		}

		fmt.Println("--------------------------------------------------")
	},
}

// -----------------------------------------------------------
// 2. Init: 标准化目录结构 (保持不变)
// -----------------------------------------------------------
var initCmd = &cobra.Command{
	Use:   "init [project_name]",
	Short: "生成 AI 项目标准目录结构",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pName := args[0]
		structure := map[string]string{
			"data/raw":        "原始不可变数据 (Immutable Source)",
			"data/processed":  "清洗后的特征数据 (Ready for Training)",
			"models":          "模型权重 checkpoints (.pt/.pth)",
			"notebooks":       "Jupyter Notebooks (Exploration)",
			"src":             "源代码 (Source Code)",
			"src/utils":       "工具函数",
			"logs":            "Training Logs",
			"configs":         "Hyperparameters (yaml)",
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

// -----------------------------------------------------------
// 3. Template: 最小训练闭环 (保持不变)
// -----------------------------------------------------------
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
    preds = model(X)            # 1. Forward
    loss = criterion(preds, y)  # 2. Loss
    optimizer.zero_grad()       # 3. Backward
    loss.backward()
    optimizer.step()            # 4. Step

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
echo -e "现在请运行: ${GREEN}bin/cli ai setup${NC} 查看完整的 3/3 步骤。"
