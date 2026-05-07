package model

import "time"

const (
	TestRunStatusPending   = "pending"
	TestRunStatusRunning   = "running"
	TestRunStatusCompleted = "completed"
	TestRunStatusCanceled  = "canceled"
	TestRunStatusArchived  = "archived"

	RunSheetStatusPending = "pending"
	RunSheetStatusPassed  = "passed"
	RunSheetStatusFailed  = "failed"
	RunSheetStatusBlocked = "blocked"
	RunSheetStatusSkipped = "skipped"
)

type TestPlan struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	MockupSettings string     `json:"mockupSettings"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

type TestSheet struct {
	ID             int64           `json:"id"`
	PlanID         int64           `json:"planId"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Prerequisites  string          `json:"prerequisites"`
	Config         string          `json:"config"`
	Command        string          `json:"command"`
	Notes          string          `json:"notes"`
	Action         string          `json:"action"`
	ExpectedResult string          `json:"expectedResult"`
	ExecutionOrder int             `json:"executionOrder"`
	MockupSettings string          `json:"mockupSettings"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Steps          []TestSheetStep `json:"steps,omitempty"`
	Documents      []TestDocument  `json:"documents,omitempty"`
}

type TestSheetStep struct {
	ID             int64          `json:"id"`
	SheetID        int64          `json:"sheetId"`
	Action         string         `json:"action"`
	Field          string         `json:"field"`
	ExpectedResult string         `json:"expectedResult"`
	ExecutionOrder int            `json:"executionOrder"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	Documents      []TestDocument `json:"documents,omitempty"`
}

type TestDocument struct {
	ID           int64     `json:"id"`
	PlanID       int64     `json:"planId"`
	OriginalName string    `json:"originalName"`
	StoredName   string    `json:"storedName"`
	StoragePath  string    `json:"-"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	SHA256       string    `json:"sha256"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"createdAt"`
}

type TestAttachment struct {
	ID        int64     `json:"id"`
	SheetID   int64     `json:"sheetId"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	MimeType  string    `json:"mimeType"`
	CreatedAt time.Time `json:"createdAt"`
}

type TestRun struct {
	ID         int64      `json:"id"`
	RunNumber  int        `json:"runNumber"`
	PlanID     int64      `json:"planId"`
	PlanName   string     `json:"planName"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Sheets     []RunSheet `json:"sheets,omitempty"`
}

type TestRunSummary struct {
	ID           int64      `json:"id"`
	RunNumber    int        `json:"runNumber"`
	PlanID       int64      `json:"planId"`
	PlanName     string     `json:"planName"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	TotalSheets  int        `json:"totalSheets"`
	TotalSteps   int        `json:"totalSteps"`
	PendingSteps int        `json:"pendingSteps"`
	PassedSteps  int        `json:"passedSteps"`
	FailedSteps  int        `json:"failedSteps"`
	BlockedSteps int        `json:"blockedSteps"`
	SkippedSteps int        `json:"skippedSteps"`
}

type TestPlanSummary struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	SheetCount  int             `json:"sheetCount"`
	RunCount    int             `json:"runCount"`
	LatestRun   *TestRunSummary `json:"latestRun,omitempty"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty"`
}

type RunSheet struct {
	ID             int64          `json:"id"`
	RunID          int64          `json:"runId"`
	SourceSheetID  *int64         `json:"sourceSheetId,omitempty"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Prerequisites  string         `json:"prerequisites"`
	Config         string         `json:"config"`
	Command        string         `json:"command"`
	Notes          string         `json:"notes"`
	Action         string         `json:"action"`
	ExpectedResult string         `json:"expectedResult"`
	ExecutionOrder int            `json:"executionOrder"`
	Status         string         `json:"status"`
	ActualResult   string         `json:"actualResult"`
	Comment        string         `json:"comment"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	Steps          []RunStep      `json:"steps,omitempty"`
	Evidences      []Evidence     `json:"evidences,omitempty"`
	Documents      []TestDocument `json:"documents,omitempty"`
}

type RunStep struct {
	ID             int64          `json:"id"`
	RunSheetID     int64          `json:"runSheetId"`
	SourceStepID   *int64         `json:"sourceStepId,omitempty"`
	Action         string         `json:"action"`
	Field          string         `json:"field"`
	ExpectedResult string         `json:"expectedResult"`
	ExecutionOrder int            `json:"executionOrder"`
	Status         string         `json:"status"`
	ActualResult   string         `json:"actualResult"`
	Comment        string         `json:"comment"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	Documents      []TestDocument `json:"documents,omitempty"`
	Evidences      []Evidence     `json:"evidences,omitempty"`
}

type Evidence struct {
	ID         int64     `json:"id"`
	RunSheetID int64     `json:"runSheetId"`
	RunStepID  int64     `json:"runStepId,omitempty"`
	Name       string    `json:"name"`
	Path       string    `json:"-"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int64     `json:"sizeBytes"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"createdAt"`
}

type PlanInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	MockupSettings string `json:"mockupSettings"`
}

type SheetInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Prerequisites  string `json:"prerequisites"`
	Config         string `json:"config"`
	Command        string `json:"command"`
	Notes          string `json:"notes"`
	Action         string `json:"action"`
	ExpectedResult string `json:"expectedResult"`
	ExecutionOrder int    `json:"executionOrder"`
	MockupSettings string `json:"mockupSettings"`
}

type ReorderInput struct {
	SheetIDs []int64 `json:"sheetIds"`
	StepIDs  []int64 `json:"stepIds"`
}

type StepInput struct {
	Action         string `json:"action"`
	Field          string `json:"field"`
	ExpectedResult string `json:"expectedResult"`
	ExecutionOrder int    `json:"executionOrder"`
}

type RunSheetResultInput struct {
	Status       string `json:"status"`
	ActualResult string `json:"actualResult"`
	Comment      string `json:"comment"`
}

type RunStepResultInput struct {
	Status       string `json:"status"`
	ActualResult string `json:"actualResult"`
	Comment      string `json:"comment"`
}
