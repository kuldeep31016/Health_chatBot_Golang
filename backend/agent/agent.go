package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"health-assistant/backend/jobs"
	"health-assistant/backend/llm"
	"health-assistant/backend/memory"
	"health-assistant/backend/tools"
)

const fallbackResponse = "I'm having trouble right now. Please try again in a moment."

type Agent struct {
	Gemini       *llm.GeminiClient
	Embedder     *memory.EmbeddingClient
	Memory       *tools.MemoryTool
	Worker       *jobs.WorkerPool
	MaxRetries   int
	LangGraphURL string
	HTTPClient   *http.Client
}

func NewAgent(g *llm.GeminiClient, e *memory.EmbeddingClient, mem *tools.MemoryTool, worker *jobs.WorkerPool) *Agent {
	return &Agent{
		Gemini:       g,
		Embedder:     e,
		Memory:       mem,
		Worker:       worker,
		MaxRetries:   3,
		LangGraphURL: strings.TrimSpace(os.Getenv("LANGGRAPH_API_URL")),
		HTTPClient:   &http.Client{Timeout: 40 * time.Second},
	}
}

func (a *Agent) Run(query string, history []ChatMessage) string {
	if response, ok := tools.TryAnswerProfileQuestion(query); ok {
		a.storeMemoryAsync(query, response)
		return response
	}

	if strings.TrimSpace(a.LangGraphURL) != "" {
		if response, err := a.runViaLangGraph(query, history); err == nil && response != "" {
			return response
		} else if err != nil {
			log.Printf("agent: langgraph call failed, falling back to local pipeline: %v", err)
		}
	}

	ctx := &AgentContext{
		Query:         query,
		CurrentState:  StateProcess,
		RetrievedData: make(map[string]interface{}),
		ChatHistory:   history,
		MaxRetries:    a.MaxRetries,
	}

	classification := "general"
	pendingTools := make([]string, 0)
	lastErr := error(nil)

	for {
		switch ctx.CurrentState {
		case StateProcess:
			classification = classifyQuery(query)
			ctx.CurrentState = Transition(StateProcess, true, ctx.RetryCount, ctx.MaxRetries)

		case StateDecide:
			if len(pendingTools) == 0 {
				pendingTools = decideTools(classification, query, ctx.RetrievedData)
			}
			if len(pendingTools) == 0 {
				ctx.CurrentState = StateSuccess
				continue
			}
			ctx.ToolToUse = pendingTools[0]
			pendingTools = pendingTools[1:]
			ctx.CurrentState = Transition(StateDecide, true, ctx.RetryCount, ctx.MaxRetries)

		case StateAction:
			err := a.runToolWithWorker(ctx)
			if err != nil {
				lastErr = err
				log.Printf("agent: tool execution failed (%s): %v", ctx.ToolToUse, err)
				ctx.CurrentState = Transition(StateAction, false, ctx.RetryCount, ctx.MaxRetries)
				continue
			}

			if len(pendingTools) > 0 {
				ctx.CurrentState = StateDecide
			} else {
				ctx.CurrentState = Transition(StateAction, true, ctx.RetryCount, ctx.MaxRetries)
			}

		case StateRetry:
			done := make(chan bool, 1)
			go func() {
				ctx.RetryCount++
				time.Sleep(2 * time.Second)
				done <- ctx.RetryCount < ctx.MaxRetries
			}()

			if <-done {
				ctx.CurrentState = Transition(StateRetry, false, ctx.RetryCount, ctx.MaxRetries)
			} else {
				ctx.CurrentState = StateFail
			}

		case StateSuccess:
			resp, err := a.generateWithWorker(ctx)
			if err != nil {
				lastErr = err
				log.Printf("agent: response generation failed: %v", err)
				if ctx.RetryCount < ctx.MaxRetries {
					ctx.RetryCount++
					time.Sleep(2 * time.Second)
					ctx.CurrentState = StateSuccess
				} else {
					ctx.CurrentState = StateFail
				}
				continue
			}
			ctx.FinalResponse = resp
			a.storeMemoryAsync(ctx.Query, ctx.FinalResponse)
			return ctx.FinalResponse

		case StateFail:
			if lastErr != nil {
				log.Printf("agent: returning fallback after retries exhausted: %v", lastErr)
			} else {
				log.Printf("agent: returning fallback after retries exhausted")
			}
			return fallbackResponse

		default:
			return fallbackResponse
		}
	}
}

func (a *Agent) runViaLangGraph(query string, history []ChatMessage) (string, error) {
	if a.HTTPClient == nil {
		a.HTTPClient = &http.Client{Timeout: 40 * time.Second}
	}

	context := a.buildLangGraphContext(query)

	reqBody := map[string]interface{}{
		"query":   query,
		"history": history,
		"context": context,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	var output string
	err = jobs.WithRetry(jobs.RetryConfig{MaxAttempts: 3, Delay: 2 * time.Second}, func() error {
		endpoint := strings.TrimRight(a.LangGraphURL, "/") + "/run"
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return fmt.Errorf("langgraph status %d: %s", resp.StatusCode, string(body))
		}

		var parsed struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return err
		}
		if parsed.Response == "" {
			return fmt.Errorf("empty langgraph response")
		}
		output = parsed.Response
		return nil
	})
	if err != nil {
		return "", err
	}

	a.storeMemoryAsync(query, output)
	return output, nil
}

func (a *Agent) buildLangGraphContext(query string) map[string]interface{} {
	ctx := map[string]interface{}{}

	if profile, err := tools.GetUserProfile(); err == nil && len(profile) > 0 {
		ctx["user"] = profile
	}

	if healthData := tools.GetHealthData(query); len(healthData) > 0 {
		ctx["health"] = healthData
	}

	if a.Embedder != nil && a.Memory != nil {
		if vec, err := a.Embedder.EmbedText(query); err == nil {
			items := a.Memory.RetrieveRelevantMemory(vec, 3)
			if len(items) > 0 {
				ctx["memory"] = map[string]interface{}{"items": items}
			}
		}
	}

	return ctx
}

func classifyQuery(query string) string {
	q := strings.ToLower(query)
	if strings.Contains(q, "earlier") || strings.Contains(q, "remember") || strings.Contains(q, "discuss") {
		return "memory"
	}

	healthTerms := []string{"health", "heart", "hrv", "bp", "glucose", "dizzy", "weak", "tired", "fatigue", "exercise", "workout", "run", "allergy", "appointment", "fitness", "hemoglobin", "haemoglobin", "hb", "biomarker", "vitamin"}
	for _, term := range healthTerms {
		if strings.Contains(q, term) {
			return "health"
		}
	}
	return "general"
}

func decideTools(classification, query string, data map[string]interface{}) []string {
	switch classification {
	case "health":
		toolsToRun := []string{"get_user_profile"}
		healthData := tools.GetHealthData(query)
		if len(healthData) > 0 {
			toolsToRun = append(toolsToRun, "get_health_data")
		}
		if strings.Contains(strings.ToLower(query), "earlier") || strings.Contains(strings.ToLower(query), "remember") {
			toolsToRun = append(toolsToRun, "get_memory")
		}
		return toolsToRun
	case "memory":
		return []string{"get_memory"}
	default:
		if strings.Contains(strings.ToLower(query), "profile") {
			return []string{"get_user_profile"}
		}
		return nil
	}
}

func (a *Agent) runToolWithWorker(ctx *AgentContext) error {
	switch ctx.ToolToUse {
	case "get_user_profile":
		profile, err := tools.GetUserProfile()
		if err != nil {
			return err
		}
		ctx.RetrievedData["profile"] = profile
	case "get_health_data":
		ctx.RetrievedData["health"] = tools.GetHealthData(ctx.Query)
	case "get_memory":
		if a.Embedder == nil || a.Memory == nil {
			ctx.RetrievedData["memory"] = []tools.MemoryEntry{}
			return nil
		}
		vec, err := a.Embedder.EmbedText(ctx.Query)
		if err != nil {
			return err
		}
		ctx.RetrievedData["memory"] = a.Memory.RetrieveRelevantMemory(vec, 3)
	default:
		return fmt.Errorf("unknown tool: %s", ctx.ToolToUse)
	}

	return nil
}

func (a *Agent) generateWithWorker(ctx *AgentContext) (string, error) {
	profile, _ := ctx.RetrievedData["profile"].(map[string]interface{})
	healthData, _ := ctx.RetrievedData["health"].(map[string]interface{})
	memoryContext := map[string]interface{}{"items": ctx.RetrievedData["memory"]}

	resp, err := a.Gemini.GenerateResponse(profile, healthData, memoryContext, ctx.Query)
	if err != nil {
		return "", err
	}

	return resp, nil
}

func (a *Agent) storeMemoryAsync(query, response string) {
	if a.Worker == nil || a.Embedder == nil || a.Memory == nil {
		return
	}

	a.Worker.Submit(jobs.Job{
		ID: "store-memory",
		Operation: func() error {
			queryVec, err := a.Embedder.EmbedText(query)
			if err != nil {
				return err
			}
			respVec, err := a.Embedder.EmbedText(response)
			if err != nil {
				return err
			}
			a.Memory.StoreMemory("User: "+query, queryVec)
			a.Memory.StoreMemory("Assistant: "+response, respVec)
			return nil
		},
	})
}
