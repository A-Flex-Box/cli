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

// 1. Setup: 专注检查 CUDA 和 GPU
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "检查 GPU/CUDA 环境可用性",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔌 AI Environment Diagnostic")
		fmt.Println("--------------------------------------------------")

		// 检查系统级驱动
		fmt.Print("[1/3] NVIDIA Driver: ")
		if cmdPath, err := exec.LookPath("nvidia-smi"); err == nil {
			fmt.Printf("✅ Detected (%s)\n", cmdPath)
			// 获取显卡型号
			out, _ := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader").Output()
			gpuInfo := strings.TrimSpace(string(out))
			fmt.Printf("      GPU: %s\n", gpuInfo)
		} else {
			fmt.Println("❌ Not Found (Training will be slow on CPU)")
		}

		// 检查 PyTorch 调用 CUDA 的能力
		fmt.Print("[2/3] PyTorch CUDA Access: ")
		checkCuda := exec.Command("python3", "-c", "import torch; print(f'{torch.__version__}|{torch.cuda.is_available()}|{torch.version.cuda}')")
		if out, err := checkCuda.CombinedOutput(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), "|")
			if len(parts) == 3 {
				ver, avail, cudaVer := parts[0], parts[1], parts[2]
				if avail == "True" {
					fmt.Printf("✅ Available (Torch v%s, CUDA v%s)\n", ver, cudaVer)
				} else {
					fmt.Printf("❌ Torch Installed (v%s) but CUDA NOT available.\n", ver)
				}
			}
		} else {
			fmt.Println("❌ Python/PyTorch not functional.")
		}
		
		fmt.Println("--------------------------------------------------")
	},
}

// 2. Init: 标准化目录结构
var initCmd = &cobra.Command{
	Use:   "init [project_name]",
	Short: "生成 AI 项目标准目录结构 (Cookiecutter Data Science 风格)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pName := args[0]
		// 业界标准的 AI 项目结构
		structure := map[string]string{
			"data/raw":        "原始不可变数据 (永远不要修改这里)",
			"data/processed":  "清洗和特征工程后的数据 (模型读取这里)",
			"models":          "保存训练好的模型文件 (.pt/.pth)",
			"notebooks":       "Jupyter Notebooks 用于探索性分析 (草稿本)",
			"src":             "正式的源代码 (模型定义, dataset定义)",
			"src/utils":       "工具函数",
			"logs":            "TensorBoard 或 WandB 日志",
			"configs":         "超参数配置文件 (yaml/json)",
		}

		fmt.Printf("🏗  Initializing AI Project: %s\n", pName)
		for path, desc := range structure {
			fullPath := filepath.Join(pName, path)
			os.MkdirAll(fullPath, 0755)
			// 创建 .gitkeep 或者是说明文件
			os.WriteFile(filepath.Join(fullPath, "README.md"), []byte(desc), 0644)
		}
		fmt.Println("✅ Done.")
	},
}

// 3. Template: 最小可运行单元
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "生成最小训练闭环代码 (train.py)",
	Run: func(cmd *cobra.Command, args []string) {
		// 这里保留代码是为了让你有一个可以直接修改的“白板”
		code := `# Minimal PyTorch Training Loop
import torch
import torch.nn as nn
import torch.optim as optim

# A. 数据准备 (Data)
# 实际项目中这里会替换为 DataLoader
X = torch.tensor([[1.0], [2.0], [3.0]], device='cpu')
y = torch.tensor([[2.0], [4.0], [6.0]], device='cpu')

# B. 模型定义 (Model Architecture)
model = nn.Linear(1, 1) # y = wx + b

# C. 定义衡量标准与优化器 (Loss & Optimizer)
criterion = nn.MSELoss()                  # 衡量 预测值 和 真实值 差多少
optimizer = optim.SGD(model.parameters(), lr=0.01) # 决定怎么调整 w 和 b

# D. 训练循环 (The Training Loop)
print("Start Training...")
for epoch in range(100):
    # 1. Forward (前向传播: 猜结果)
    preds = model(X)
    
    # 2. Loss (计算误差: 猜错多少?)
    loss = criterion(preds, y)
    
    # 3. Backward (反向传播: 谁背锅?)
    # 计算 loss 对每个参数(w, b)的梯度(导数)
    optimizer.zero_grad() 
    loss.backward()
    
    # 4. Step (参数更新: 改正错误)
    # w_new = w_old - lr * gradient
    optimizer.step()

print(f"Final Model: y = {model.weight.item():.2f}x + {model.bias.item():.2f}")
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
