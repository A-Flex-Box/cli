package ai

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

// Setup runs AI environment diagnostic (GPU, CUDA, virtual envs).
func Setup(log *zap.Logger) {
	if log == nil {
		log = logger.NewLogger()
		defer log.Sync()
	}
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
}
