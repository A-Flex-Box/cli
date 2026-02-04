package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
		log := logger.NewLogger()
		defer log.Sync()

		log.Info("AI环境诊断开始")
		fmt.Println("🔌 AI Environment Diagnostic")
		fmt.Println("--------------------------------------------------")

		// Step 1: NVIDIA Driver
		fmt.Print("[1/3] NVIDIA Driver: ")
		if cmdPath, err := exec.LookPath("nvidia-smi"); err == nil {
			log.Info("检测到NVIDIA驱动", zap.String("path", cmdPath))
			fmt.Printf("✅ Detected (%s)\n", cmdPath)
			out, _ := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader").Output()
			gpuInfo := strings.TrimSpace(string(out))
			fmt.Printf("      GPU: %s\n", gpuInfo)
			log.Info("GPU信息", zap.String("info", gpuInfo))
		} else {
			log.Warn("未找到NVIDIA驱动，运行在CPU模式")
			fmt.Println("❌ Not Found (Running on CPU mode)")
		}

		// Step 2: PyTorch CUDA
		fmt.Print("[2/3] PyTorch Stack: ")
		checkCuda := exec.Command("python3", "-c", "import torch; print(f'{torch.__version__}|{torch.cuda.is_available()}|{torch.version.cuda}')")
		if out, err := checkCuda.CombinedOutput(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), "|")
			if len(parts) == 3 {
				ver, avail, cudaVer := parts[0], parts[1], parts[2]
				log.Info("PyTorch信息", zap.String("version", ver), zap.String("cuda_available", avail), zap.String("cuda_version", cudaVer))
				if avail == "True" {
					fmt.Printf("✅ Ready (Torch v%s + CUDA v%s)\n", ver, cudaVer)
				} else {
					fmt.Printf("⚠️  Torch v%s Installed (No CUDA)\n", ver)
				}
			}
		} else {
			log.Error("Python/PyTorch检查失败", zap.Error(err))
			fmt.Println("❌ Python/PyTorch not working.")
		}

		// Step 3: Virtual Environments (List All)
		fmt.Println("[3/3] Virtual Environments:")
		
		if _, err := exec.LookPath("conda"); err == nil {
			log.Info("检测到conda，列出虚拟环境")
			cmd := exec.Command("conda", "env", "list")
			stdout, _ := cmd.StdoutPipe()
			cmd.Start()

			scanner := bufio.NewScanner(stdout)
			envCount := 0
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
					continue
				}
				envCount++
				if strings.Contains(line, "*") {
					fmt.Printf("      👉 \033[1;32m%s\033[0m\n", line)
					log.Info("当前激活的虚拟环境", zap.String("line", line))
				} else {
					fmt.Printf("         %s\n", line)
				}
			}
			cmd.Wait()
			log.Info("虚拟环境列表", zap.Int("count", envCount))
		} else {
			log.Info("未找到conda，检查活动VENV")
			fmt.Println("      (Conda not found, checking active VENV only)")
			if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
				envName := filepath.Base(venv)
				fmt.Printf("      👉 Active Venv: \033[1;32m%s\033[0m (%s)\n", envName, venv)
				log.Info("活动虚拟环境", zap.String("name", envName), zap.String("path", venv))
			} else {
				log.Warn("未找到活动虚拟环境")
				fmt.Println("      ⚠️  No Active Virtual Environment")
			}
		}

		checkPath := exec.Command("python3", "-c", "import sys; print(sys.executable)")
		if out, err := checkPath.CombinedOutput(); err == nil {
			realPath := strings.TrimSpace(string(out))
			fmt.Printf("\n      Interpreter: %s\n", realPath)
			log.Info("Python解释器路径", zap.String("path", realPath))
		}

		fmt.Println("--------------------------------------------------")
		log.Info("AI环境诊断完成")
	},
}

// 2. Init: 标准化目录结构
var initCmd = &cobra.Command{
	Use:   "init [project_name]",
	Short: "生成 AI 项目标准目录结构",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		defer log.Sync()

		pName := args[0]
		log.Info("初始化AI项目", zap.String("project_name", pName))
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
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				log.Error("创建目录失败", zap.String("path", fullPath), zap.Error(err))
				continue
			}
			readmePath := filepath.Join(fullPath, "README.md")
			if err := os.WriteFile(readmePath, []byte(desc), 0644); err != nil {
				log.Error("创建README失败", zap.String("path", readmePath), zap.Error(err))
			} else {
				log.Info("创建目录和README", zap.String("path", fullPath))
			}
		}
		fmt.Println("✅ Done.")
		log.Info("项目初始化完成", zap.String("project_name", pName))
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(setupCmd)
	aiCmd.AddCommand(initCmd)
	// templateCmd has been removed
}
