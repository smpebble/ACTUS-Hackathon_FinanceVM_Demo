// Package api provides the HTTP API server for the ACTUS-FVM web demo.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/smpebble/actus-fvm/internal/demo"
	"github.com/smpebble/actus-fvm/internal/model"
)

// RunRequest is the body for POST /api/run.
type RunRequest struct {
	Scenario string                 `json:"scenario"`
	Params   map[string]interface{} `json:"params"`
}

// RunResponse is returned by POST /api/run.
type RunResponse struct {
	Results     []*model.ScenarioResult `json:"results"`
	Duration    string                  `json:"duration"`
	TotalEvents int                     `json:"totalEvents"`
	Success     bool                    `json:"success"`
	Error       string                  `json:"error,omitempty"`
}

// ScenarioMeta describes a scenario for GET /api/scenarios.
type ScenarioMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Explanation string `json:"explanation"`
	Contract    string `json:"contract"`
}

// Server is the HTTP API server.
type Server struct {
	runner  *demo.Runner
	mux     *http.ServeMux
	baseDir string
}

// NewServer creates a new API server rooted at baseDir.
func NewServer(baseDir string) *Server {
	s := &Server{
		runner:  demo.NewRunner(baseDir),
		mux:     http.NewServeMux(),
		baseDir: baseDir,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/scenarios", s.handleScenarios)
	s.mux.HandleFunc("/api/run", s.handleRun)
	// Serve SPA from web/
	webDir := filepath.Join(s.baseDir, "web")
	s.mux.Handle("/", http.FileServer(http.Dir(webDir)))
}

// ServeHTTP implements http.Handler with CORS pre-flight support.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleScenarios(w http.ResponseWriter, _ *http.Request) {
	metas := []ScenarioMeta{
		{
			ID:          "stablecoin",
			Name:        "Stablecoin (TWDX)",
			Description: "1:1 TWD-backed stablecoin. Mint 1,000,000 TWDX with zero precision loss.",
			Explanation: "The PAM (Principal At Maturity) contract is used because a stablecoin fundamentally represents a zero-interest bearing note where the principal amount is rigidly maintained, mirroring a PAM's principal redemption behavior.",
			Contract:    "PAM",
		},
		{
			ID:          "bond",
			Name:        "Corporate Bond (5Y)",
			Description: "5-year bond, 100M TWD, 3.5% coupon, semi-annual, Act/360.",
			Explanation: "A PAM contract accurately models a standard corporate bond where interest is paid periodically, but the entire principal component is returned exclusively at the maturity date.",
			Contract:    "PAM",
		},
		{
			ID:          "loan",
			Name:        "SME Loan (ANN)",
			Description: "10M TWD, 5-year, 4.5% fixed, monthly amortization (60 payments).",
			Explanation: "The ANN (Annuity) contract perfectly models an SME loan because it natively handles uniform periodic payments consisting of progressively shifting interest and principal amortization.",
			Contract:    "ANN",
		},
		{
			ID:          "derivative",
			Name:        "Interest Rate Swap",
			Description: "3-year IRS, 50M TWD, fixed 3.0% vs floating TAIBOR+50bps, quarterly.",
			Explanation: "The SWAPS contract provides the precise multi-leg structure required to manage the exchange of fixed rate cashflows versus floating rate cashflows tied to an external market index like TAIBOR.",
			Contract:    "SWAPS",
		},
	}
	writeJSON(w, http.StatusOK, metas)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, RunResponse{Error: "invalid request body"})
		return
	}

	start := time.Now()
	var results []*model.ScenarioResult
	var runErr error

	switch strings.ToLower(strings.TrimSpace(req.Scenario)) {
	case "all", "":
		results, runErr = s.runner.RunAllAndCollect(req.Params)
	case "stablecoin":
		results, runErr = runOne(s.runner, model.ScenarioStablecoin, req.Params)
	case "bond":
		results, runErr = runOne(s.runner, model.ScenarioBond, req.Params)
	case "loan":
		results, runErr = runOne(s.runner, model.ScenarioLoan, req.Params)
	case "derivative":
		results, runErr = runOne(s.runner, model.ScenarioDerivative, req.Params)
	default:
		writeJSON(w, http.StatusBadRequest, RunResponse{
			Error: fmt.Sprintf("unknown scenario: %q", req.Scenario),
		})
		return
	}

	totalEvents := 0
	for _, res := range results {
		totalEvents += len(res.CashFlowEvents)
	}

	resp := RunResponse{
		Results:     results,
		Duration:    time.Since(start).Round(time.Millisecond).String(),
		TotalEvents: totalEvents,
		Success:     runErr == nil,
	}
	if runErr != nil {
		resp.Error = runErr.Error()
	}

	status := http.StatusOK
	if runErr != nil {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, resp)
}

func runOne(runner *demo.Runner, t model.ScenarioType, params map[string]interface{}) ([]*model.ScenarioResult, error) {
	res, err := runner.RunScenario(t, params)
	if res != nil {
		return []*model.ScenarioResult{res}, err
	}
	return nil, err
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}
