package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zwh8800/dnd-core/pkg/engine"

	gameengine "github.com/zwh8800/cdndv2/game_engine"
	"github.com/zwh8800/cdndv2/game_engine/llm/openai"
)

func main() {
	fmt.Println("=== D&D LLM 游戏引擎 ===")
	fmt.Println()

	// 从环境变量读取配置
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("警告: 未设置 OPENAI_API_KEY 环境变量")
		fmt.Println("请运行: export OPENAI_API_KEY=sk-your-key-here")
		fmt.Println()
		fmt.Println("按回车键继续使用占位符密钥（将会失败），或 Ctrl+C 退出")
		bufio.NewScanner(os.Stdin).Scan()
		apiKey = "sk-placeholder-key"
	}

	// 读取模型配置（默认 gpt-4o）
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	// 读取 Base URL（可选，用于代理或兼容 API）
	baseURL := os.Getenv("OPENAI_BASE_URL")

	// 创建游戏引擎
	llmConfig := openai.OpenAIConfig{
		Model:       model,
		APIKey:      apiKey,
		Temperature: 0.8,
		MaxTokens:   2048,
	}
	if baseURL != "" {
		llmConfig.BaseURL = baseURL
	}

	ge, err := gameengine.NewGameEngine(gameengine.EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig:       llmConfig,
		MaxIterations:   10,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建游戏引擎失败: %v\n", err)
		os.Exit(1)
	}
	defer ge.Close()

	fmt.Println("游戏引擎初始化成功！")
	client := ge.GetLLMClient().(*openai.OpenAIClient)
	config := client.GetConfig()
	fmt.Printf("使用模型: %s\n", config.Model)
	if config.BaseURL != "" {
		fmt.Printf("API 端点: %s\n", config.BaseURL)
	}
	fmt.Println()

	// 创建新游戏
	ctx := context.Background()
	session, err := ge.NewGame(ctx, "第一次冒险", "一个古老的地牢探险")
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建游戏失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("游戏已创建: %s\n", session.ID)
	fmt.Println()

	// 启动交互式循环
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== 游戏开始 ===")
	fmt.Println("输入你的行动，输入 'quit' 退出游戏")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.ToLower(input) == "quit" {
			fmt.Println("感谢游玩！再见！")
			break
		}

		// 处理玩家输入
		response, err := ge.ProcessInput(ctx, session, input)
		if err != nil {
			fmt.Printf("处理输入时发生错误: %v\n", err)
			continue
		}

		// 打印响应
		if response != "" {
			fmt.Println(response)
		}

		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "读取输入失败: %v\n", err)
	}
}
