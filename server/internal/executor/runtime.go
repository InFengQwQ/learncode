package executor

import (
	"encoding/json"
	"os/exec"
)

// RuntimeConfig defines how to execute code for a given language version.
type RuntimeConfig struct {
	Type        string `json:"type"`        // "interpreted" or "compiled"
	Interpreter string `json:"interpreter"` // binary name e.g. "python", "node"
	Extension   string `json:"extension"`   // source file extension e.g. ".py", ".js"
	CompileCmd  string `json:"compile_cmd"` // template for compilation (compiled only)
	RunCmd      string `json:"run_cmd"`     // template for running. {file} is the source file
	Image       string `json:"image"`       // Docker image e.g. "python:3.12-slim"
}

// DefaultImage always returns false — no default image is known.
// Image references come from DiscoveredVersion (image_tag, docker_refs),
// not from a lookup table.
func DefaultImage(slug string) (string, bool) {
	return "", false
}

// DefaultRuntimeConfig returns a minimal runtime config for a language slug.
// Interpreter defaults to the slug (Docker image name ≈ binary name).
// Type defaults to "interpreted" — most languages are interpreted or have
// an interactive mode. The config is intentionally minimal; real configuration
// happens during init (Docker pull) or manual setup.
func DefaultRuntimeConfig(slug string) RuntimeConfig {
	return RuntimeConfig{
		Type:        "interpreted",
		Interpreter: slug,
		Extension:   ".txt",
		RunCmd:      "{interpreter} {file}",
	}
}

// IsComplete returns true if the config has enough information to execute code.
func (rc *RuntimeConfig) IsComplete() bool {
	if rc.Type == "unknown" || rc.Extension == "" || rc.RunCmd == "" {
		return false
	}
	if rc.Type == "interpreted" && rc.Interpreter == "" {
		return false
	}
	return true
}

// FindInterpreter resolves the interpreter binary, trying multiple candidates.
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

// ParseStoredRuntimeConfig reads a stored runtime_config and merges it with
// sensible defaults for the given slug. If the stored config has an Image, it's
// ready for Docker execution; otherwise, the default is returned for host fallback.
func ParseStoredRuntimeConfig(raw json.RawMessage, slug string) RuntimeConfig {
	rc, err := ParseRuntimeConfig(raw)
	if err != nil || rc.Image == "" {
		rc = DefaultRuntimeConfig(slug)
	}
	return rc
}

// Marshal returns the JSON encoding of the config.
func (rc *RuntimeConfig) Marshal() []byte {
	b, _ := json.Marshal(rc)
	return b
}

// ParseRuntimeConfig decodes a stored runtime_config JSONB value.
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
