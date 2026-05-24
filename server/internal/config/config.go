package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	LLM      LLMConfig      `yaml:"llm"`
}

type ServerConfig struct {
	Port       int      `yaml:"port"`
	CORSOrigins []string `yaml:"cors_origins"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type StorageConfig struct {
	Root string `yaml:"root"`
}

type LLMConfig struct {
	Default   string              `yaml:"default"`
	Providers []LLMProviderConfig `yaml:"providers"`
}

type LLMProviderConfig struct {
	Name     string   `yaml:"name"`
	Endpoint string   `yaml:"endpoint"`
	Models   []string `yaml:"models"`
	APIKey   string   `yaml:"api_key"`
	Model    string   `yaml:"-"` // backward compat: defaults to Models[0]
}

// DefaultModel returns the first model in Models, or empty string if none.
func (p *LLMProviderConfig) DefaultModel() string {
	if p.Model != "" {
		return p.Model
	}
	if len(p.Models) > 0 {
		return p.Models[0]
	}
	return ""
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DATABASE_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DATABASE_PORT"); v != "" {
		cfg.Database.Port = parseEnvInt(v)
	}
	if v := os.Getenv("DATABASE_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DATABASE_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DATABASE_NAME"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = parseEnvInt(v)
	}
	if v := os.Getenv("STORAGE_ROOT"); v != "" {
		cfg.Storage.Root = v
	}
	if v := os.Getenv("LLM_DEFAULT"); v != "" {
		cfg.LLM.Default = v
	}
	for i := range cfg.LLM.Providers {
		prefix := "LLM_" + cfg.LLM.Providers[i].Name + "_"
		if v := os.Getenv(prefix + "API_KEY"); v != "" {
			cfg.LLM.Providers[i].APIKey = v
		}
		if v := os.Getenv(prefix + "MODEL"); v != "" {
			cfg.LLM.Providers[i].Models = []string{v}
		}
	}
}

func parseEnvInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LLMConfigResponse is the API-facing LLM config with masked secrets.
type LLMConfigResponse struct {
	Default   string                `json:"default"`
	Providers []LLMProviderResponse `json:"providers"`
}

// LLMProviderResponse is a provider entry with API key masked for safe display.
type LLMProviderResponse struct {
	Name     string   `json:"name"`
	Endpoint string   `json:"endpoint"`
	Models   []string `json:"models"`
	APIKey   string   `json:"api_key"`
}

// ToLLMConfigResponse converts the internal LLM config to an API-safe version.
func (c *LLMConfig) ToResponse() LLMConfigResponse {
	providers := make([]LLMProviderResponse, len(c.Providers))
	for i, p := range c.Providers {
		models := p.Models
		if len(models) == 0 && p.Model != "" {
			models = []string{p.Model}
		}
		providers[i] = LLMProviderResponse{
			Name:     p.Name,
			Endpoint: p.Endpoint,
			Models:   models,
			APIKey:   MaskKey(p.APIKey),
		}
	}
	return LLMConfigResponse{
		Default:   c.Default,
		Providers: providers,
	}
}

func MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
