package toolboxversion

import (
	"fmt"

	"toolBox/pkg/modulecontract"
)

const (
	APIVersion       = "1.0.0"
	WebServerVersion = "1.1.0"
	TestSheetVersion = "1.0.0"
	V10LabVersion    = "1.1.0"
)

var (
	Commit    = "unknown"
	BuildDate = "unknown"
)

type BuildInfo struct {
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

type VersionInfo struct {
	Version string    `json:"version"`
	Build   BuildInfo `json:"build"`
}

func Build() BuildInfo {
	return BuildInfo{
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

func ModuleBuild() modulecontract.BuildInfo {
	return modulecontract.BuildInfo{
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

func Info(version string) VersionInfo {
	return VersionInfo{
		Version: version,
		Build:   Build(),
	}
}

func Banner(componentName string, version string) string {
	return fmt.Sprintf("========================================\n%s v%s\nCommit: %s\nBuild: %s\n========================================", componentName, version, Commit, BuildDate)
}
