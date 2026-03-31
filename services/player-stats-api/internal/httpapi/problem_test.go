package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONProblemWriterWriteProblem(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	NewProblemWriter().WriteProblem(recorder, http.StatusBadRequest, Problem{
		Type:   "https://ilha.dev/problems/invalid-query",
		Title:  "Invalid query parameters",
		Status: http.StatusBadRequest,
		Detail: "one or more query parameters are invalid",
		InvalidParams: []InvalidParam{
			{Name: "limit", Reason: "must be between 1 and 100", Value: "101"},
		},
	})

	if got := recorder.Code; got != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", got, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", got)
	}

	var problem Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if problem.Title != "Invalid query parameters" {
		t.Fatalf("Title = %q, want Invalid query parameters", problem.Title)
	}
	if len(problem.InvalidParams) != 1 {
		t.Fatalf("InvalidParams len = %d, want 1", len(problem.InvalidParams))
	}
	if problem.InvalidParams[0].Name != "limit" {
		t.Fatalf("InvalidParams[0].Name = %q, want limit", problem.InvalidParams[0].Name)
	}
}
