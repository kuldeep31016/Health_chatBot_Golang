package agent

type State string

const (
	StateProcess State = "process"
	StateDecide  State = "decide"
	StateAction  State = "action"
	StateRetry   State = "retry"
	StateSuccess State = "success"
	StateFail    State = "fail"
)

type AgentContext struct {
	Query         string
	CurrentState  State
	ToolToUse     string
	RetrievedData map[string]interface{}
	ChatHistory   []ChatMessage
	FinalResponse string
	RetryCount    int
	MaxRetries    int
}

type ChatMessage struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Vector  []float64 `json:"vector,omitempty"`
}
