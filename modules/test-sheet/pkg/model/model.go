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

type TestGroup struct {
	ID             int64           `json:"id"`
	PlanID         int64           `json:"planId"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	ExecutionOrder int             `json:"executionOrder"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Sheets         []TestSheet     `json:"sheets,omitempty"`
	LatestRun      *TestRunSummary `json:"latestRun,omitempty"`
	RunCount       int             `json:"runCount"`
	SheetCount     int             `json:"sheetCount"`
}

type TestSheet struct {
	ID             int64           `json:"id"`
	PlanID         int64           `json:"planId"`
	GroupID        int64           `json:"groupId"`
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
	GroupID    int64      `json:"groupId"`
	PlanName   string     `json:"planName"`
	GroupName  string     `json:"groupName"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Groups     []RunGroup `json:"groups,omitempty"`
	Sheets     []RunSheet `json:"sheets,omitempty"`
}

type RunGroup struct {
	ID             int64      `json:"id"`
	RunID          int64      `json:"runId"`
	SourceGroupID  *int64     `json:"sourceGroupId,omitempty"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	ExecutionOrder int        `json:"executionOrder"`
	CreatedAt      time.Time  `json:"createdAt"`
	Sheets         []RunSheet `json:"sheets,omitempty"`
}

type TestRunSummary struct {
	ID            int64      `json:"id"`
	RunNumber     int        `json:"runNumber"`
	PlanID        int64      `json:"planId"`
	GroupID       int64      `json:"groupId"`
	PlanName      string     `json:"planName"`
	GroupName     string     `json:"groupName"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	TotalSheets   int        `json:"totalSheets"`
	TotalGroups   int        `json:"totalGroups"`
	PendingGroups int        `json:"pendingGroups"`
	PassedGroups  int        `json:"passedGroups"`
	FailedGroups  int        `json:"failedGroups"`
	BlockedGroups int        `json:"blockedGroups"`
	SkippedGroups int        `json:"skippedGroups"`
	TotalSteps    int        `json:"totalSteps"`
	PendingSteps  int        `json:"pendingSteps"`
	PassedSteps   int        `json:"passedSteps"`
	FailedSteps   int        `json:"failedSteps"`
	BlockedSteps  int        `json:"blockedSteps"`
	SkippedSteps  int        `json:"skippedSteps"`
}

type TestPlanSummary struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	SheetCount  int             `json:"sheetCount"`
	GroupCount  int             `json:"groupCount"`
	RunCount    int             `json:"runCount"`
	LatestRun   *TestRunSummary `json:"latestRun,omitempty"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty"`
}

type RunSheet struct {
	ID             int64          `json:"id"`
	RunID          int64          `json:"runId"`
	RunGroupID     int64          `json:"runGroupId"`
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
	ExportPath string    `json:"exportPath,omitempty"`
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

type GroupInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ExecutionOrder int    `json:"executionOrder"`
}

type DuplicateGroupInput struct {
	TargetPlanID int64  `json:"targetPlanId"`
	Name         string `json:"name"`
}

type ReorderInput struct {
	SheetIDs []int64 `json:"sheetIds"`
	StepIDs  []int64 `json:"stepIds"`
	GroupIDs []int64 `json:"groupIds"`
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

type ExportOptions struct {
	IncludeGroups    bool `json:"includeGroups"`
	IncludeSheets    bool `json:"includeSheets"`
	IncludeSteps     bool `json:"includeSteps"`
	IncludeDocuments bool `json:"includeDocuments"`
	IncludeHistory   bool `json:"includeHistory"`
	IncludeEvidences bool `json:"includeEvidences"`
}

type ImportPreview struct {
	PlanName      string `json:"planName"`
	SchemaVersion int    `json:"schemaVersion"`
	Groups        int    `json:"groups"`
	Sheets        int    `json:"sheets"`
	Steps         int    `json:"steps"`
	Documents     int    `json:"documents"`
	Runs          int    `json:"runs"`
	Evidences     int    `json:"evidences"`
}

type ImportResult struct {
	PlanID int64  `json:"planId"`
	Name   string `json:"name"`
}
