package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ProgressStep struct {
	Step    string `json:"step"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ProgressWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

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

func (p *ProgressWriter) Emit(step ProgressStep) {
	b, _ := json.Marshal(step)
	p.w.Write(b)
	p.w.Write([]byte("\n"))
	p.flusher.Flush()
}

func (p *ProgressWriter) EmitRunning(step, message string) {
	p.Emit(ProgressStep{Step: step, Status: "running", Message: message})
}

func (p *ProgressWriter) EmitDone(step, message string) {
	p.Emit(ProgressStep{Step: step, Status: "done", Message: message})
}

func (p *ProgressWriter) EmitError(step, message string) {
	p.Emit(ProgressStep{Step: step, Status: "error", Message: message})
}
