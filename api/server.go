package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/styx-oracle/styx/oracle"
	"github.com/styx-oracle/styx/types"
)

// Server provides HTTP API for STYX Oracle
type Server struct {
	oracle *oracle.Oracle
	mu     sync.RWMutex
}

// NewServer creates a new API server
func NewServer(selfID uint64) *Server {
	return &Server{
		oracle: oracle.New(types.NewNodeID(selfID)),
	}
}

// QueryResponse is the JSON response for queries
type QueryResponse struct {
	Target          uint64   `json:"target"`
	AliveConfidence float64  `json:"alive_confidence"`
	DeadConfidence  float64  `json:"dead_confidence"`
	Unknown         float64  `json:"unknown"`
	Refused         bool     `json:"refused"`
	RefusalReason   string   `json:"refusal_reason,omitempty"`
	Dead            bool     `json:"dead"`
	WitnessCount    int      `json:"witness_count"`
	Disagreement    float64  `json:"disagreement"`
	PartitionState  string   `json:"partition_state"`
	Evidence        []string `json:"evidence"`
}

// ReportRequest is the JSON request for reporting beliefs
type ReportRequest struct {
	Witness uint64  `json:"witness"`
	Target  uint64  `json:"target"`
	Alive   float64 `json:"alive"`
	Dead    float64 `json:"dead"`
	Unknown float64 `json:"unknown"`
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/query", s.handleQuery)
	mux.HandleFunc("/report", s.handleReport)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/witnesses", s.handleWitnesses)
	mux.HandleFunc("/metrics", s.handleMetrics)

	return mux
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write([]byte("# HELP styx_up STYX server is up\n"))
	w.Write([]byte("# TYPE styx_up gauge\n"))
	w.Write([]byte("styx_up 1\n"))
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetStr := r.URL.Query().Get("target")
	if targetStr == "" {
		http.Error(w, "missing target parameter", http.StatusBadRequest)
		return
	}

	targetID, err := strconv.ParseUint(targetStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid target id", http.StatusBadRequest)
		return
	}

	result := s.oracle.Query(types.NewNodeID(targetID))

	resp := QueryResponse{
		Target:          targetID,
		AliveConfidence: result.Belief.Alive().Value(),
		DeadConfidence:  result.Belief.Dead().Value(),
		Unknown:         result.Belief.Unknown().Value(),
		Refused:         result.Refused,
		RefusalReason:   result.RefusalReason,
		Dead:            result.Dead,
		WitnessCount:    result.WitnessCount,
		Disagreement:    result.Disagreement,
		PartitionState:  result.PartitionState.String(),
		Evidence:        result.Evidence,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	belief, err := types.NewBelief(req.Alive, req.Dead, req.Unknown)
	if err != nil {
		http.Error(w, "invalid belief: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.oracle.ReceiveReport(
		types.NewNodeID(req.Witness),
		types.NewNodeID(req.Target),
		belief,
	)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"styx"}`))
}

func (s *Server) handleWitnesses(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Register witness
		var req struct {
			ID uint64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		s.oracle.RegisterWitness(types.NewNodeID(req.ID))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"registered"}`))
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}
