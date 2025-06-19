/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package entitystore

// Version information set by build flags
var (
	// Version is the semantic version of EntityStore
	Version = "0.2.0"
	
	// GitCommit is the git commit hash (set by build flags)
	GitCommit = "unknown"
	
	// BuildDate is the build date (set by build flags)
	BuildDate = "unknown"
	
	// GoVersion is the Go version used to build
	GoVersion = "unknown"
)

// VersionInfo contains version information
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// GetVersionInfo returns the version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
	}
}