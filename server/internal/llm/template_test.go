package llm

import (
	"os"
	"testing"
)

func TestLoadTemplate(t *testing.T) {
	tmp, err := os.CreateTemp("", "template-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
system_prompt: |
  You are {{.Role}}.
user_prompt: |
  Analyze: {{.Topic}}
temperature: 0.5
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	tmpl, err := LoadTemplate(tmp.Name(), map[string]string{
		"Role":  "an expert",
		"Topic": "concurrency",
	})
	if err != nil {
		t.Fatalf("LoadTemplate() error: %v", err)
	}

	if tmpl.SystemPrompt != "You are an expert.\n" {
		t.Errorf("expected 'You are an expert.', got %q", tmpl.SystemPrompt)
	}
	if tmpl.UserPrompt != "Analyze: concurrency\n" {
		t.Errorf("expected 'Analyze: concurrency', got %q", tmpl.UserPrompt)
	}
	if tmpl.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", tmpl.Temperature)
	}
	if tmpl.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", tmpl.MaxTokens)
	}
}

func TestLoadTemplateDefaults(t *testing.T) {
	tmp, err := os.CreateTemp("", "template-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
system_prompt: "Hello"
user_prompt: "World"
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	tmpl, err := LoadTemplate(tmp.Name(), nil)
	if err != nil {
		t.Fatalf("LoadTemplate() error: %v", err)
	}

	if tmpl.Temperature != 0.3 {
		t.Errorf("expected default temperature 0.3, got %f", tmpl.Temperature)
	}
	if tmpl.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", tmpl.MaxTokens)
	}
}
