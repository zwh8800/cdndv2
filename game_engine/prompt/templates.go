package prompt

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed *.md
var promptFS embed.FS

// LoadSystemPrompt 从嵌入的文件系统加载系统提示词
func LoadSystemPrompt(name string) (string, error) {
	content, err := promptFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("failed to load prompt %s: %w", name, err)
	}
	return string(content), nil
}

// RenderTemplate 渲染提示词模板
func RenderTemplate(templateStr string, data map[string]any) (string, error) {
	tmpl, err := template.New("prompt").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// LoadAndRender 加载并渲染提示词模板
func LoadAndRender(name string, data map[string]any) (string, error) {
	templateStr, err := LoadSystemPrompt(name)
	if err != nil {
		return "", err
	}
	return RenderTemplate(templateStr, data)
}
