package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
server:
  port: 9090
  cors_origins:
    - "http://localhost:3000"
database:
  host: dbhost
  port: 1234
  user: testuser
  password: testpass
  dbname: testdb
  sslmode: require
storage:
  root: /data/learncode
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "dbhost" {
		t.Errorf("expected host dbhost, got %s", cfg.Database.Host)
	}
	if cfg.Storage.Root != "/data/learncode" {
		t.Errorf("expected storage root /data/learncode, got %s", cfg.Storage.Root)
	}
}

func TestStorageRootDefault(t *testing.T) {
	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: u
  password: p
  dbname: d
  sslmode: disable
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Storage.Root != "" {
		t.Errorf("expected empty storage root (no default applied by Load), got %s", cfg.Storage.Root)
	}
}

func TestStorageRootEnvOverride(t *testing.T) {
	os.Setenv("STORAGE_ROOT", "/custom/storage")
	defer os.Unsetenv("STORAGE_ROOT")

	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: u
  password: p
  dbname: d
  sslmode: disable
storage:
  root: ./default-storage
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Storage.Root != "/custom/storage" {
		t.Errorf("expected env override /custom/storage, got %s", cfg.Storage.Root)
	}
}

func TestLLMConfig(t *testing.T) {
	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: u
  password: p
  dbname: d
  sslmode: disable
llm:
  default: ollama
  providers:
    - name: deepseek
      endpoint: https://api.deepseek.com/v1
      models:
        - deepseek-chat
      api_key: ""
    - name: ollama
      endpoint: http://localhost:11434/v1
      models:
        - qwen3
      api_key: ""
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.LLM.Default != "ollama" {
		t.Errorf("expected default ollama, got %s", cfg.LLM.Default)
	}
	if len(cfg.LLM.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(cfg.LLM.Providers))
	}
	if cfg.LLM.Providers[0].Name != "deepseek" {
		t.Errorf("expected first provider deepseek, got %s", cfg.LLM.Providers[0].Name)
	}
}

func TestLLMEnvOverride(t *testing.T) {
	os.Setenv("LLM_DEFAULT", "deepseek")
	os.Setenv("LLM_deepseek_API_KEY", "sk-test-key")
	os.Setenv("LLM_deepseek_MODEL", "deepseek-v3")
	defer func() {
		os.Unsetenv("LLM_DEFAULT")
		os.Unsetenv("LLM_deepseek_API_KEY")
		os.Unsetenv("LLM_deepseek_MODEL")
	}()

	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := `
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: u
  password: p
  dbname: d
  sslmode: disable
llm:
  default: ollama
  providers:
    - name: deepseek
      endpoint: https://api.deepseek.com/v1
      model: deepseek-chat
      api_key: ""
`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.LLM.Default != "deepseek" {
		t.Errorf("expected env override default=deepseek, got %s", cfg.LLM.Default)
	}
	if cfg.LLM.Providers[0].APIKey != "sk-test-key" {
		t.Errorf("expected env override api_key, got %s", cfg.LLM.Providers[0].APIKey)
	}
	if len(cfg.LLM.Providers[0].Models) == 0 || cfg.LLM.Providers[0].Models[0] != "deepseek-v3" {
		t.Errorf("expected env override model deepseek-v3, got %v", cfg.LLM.Providers[0].Models)
	}
}
