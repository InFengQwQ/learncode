package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProgressStep represents one step in a multi-stage query pipeline.
type ProgressStep struct {
	Step    string `json:"step"`
	Status  string `json:"status"` // "running" | "done" | "error"
	Message string `json:"message"`
}

// ProgressWriter streams progress events to the HTTP response via
// newline-delimited JSON (NDJSON). Each line is a self-contained JSON object
// that the frontend can parse and render incrementally.
type ProgressWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewProgressWriter prepares the HTTP response for streaming.
func NewProgressWriter(w http.ResponseWriter) (*ProgressWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &ProgressWriter{w: w, flusher: flusher}, nil
}

// Emit writes a progress step and flushes it to the client.
func (p *ProgressWriter) Emit(step ProgressStep) {
	b, _ := json.Marshal(step)
	p.w.Write(b)
	p.w.Write([]byte("\n"))
	p.flusher.Flush()
}

// EmitRunning sends a "running" event for a step.
func (p *ProgressWriter) EmitRunning(step, message string) {
	p.Emit(ProgressStep{Step: step, Status: "running", Message: message})
}

// EmitDone sends a "done" event for a step.
func (p *ProgressWriter) EmitDone(step, message string) {
	p.Emit(ProgressStep{Step: step, Status: "done", Message: message})
}

// EmitError sends an "error" event for a step.
func (p *ProgressWriter) EmitError(step, message string) {
	p.Emit(ProgressStep{Step: step, Status: "error", Message: message})
}
