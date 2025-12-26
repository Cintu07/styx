package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	server := NewServer(1)
	handler := server.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestQueryEndpoint(t *testing.T) {
	server := NewServer(1)
	handler := server.Handler()

	req := httptest.NewRequest("GET", "/query?target=99", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp QueryResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Target != 99 {
		t.Errorf("expected target 99, got %d", resp.Target)
	}

	// No evidence should give unknown
	total := resp.AliveConfidence + resp.DeadConfidence + resp.Unknown
	if total < 0.99 || total > 1.01 {
		t.Errorf("beliefs should sum to 1, got %f", total)
	}
}

func TestReportEndpoint(t *testing.T) {
	server := NewServer(1)
	handler := server.Handler()

	report := ReportRequest{
		Witness: 10,
		Target:  99,
		Alive:   0.8,
		Dead:    0.1,
		Unknown: 0.1,
	}

	body, _ := json.Marshal(report)
	req := httptest.NewRequest("POST", "/report", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	// Now query should have data
	req2 := httptest.NewRequest("GET", "/query?target=99", nil)
	w2 := httptest.NewRecorder()

	handler.ServeHTTP(w2, req2)

	var resp QueryResponse
	json.NewDecoder(w2.Body).Decode(&resp)

	if resp.WitnessCount != 1 {
		t.Errorf("expected 1 witness, got %d", resp.WitnessCount)
	}
}

func TestInvalidBelief(t *testing.T) {
	server := NewServer(1)
	handler := server.Handler()

	// Invalid: doesnt sum to 1
	report := ReportRequest{
		Witness: 10,
		Target:  99,
		Alive:   0.5,
		Dead:    0.5,
		Unknown: 0.5,
	}

	body, _ := json.Marshal(report)
	req := httptest.NewRequest("POST", "/report", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid belief, got %d", w.Code)
	}
}
