package service

import (
	"testing"

	"learncode/internal/llm"
)

func TestParseLLMJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  bool
	}{
		{
			name:     "plain json",
			input:    `{"is_language":true,"official_name":"Python","description":"A popular language","confidence":9}`,
			wantName: "Python",
		},
		{
			name:     "markdown fenced json",
			input:    "```json\n{\"is_language\":true,\"official_name\":\"Go\",\"description\":\"Systems language\",\"confidence\":10}\n```",
			wantName: "Go",
		},
		{
			name:     "markdown fence without language",
			input:    "```\n{\"is_language\":true,\"official_name\":\"Rust\",\"description\":\"Safe systems language\",\"confidence\":8}\n```",
			wantName: "Rust",
		},
		{
			name:     "text before json",
			input:    "Here is the result:\n{\"is_language\":true,\"official_name\":\"C++\",\"description\":\"Low-level language\",\"confidence\":7}",
			wantName: "C++",
		},
		{
			name:    "invalid json",
			input:   "not json at all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result analyzeResult
			err := llm.ParseLLMJSON(tt.input, &result)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.OfficialName != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.OfficialName)
			}
		})
	}
}
