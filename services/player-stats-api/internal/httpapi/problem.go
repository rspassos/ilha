package httpapi

import (
	"encoding/json"
	"net/http"
)

type Problem struct {
	Type          string         `json:"type,omitempty"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
}

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Value  string `json:"value,omitempty"`
}

type ProblemWriter interface {
	WriteProblem(w http.ResponseWriter, status int, problem Problem)
}

type JSONProblemWriter struct{}

func NewProblemWriter() JSONProblemWriter {
	return JSONProblemWriter{}
}

func (JSONProblemWriter) WriteProblem(w http.ResponseWriter, status int, problem Problem) {
	if problem.Status == 0 {
		problem.Status = status
	}
	if problem.Title == "" {
		problem.Title = http.StatusText(status)
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}
