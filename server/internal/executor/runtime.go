package executor

import (
	"encoding/json"
	"os/exec"
)

type RuntimeConfig struct {
	Type        string `json:"type"`
	Interpreter string `json:"interpreter"`
	Extension   string `json:"extension"`
	CompileCmd  string `json:"compile_cmd"`
	RunCmd      string `json:"run_cmd"`
	Image       string `json:"image"`
}

func DefaultImage(slug string) (string, bool) {
	return "", false
}

func DefaultRuntimeConfig(slug string) RuntimeConfig {
	return RuntimeConfig{
		Type:        "interpreted",
		Interpreter: slug,
		Extension:   ".txt",
		RunCmd:      "{interpreter} {file}",
	}
}

func (rc *RuntimeConfig) IsComplete() bool {
	if rc.Type == "unknown" || rc.Extension == "" || rc.RunCmd == "" {
		return false
	}
	if rc.Type == "interpreted" && rc.Interpreter == "" {
		return false
	}
	return true
}

func (rc *RuntimeConfig) FindInterpreter(candidates ...string) string {
	if rc.Interpreter != "" {
		if _, err := exec.LookPath(rc.Interpreter); err == nil {
			return rc.Interpreter
		}
	}
	for _, name := range candidates {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return rc.Interpreter
}

func ParseStoredRuntimeConfig(raw json.RawMessage, slug string) RuntimeConfig {
	rc, err := ParseRuntimeConfig(raw)
	if err != nil || rc.Image == "" {
		rc = DefaultRuntimeConfig(slug)
	}
	return rc
}

func (rc *RuntimeConfig) Marshal() []byte {
	b, _ := json.Marshal(rc)
	return b
}

func ParseRuntimeConfig(raw []byte) (RuntimeConfig, error) {
	var rc RuntimeConfig
	if len(raw) == 0 {
		return RuntimeConfig{}, nil
	}
	if err := json.Unmarshal(raw, &rc); err != nil {
		return RuntimeConfig{}, err
	}
	return rc, nil
}
