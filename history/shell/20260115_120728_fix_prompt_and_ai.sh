#!/bin/bash
# METADATA_START
# timestamp: 2026-01-05 13:10:00
# original_prompt: 去掉ai的训练闭环代码,而且现在好像prompt输出的history step没有项目结构请你修复
# summary: 移除 AI 训练模板代码并修复 Prompt 中项目结构丢失的问题
# action: 修改 cmd/ai.go 删除 template 子命令；重写 cmd/prompt.go，在构建 Context 时重新加入项目结构 (project_structure) 的读取与展示逻辑。
# expected_outcome: cli ai template 命令不再存在；cli prompt 输出将包含 ## Current Project Structure 区域。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> [1/2] 正在修复 cmd/prompt.go (找回丢失的项目结构)...${NC}"

cat << 'GO_EOF' > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// 数据结构
type promptHistoryItem struct {
	Timestamp       string            `json:"timestamp"`
	OriginalPrompt  string            `json:"original_prompt"`
	Summary         string            `json:"summary"`
	Action          string            `json:"action"`
	ExpectedOutcome string            `json:"expected_outcome"`
	Context         map[string]string `json:"context,omitempty"`
}

var outputFormat string

// -----------------------------------------------------------
// 辅助函数: 生成基础 Context Prompt
// -----------------------------------------------------------
func buildContextPrompt() string {
	historyPath := "history/history.json"
	var items []promptHistoryItem
	
	// 读取历史记录
	if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
		json.Unmarshal(data, &items)
	}

	var sb strings.Builder
	sb.WriteString("# Project Context (History)\n")
	
	// 1. 提取最新的项目结构快照 (遍历所有记录查找最新值)
	var lastStructure string
	for _, item := range items {
		if val, ok := item.Context["project_structure"]; ok && val != "" {
			lastStructure = val
		}
	}

	// 2. 生成历史操作摘要 (只取最近 3 条，避免上下文过长)
	sb.WriteString("Recent development steps for context:\n\n")
	startIdx := 0
	if len(items) > 3 {
		startIdx = len(items) - 3
	}

	for i := startIdx; i < len(items); i++ {
		item := items[i]
		sb.WriteString(fmt.Sprintf("## History Step %d (%s)\n", i+1, item.Timestamp))
		
		// 简单的文本截断处理
		shortPrompt := strings.ReplaceAll(item.OriginalPrompt, "\n", " ")
		if len(shortPrompt) > 80 {
			shortPrompt = shortPrompt[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("- Request: %s\n", shortPrompt)) 
		sb.WriteString(fmt.Sprintf("- Action: %s\n\n", item.Action))
	}
	
	// 3. 如果找到了项目结构，追加到 Context 末尾 (这就是修复点)
	if lastStructure != "" {
		sb.WriteString("## Current Project Structure\n")
		sb.WriteString("```text\n")
		sb.WriteString(lastStructure)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("--------------------------------------------------\n")
	return sb.String()
}

// -----------------------------------------------------------
// 辅助函数: 附加输出格式约束
// -----------------------------------------------------------
func appendOutputConstraints(sb *strings.Builder, requirement string) {
	if outputFormat != "" && outputFormat != "text" {
		sb.WriteString(fmt.Sprintf("## Output Format Constraints\n"))
		sb.WriteString(fmt.Sprintf("1. Provide the solution as a **single %s file**.\n", outputFormat))
		sb.WriteString("2. **CRITICAL: METADATA HEADER REQUIRED**\n")
		sb.WriteString("   The file MUST start with a metadata header block in comments.\n")
		sb.WriteString("   The `original_prompt` field MUST contain the **EXACT FULL TEXT** of the 'New Requirement' section below. **DO NOT TRUNCATE.**\n")
		
		commentChar := "#"
		if outputFormat == "go" || outputFormat == "cpp" {
			commentChar = "//"
		}
		
		sb.WriteString(fmt.Sprintf("   Format: %s METADATA_START ... %s METADATA_END\n\n", commentChar, commentChar))
	}
}

// -----------------------------------------------------------
// 主命令: prompt
// -----------------------------------------------------------
var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成 AI 提示词 (任务模式)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var sb strings.Builder
		sb.WriteString(buildContextPrompt())
		
		sb.WriteString("# New Requirement (Current Task)\n")
		userRequirement := strings.Join(args, " ")
		sb.WriteString(userRequirement)
		sb.WriteString("\n\n")

		appendOutputConstraints(&sb, userRequirement)
		fmt.Println(sb.String())
	},
}

// -----------------------------------------------------------
// 子命令: prompt commit
// -----------------------------------------------------------
var promptCommitCmd = &cobra.Command{
	Use:   "commit [optional_instruction]",
	Short: "根据当前 Git 变更生成 Commit Message 提示词",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		diffCmd := exec.Command("git", "diff", "HEAD")
		diffOut, err := diffCmd.CombinedOutput()
		diffStr := string(diffOut)

		if err != nil {
			fmt.Printf("Error running git diff: %v\n", err)
			return
		}
		if len(strings.TrimSpace(diffStr)) == 0 {
			fmt.Println("❌ No changes detected (git diff is empty). Nothing to commit.")
			return
		}

		var sb strings.Builder
		
		sb.WriteString("# Task: Generate Git Commit Message\n")
		sb.WriteString("You are a Senior Developer. Please write a semantic git commit message for the following code changes.\n\n")
		
		sb.WriteString("## Code Changes (Git Diff)\n")
		sb.WriteString("```diff\n")
		if len(diffStr) > 8000 {
			sb.WriteString(diffStr[:8000] + "\n... (diff truncated) ...")
		} else {
			sb.WriteString(diffStr)
		}
		sb.WriteString("\n```\n\n")

		sb.WriteString(buildContextPrompt())

		sb.WriteString("## Instruction\n")
		instruction := "Analyze the diff above. Generate a concise and meaningful commit message following **Conventional Commits** format."
		if len(args) > 0 {
			instruction = strings.Join(args, " ")
		}
		sb.WriteString(instruction)
		sb.WriteString("\n\n")
		
		sb.WriteString("## Expected Output Format\n")
		sb.WriteString("```text\n")
		sb.WriteString("<type>(<scope>): <subject>\n")
		sb.WriteString("\n")
		sb.WriteString("<body>\n")
		sb.WriteString("\n")
		sb.WriteString("[Optional Footer: Ref #IssueID]\n")
		sb.WriteString("```\n")

		fmt.Println(sb.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(promptCommitCmd)
	promptCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format")
}
GO_EOF


echo -e "${GREEN}-> [2/2] 正在修复 cmd/ai.go (移除 template 闭环代码)...${NC}"

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
GO_EOF

echo -e "${GREEN}-> 重新编译...${NC}"
make build

echo -e "${GREEN}=== 修复完成 ===${NC}"
echo -e "现在运行 ${GREEN}bin/cli prompt \"查看结构\"${NC} 将重新看到文件树。"
