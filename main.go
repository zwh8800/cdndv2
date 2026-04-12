package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zwh8800/dnd-core/pkg/engine"

	gameengine "github.com/zwh8800/cdndv2/game_engine"
	"github.com/zwh8800/cdndv2/game_engine/agent"
	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/llm/openai"
)

func main() {
	fmt.Println("=== D&D LLM 游戏引擎 ===")
	fmt.Println()

	// 创建 Mock LLM 客户端（用于演示）
	mockClient := openai.NewMockClient([]*llm.CompletionResponse{
		{
			Content:      "欢迎来到这个古老的地牢！你站在一条昏暗的走廊入口，空气中弥漫着潮湿的气息。火把在墙壁上摇曳，投下跳跃的影子。你想要做什么？",
			FinishReason: llm.FinishReasonStop,
		},
	})

	// 创建游戏引擎（使用假密钥用于演示，实际需要真实 API Key）
	ge, err := gameengine.NewGameEngine(gameengine.EngineConfig{
		DNDEngineConfig: engine.DefaultConfig(),
		LLMConfig: openai.OpenAIConfig{
			Model:  "gpt-4o",
			APIKey: "sk-mock-key-for-demo",
		},
		MaxIterations: 10,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建游戏引擎失败: %v\n", err)
		os.Exit(1)
	}
	defer ge.Close()

	// 使用 Mock 客户端替换 OpenAI 客户端
	ge.SetLLMClient(mockClient)

	// 创建并设置主 Agent（使用 Mock 客户端）
	mainAgent := agent.NewMainAgent(ge.GetRegistry(), mockClient, nil)
	ge.SetMainAgent(mainAgent)

	fmt.Println("游戏引擎初始化成功！")
	fmt.Println("提示：当前使用 Mock LLM 客户端，实际使用需要配置 OpenAI API Key")
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
