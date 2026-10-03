//go:build windows

package main

const (
	githubRepository = "chematic/LowBar"
	releaseAssetName = "LowBar.zip"
	updateStateName  = "update-state.json"
)

// buildVersion is replaced by scripts/package-release.ps1 for official builds.
// The fallback keeps local development builds usable.
var buildVersion = "0.1.3"
