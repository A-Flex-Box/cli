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
		
		if _, err := exec.LookPath("conda"); err == nil {
			cmd := exec.Command("conda", "env", "list")
			stdout, _ := cmd.StdoutPipe()
			cmd.Start()

			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
					continue
				}
				if strings.Contains(line, "*") {
					fmt.Printf("      👉 \033[1;32m%s\033[0m\n", line)
				} else {
					fmt.Printf("         %s\n", line)
				}
			}
			cmd.Wait()
		} else {
			fmt.Println("      (Conda not found, checking active VENV only)")
			if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
				envName := filepath.Base(venv)
				fmt.Printf("      👉 Active Venv: \033[1;32m%s\033[0m (%s)\n", envName, venv)
			} else {
				fmt.Println("      ⚠️  No Active Virtual Environment")
			}
		}

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

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(setupCmd)
	aiCmd.AddCommand(initCmd)
	// templateCmd has been removed
}
