package llm

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

type PromptTemplate struct {
	SystemPrompt string  `yaml:"system_prompt"`
	UserPrompt   string  `yaml:"user_prompt"`
	Temperature  float64 `yaml:"temperature"`
	MaxTokens    int     `yaml:"max_tokens"`
}

func LoadTemplate(path string, vars map[string]string) (*PromptTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}

	var tmpl PromptTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse template yaml: %w", err)
	}

	tmpl.SystemPrompt, err = renderTemplate(tmpl.SystemPrompt, vars)
	if err != nil {
		return nil, fmt.Errorf("system_prompt: %w", err)
	}
	tmpl.UserPrompt, err = renderTemplate(tmpl.UserPrompt, vars)
	if err != nil {
		return nil, fmt.Errorf("user_prompt: %w", err)
	}

	if tmpl.Temperature == 0 {
		tmpl.Temperature = 0.3
	}
	if tmpl.MaxTokens == 0 {
		tmpl.MaxTokens = 4096
	}

	return &tmpl, nil
}

func renderTemplate(s string, vars map[string]string) (string, error) {
	t, err := template.New("").Parse(s)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
