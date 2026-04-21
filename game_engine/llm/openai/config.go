package openai

// OpenAIConfig OpenAI客户端配置
type OpenAIConfig struct {
	// APIKey OpenAI API密钥
	APIKey string

	// Model 模型名称，默认 gpt-4o
	Model string

	// BaseURL 自定义API端点（用于代理或兼容API）
	BaseURL string

	// Temperature 温度参数，默认1.0
	Temperature float64

	// MaxTokens 最大token数，默认4096
	MaxTokens int

	// ContextWindowSize 模型上下文窗口大小（token数），默认128000（gpt-4o）
	// 用于上下文压缩判断
	ContextWindowSize int
}

// DefaultOpenAIConfig 返回默认配置
func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		Model:             "gpt-4o",
		Temperature:       1.0,
		MaxTokens:         4096,
		ContextWindowSize: 128000,
	}
}

// Validate 验证配置
func (c *OpenAIConfig) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}
	if c.Model == "" {
		c.Model = "gpt-4o"
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		c.Temperature = 1.0
	}
	if c.MaxTokens <= 0 {
		c.MaxTokens = 4096
	}
	if c.ContextWindowSize <= 0 {
		c.ContextWindowSize = 128000
	}
	return nil
}
