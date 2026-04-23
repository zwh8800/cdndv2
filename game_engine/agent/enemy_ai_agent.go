package agent

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/zwh8800/cdndv2/game_engine/llm"
	"github.com/zwh8800/cdndv2/game_engine/prompt"
)

// EnemyAIAgent 敌人/NPC AI Agent，负责生成敌方角色的行动意图和动作解析
// 这是一个轻量级Agent，不继承BaseSubAgent，由CombatSession直接调用
type EnemyAIAgent struct {
	llmClient llm.LLMClient
	logger    *zap.Logger
}

// NewEnemyAIAgent 创建敌人AI Agent
func NewEnemyAIAgent(llmClient llm.LLMClient, logger *zap.Logger) *EnemyAIAgent {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EnemyAIAgent{
		llmClient: llmClient,
		logger:    logger,
	}
}

// Name 返回Agent名称
func (a *EnemyAIAgent) Name() string {
	return EnemyAIAgentName
}

// SystemPrompt 加载并渲染敌人AI系统提示词
func (a *EnemyAIAgent) SystemPrompt(actorName, actorType string) string {
	rendered, err := prompt.LoadAndRender("enemy_ai_system.md", map[string]any{
		"ActorName": actorName,
		"ActorType": actorType,
	})
	if err != nil {
		a.logger.Error("Failed to load enemy AI system prompt", zap.Error(err))
		return fmt.Sprintf("你是%s，一个D&D战斗中的%s。用简短的话描述你的行动意图。", actorName, actorType)
	}
	return rendered
}

// GenerateIntent 使用LLM为敌人/NPC生成行动意图
// 返回自然语言描述的意图（如"我要冲向法师砍他"）
func (a *EnemyAIAgent) GenerateIntent(
	ctx context.Context,
	actorName, actorType string,
	battlefield string,
	actionsText string,
) (string, error) {
	systemPrompt := a.SystemPrompt(actorName, actorType)

	userPrompt := fmt.Sprintf(`当前战场:
%s

你可以做的事:
%s

用1-2句话描述你想做什么。`, battlefield, actionsText)

	resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			llm.NewSystemMessage(systemPrompt),
			llm.NewUserMessage(userPrompt),
		},
		Temperature: 0.7,
		MaxTokens:   150,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate enemy intent: %w", err)
	}

	a.logger.Info("[EnemyAI] LLM output",
		zap.String("actor", actorName),
		zap.String("actorType", actorType),
		zap.String("systemPrompt", systemPrompt),
		zap.String("userPrompt", userPrompt),
		zap.String("response", resp.Content),
	)

	return resp.Content, nil
}
