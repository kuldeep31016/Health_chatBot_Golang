package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"health-assistant/backend/agent"
	"health-assistant/backend/jobs"
)

type ChatHandler struct {
	agent          *agent.Agent
	worker         *jobs.WorkerPool
	sessionHistory map[string][]agent.ChatMessage
	jobStore       map[string]ChatJob
	mu             sync.RWMutex
}

func NewChatHandler(a *agent.Agent, worker *jobs.WorkerPool) *ChatHandler {
	return &ChatHandler{
		agent:          a,
		worker:         worker,
		sessionHistory: make(map[string][]agent.ChatMessage),
		jobStore:       make(map[string]ChatJob),
	}
}

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type ChatResponse struct {
	JobID    string `json:"job_id,omitempty"`
	Response string `json:"response"`
	Status   string `json:"status"`
}

type ChatJob struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Response  string `json:"response,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.enableCORS(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == http.MethodGet {
		h.getJobStatus(w, r)
		return
	}

	if r.Method != http.MethodPost {
		h.writeJSON(w, http.StatusMethodNotAllowed, ChatResponse{Response: "method not allowed", Status: "fail"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		h.writeJSON(w, http.StatusBadRequest, ChatResponse{Response: "invalid request", Status: "fail"})
		return
	}
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)
	h.setJob(ChatJob{
		ID:        jobID,
		SessionID: req.SessionID,
		Status:    "processing",
		CreatedAt: now,
		UpdatedAt: now,
	})

	h.worker.Submit(jobs.Job{
		ID: jobID,
		Operation: func() error {
			history := h.getHistory(req.SessionID)
			response := h.agent.Run(req.Message, history)
			status := "success"
			if response == "I'm having trouble right now. Please try again in a moment." {
				status = "fail"
			}

			h.appendHistory(req.SessionID, agent.ChatMessage{Role: "user", Content: req.Message})
			h.appendHistory(req.SessionID, agent.ChatMessage{Role: "assistant", Content: response})

			h.setJob(ChatJob{
				ID:        jobID,
				SessionID: req.SessionID,
				Status:    status,
				Response:  response,
				CreatedAt: now,
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
			return nil
		},
		OnFail: func(error) {
			h.setJob(ChatJob{
				ID:        jobID,
				SessionID: req.SessionID,
				Status:    "fail",
				Response:  "I'm having trouble right now. Please try again in a moment.",
				CreatedAt: now,
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
		},
	})

	h.writeJSON(w, http.StatusAccepted, ChatResponse{JobID: jobID, Response: "processing", Status: "processing"})
}

func (h *ChatHandler) enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (h *ChatHandler) writeJSON(w http.ResponseWriter, code int, payload ChatResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *ChatHandler) getHistory(sessionID string) []agent.ChatMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]agent.ChatMessage(nil), h.sessionHistory[sessionID]...)
}

func (h *ChatHandler) appendHistory(sessionID string, message agent.ChatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessionHistory[sessionID] = append(h.sessionHistory[sessionID], message)
}

func (h *ChatHandler) setJob(job ChatJob) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.jobStore[job.ID] = job
}

func (h *ChatHandler) getJob(jobID string) (ChatJob, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	job, ok := h.jobStore[jobID]
	return job, ok
}

func (h *ChatHandler) getJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		h.writeJSON(w, http.StatusBadRequest, ChatResponse{Response: "missing job_id", Status: "fail"})
		return
	}

	job, ok := h.getJob(jobID)
	if !ok {
		h.writeJSON(w, http.StatusNotFound, ChatResponse{Response: "job not found", Status: "fail"})
		return
	}

	h.writeJSON(w, http.StatusOK, ChatResponse{JobID: job.ID, Response: job.Response, Status: job.Status})
}
