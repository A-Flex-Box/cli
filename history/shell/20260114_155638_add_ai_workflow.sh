
#!/bin/bash

# METADATA_START

# timestamp: 2026-01-04 22:00:00

# original_prompt: 需要加一个主命令叫做环境一键装配,比如一键装配coda环境,但是我的本意不只是搭建环境我更希望做到模型的训练或者说其他相关的高频的操作集成到cli,我现在有一个大概的方向,主要就是做模型设计训练所以对于这个人工智能方向来说从环境数据集训练等方向来说,当然我不懂训练模型这些应该怎么做,我希望你给我十足的建议以保证我在刚开始学习时有一个明确的规划

# summary: 新增 AI 工程化模块 (环境检查、项目初始化、代码模板)

# action: 创建 cmd/ai.go，实现 setup(环境检查), init(目录结构), template(生成训练脚本), roadmap(学习路径) 四个子命令。

# expected_outcome: bin/cli ai 命令可用，用户可以通过 cli ai template 快速获得可运行的 PyTorch 训练代码，通过 cli ai roadmap 查看学习规划。

# METADATA_END



set -e

RED='\033[0;31m'

GREEN='\033[0;32m'

CYAN='\033[1;36m'

NC='\033[0m'



echo -e "${GREEN}-> 正在构建 AI 工程化模块 (cmd/ai.go)...${NC}"



# ========================================================

# 生成 cmd/ai.go

# ========================================================

cat << 'GO_EOF' > cmd/ai.go

package cmd



import (

	"fmt"

	"os"

	"os/exec"

	"path/filepath"



	"github.com/spf13/cobra"

)



// aiCmd: 主命令

var aiCmd = &cobra.Command{

	Use:   "ai",

	Short: "AI 模型训练与工程化工具箱",

	Long:  `提供从环境检查、项目初始化到代码生成的 AI 全流程辅助工具。`,

}



// -----------------------------------------------------------

// 1. Setup: 环境检查与建议 (环境装配)

// -----------------------------------------------------------

var setupCmd = &cobra.Command{

	Use:   "setup",

	Short: "检查 CUDA 和 Python 环境",

	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("🔍 Checking AI Environment...")

		fmt.Println("--------------------------------------------------")



		// Check Nvidia Driver

		fmt.Print("1. Checking GPU/CUDA (nvidia-smi): ")

		if _, err := exec.LookPath("nvidia-smi"); err == nil {

			fmt.Println("✅ Detected")

			out, _ := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader").Output()

			fmt.Printf("   %s", out)

		} else {

			fmt.Println("❌ Not Found")

			fmt.Println("   👉 Advice: Install Nvidia Drivers and CUDA Toolkit: https://developer.nvidia.com/cuda-downloads")

		}



		// Check Python

		fmt.Print("2. Checking Python: ")

		if path, err := exec.LookPath("python3"); err == nil {

			fmt.Printf("✅ Detected (%s)\n", path)

			// Check PyTorch

			fmt.Print("   Checking PyTorch: ")

			checkTorch := exec.Command("python3", "-c", "import torch; print(torch.__version__)")

			if out, err := checkTorch.CombinedOutput(); err == nil {

				fmt.Printf("✅ Installed (v%s)", out)

				// Check CUDA availability in Torch

				checkCuda := exec.Command("python3", "-c", "import torch; print(torch.cuda.is_available())")

				outCuda, _ := checkCuda.CombinedOutput()

				fmt.Printf("   Torch CUDA Available: %s", outCuda)

			} else {

				fmt.Println("❌ Not Found")

				fmt.Println("   👉 Advice: pip install torch torchvision torchaudio")

			}

		} else {

			fmt.Println("❌ Not Found")

			fmt.Println("   👉 Advice: Install Miniconda (Highly Recommended for AI): https://docs.conda.io/en/latest/miniconda.html")

		}

		fmt.Println("--------------------------------------------------")

	},

}



// -----------------------------------------------------------

// 2. Init: 初始化项目结构 (工程化)

// -----------------------------------------------------------

var initCmd = &cobra.Command{

	Use:   "init [project_name]",

	Short: "初始化标准的 AI 项目目录结构",

	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {

		pName := args[0]

		dirs := []string{

			pName + "/data/raw",         // 原始数据

			pName + "/data/processed",   // 处理后的数据

			pName + "/src/models",       // 模型代码

			pName + "/src/utils",        // 工具函数

			pName + "/checkpoints",      // 训练好的模型保存

			pName + "/logs",             // TensorBoard 日志

			pName + "/notebooks",        // Jupyter 实验本

		}



		fmt.Printf("🏗  Scaffolding AI Project: %s\n", pName)

		for _, d := range dirs {

			if err := os.MkdirAll(d, 0755); err == nil {

				fmt.Printf("   ✅ Created %s\n", d)

			} else {

				fmt.Printf("   ❌ Error creating %s: %v\n", d, err)

			}

		}

		

		// 创建一个 README

		readmePath := filepath.Join(pName, "README.md")

		readmeContent := fmt.Sprintf("# %s\n\nAI Project initialized by CLI tool.\n", pName)

		os.WriteFile(readmePath, []byte(readmeContent), 0644)

		

		fmt.Println("\n🚀 Project ready! cd " + pName)

	},

}



// -----------------------------------------------------------

// 3. Template: 生成训练代码 (教学与起步)

// -----------------------------------------------------------

var templateCmd = &cobra.Command{

	Use:   "template",

	Short: "生成基础的 PyTorch 训练脚本 (train.py)",

	Run: func(cmd *cobra.Command, args []string) {

		code := `import torch

import torch.nn as nn

import torch.optim as optim



# 1. Define a simple model (Linear Regression)

class SimpleModel(nn.Module):

    def __init__(self):

        super(SimpleModel, self).__init__()

        self.linear = nn.Linear(1, 1)  # One input, one output



    def forward(self, x):

        return self.linear(x)



# 2. Setup Device

device = torch.device("cuda" if torch.cuda.is_available() else "cpu")

print(f"Using device: {device}")



# 3. Data (Dummy Data)

X = torch.tensor([[1.0], [2.0], [3.0], [4.0]], device=device)

y = torch.tensor([[2.0], [4.0], [6.0], [8.0]], device=device) # y = 2x



# 4. Training Loop

model = SimpleModel().to(device)

criterion = nn.MSELoss()

optimizer = optim.SGD(model.parameters(), lr=0.01)



print("🚀 Starting Training...")

for epoch in range(100):

    model.train()

    

    # Forward pass

    outputs = model(X)

    loss = criterion(outputs, y)

    

    # Backward and optimize

    optimizer.zero_grad()

    loss.backward()

    optimizer.step()

    

    if (epoch+1) % 10 == 0:

        print(f'Epoch [{epoch+1}/100], Loss: {loss.item():.4f}')



# 5. Test

model.eval()

with torch.no_grad():

    test_input = torch.tensor([[5.0]], device=device)

    prediction = model(test_input)

    print(f"Prediction for input 5.0 (Expected 10.0): {prediction.item():.4f}")

`

		err := os.WriteFile("train.py", []byte(code), 0644)

		if err != nil {

			fmt.Printf("❌ Error writing file: %v\n", err)

			return

		}

		fmt.Println("✅ Generated 'train.py'. Run it with: python3 train.py")

		fmt.Println("   This is a simple PyTorch 'Hello World' to ensure your env is working.")

	},

}



// -----------------------------------------------------------

// 4. Roadmap: 学习建议 (规划)

// -----------------------------------------------------------

var roadmapCmd = &cobra.Command{

	Use:   "roadmap",

	Short: "查看 AI 学习与开发路线图",

	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("🗺️  AI Development Roadmap (Guided by CLI)")

		fmt.Println("--------------------------------------------------")

		fmt.Println("🟢 Phase 1: Environment (Foundations)")

		fmt.Println("   Command: cli ai setup")

		fmt.Println("   Goal: Ensure NVIDIA Drivers, CUDA, and Python/Conda are ready.")

		fmt.Println("")

		fmt.Println("🔵 Phase 2: Project Structure (Engineering)")

		fmt.Println("   Command: cli ai init my_project")

		fmt.Println("   Goal: Don't put everything in one file. Organize data, src, and logs.")

		fmt.Println("")

		fmt.Println("🟡 Phase 3: First Model (Hello World)")

		fmt.Println("   Command: cli ai template")

		fmt.Println("   Goal: Run a simple PyTorch script to understand the Training Loop (Forward -> Loss -> Backward).")

		fmt.Println("")

		fmt.Println("🟠 Phase 4: Data Pipeline (The Hard Part)")

		fmt.Println("   Action: Learn pandas and torch.utils.data.Dataset.")

		fmt.Println("   Goal: Clean your data and verify it before throwing it into a model.")

		fmt.Println("--------------------------------------------------")

	},

}



func init() {

	rootCmd.AddCommand(aiCmd)

	aiCmd.AddCommand(setupCmd)

	aiCmd.AddCommand(initCmd)

	aiCmd.AddCommand(templateCmd)

	aiCmd.AddCommand(roadmapCmd)

}

GO_EOF



echo -e "${GREEN}-> 重新编译...${NC}"

make build



echo -e "${GREEN}=== AI 模块构建完成 ===${NC}"

echo -e "尝试以下命令开始你的 AI 之旅："

echo -e "1. ${CYAN}bin/cli ai roadmap${NC} (查看学习规划)"

echo -e "2. ${CYAN}bin/cli ai setup${NC}   (检查环境)"

echo -e "3. ${CYAN}bin/cli ai init demo${NC} (创建项目)"

echo -e "4. ${CYAN}cd demo && ../bin/cli ai template${NC} (生成训练代码)"

