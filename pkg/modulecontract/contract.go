package modulecontract

type ModuleInfo struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Actions     []ModuleAction `json:"actions"`
}

type ModuleAction struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ActionRequest struct {
	Args    []string       `json:"args,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
	Async   bool           `json:"async,omitempty"`
}

type ActionResponse struct {
	ModuleID string         `json:"moduleId"`
	ActionID string         `json:"actionId"`
	JobID    string         `json:"jobId,omitempty"`
	Status   string         `json:"status"`
	Output   map[string]any `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type JobStatus struct {
	ID       string         `json:"id"`
	Status   string         `json:"status"`
	ModuleID string         `json:"moduleId,omitempty"`
	ActionID string         `json:"actionId,omitempty"`
	Output   map[string]any `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
}
