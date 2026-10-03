//go:build windows

package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	updateCheckInterval  = 24 * time.Hour
	updateStartupDelay   = 10 * time.Second
	updateHTTPTimeout    = 20 * time.Second
	maxReleaseJSONSize   = 2 << 20
	maxUpdatePackageSize = 64 << 20
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	HTMLURL    string        `json:"html_url"`
	Assets     []githubAsset `json:"assets"`
}

type updateState struct {
	LastCheckUnix int64  `json:"last_check_unix"`
	ETag          string `json:"etag,omitempty"`
	LastTag       string `json:"last_tag,omitempty"`
}

type parsedVersion struct {
	major int
	minor int
	patch int
}

var (
	updateMu      sync.Mutex
	updateRunning bool
)

func updateStatePath() string {
	return filepath.Join(appDataDir(), updateStateName)
}

func loadUpdateState() updateState {
	var state updateState
	data, err := os.ReadFile(updateStatePath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logError("updater.state", "unable to read update state", err)
		}
		return state
	}
	if err := json.Unmarshal(data, &state); err != nil {
		logError("updater.state", "invalid update state; using defaults", err)
		return updateState{}
	}
	return state
}

func saveUpdateState(state updateState) {
	if err := os.MkdirAll(appDataDir(), 0700); err != nil {
		logError("updater.state", "unable to create application data directory", err)
		return
	}
	data, err := json.Marshal(state)
	if err != nil {
		logError("updater.state", "unable to encode update state", err)
		return
	}
	tmp := updateStatePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		logError("updater.state", "unable to write temporary update state", err)
		return
	}
	if err := os.Rename(tmp, updateStatePath()); err != nil {
		_ = os.Remove(tmp)
		logError("updater.state", "unable to replace update state", err)
	}
}

func scheduleUpdateChecks(cfg *config) {
	if cfg == nil || !cfg.autoUpdate {
		return
	}
	state := loadUpdateState()
	delay := updateStartupDelay
	if state.LastCheckUnix > 0 {
		remaining := updateCheckInterval - time.Since(time.Unix(state.LastCheckUnix, 0))
		if remaining > delay {
			delay = remaining
		}
	}
	time.AfterFunc(delay, func() {
		checkForUpdates(false)
	})
	logEvent("INFO", "updater", fmt.Sprintf("automatic update check scheduled in %s", delay.Round(time.Second)))
}

func startUpdateCheckManually() {
	go checkForUpdates(true)
}

func checkForUpdates(manual bool) {
	if !manual && (globalConfig == nil || !globalConfig.autoUpdate) {
		return
	}
	if !acquireUpdateRun() {
		if manual {
			postUpdateResult("An update check is already in progress.")
		}
		return
	}
	defer releaseUpdateRun()

	state := loadUpdateState()
	etagToUse := state.ETag
	if state.LastTag != "" {
		if last, parseErr := parseReleaseVersion(state.LastTag); parseErr == nil {
			if current, currentErr := parseReleaseVersion(buildVersion); currentErr == nil && last.greaterThan(current) {
				// A previous update attempt saw a newer release but did not finish.
				// Bypass the cached 304 so we can retry the download.
				etagToUse = ""
			}
		}
	}
	release, etag, notModified, err := fetchLatestRelease(etagToUse)
	state.LastCheckUnix = time.Now().Unix()
	if err != nil {
		logError("updater.check", "GitHub release check failed", err)
		saveUpdateState(state)
		if manual {
			postUpdateResult("Unable to check for updates. See lowbar.log for details.")
		}
		return
	}
	if notModified {
		saveUpdateState(state)
		if manual {
			postUpdateResult("LowBar is up to date.")
		}
		return
	}

	state.ETag = etag
	state.LastTag = release.TagName
	saveUpdateState(state)

	latest, err := parseReleaseVersion(release.TagName)
	if err != nil {
		logError("updater.check", "latest release tag is not a supported version", err)
		if manual {
			postUpdateResult("The latest release uses an unsupported version format.")
		}
		return
	}
	current, err := parseReleaseVersion(buildVersion)
	if err != nil {
		logError("updater.check", "current LowBar version is invalid", err)
		return
	}
	if !latest.greaterThan(current) {
		logEvent("INFO", "updater.check", fmt.Sprintf("no update available current=%s latest=%s", buildVersion, release.TagName))
		if manual {
			postUpdateResult("LowBar is up to date.")
		}
		return
	}

	asset, ok := findReleaseAsset(release)
	if !ok {
		logEvent("ERROR", "updater.check", fmt.Sprintf("release %s has no %s asset", release.TagName, releaseAssetName))
		if manual {
			postUpdateResult("The latest release package is unavailable.")
		}
		return
	}
	logEvent("INFO", "updater.check", fmt.Sprintf("update available current=%s latest=%s size=%d", buildVersion, release.TagName, asset.Size))

	zipPath, err := downloadReleaseAsset(asset)
	if err != nil {
		logError("updater.download", "unable to download update package", err)
		if manual {
			postUpdateResult("Unable to download the update. See lowbar.log for details.")
		}
		return
	}

	if err := verifyDownloadedAsset(zipPath, asset.Digest); err != nil {
		_ = os.Remove(zipPath)
		logError("updater.verify", "update package verification failed", err)
		if manual {
			postUpdateResult("The downloaded update failed verification and was discarded.")
		}
		return
	}

	if err := validateReleaseArchive(zipPath); err != nil {
		_ = os.Remove(zipPath)
		logError("updater.verify", "update package structure is invalid", err)
		if manual {
			postUpdateResult("The downloaded update package is invalid and was discarded.")
		}
		return
	}

	if err := launchUpdateHelper(zipPath, release.TagName); err != nil {
		_ = os.Remove(zipPath)
		logError("updater.apply", "unable to start update helper", err)
		if manual {
			postUpdateResult("The update was downloaded but could not be started.")
		}
		return
	}

	logEvent("INFO", "updater.apply", fmt.Sprintf("update helper launched for release %s", release.TagName))
	postMessageW.Call(uintptr(mainHwnd), wmBeginUpdate, 0, 0)
}

func acquireUpdateRun() bool {
	updateMu.Lock()
	defer updateMu.Unlock()
	if updateRunning {
		return false
	}
	updateRunning = true
	return true
}

func releaseUpdateRun() {
	updateMu.Lock()
	updateRunning = false
	updateMu.Unlock()
}

func fetchLatestRelease(existingETag string) (githubRelease, string, bool, error) {
	var release githubRelease
	ctx, cancel := context.WithTimeout(context.Background(), updateHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+githubRepository+"/releases/latest", nil)
	if err != nil {
		return release, "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "LowBar/"+buildVersion)
	if existingETag != "" {
		req.Header.Set("If-None-Match", existingETag)
	}

	client := &http.Client{Timeout: updateHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return release, "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return release, resp.Header.Get("ETag"), true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return release, "", false, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxReleaseJSONSize)
	if err := json.NewDecoder(limited).Decode(&release); err != nil {
		return release, "", false, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return release, "", false, errors.New("GitHub response did not contain a release tag")
	}
	if release.Draft || release.Prerelease {
		return release, resp.Header.Get("ETag"), false, errors.New("latest GitHub release is not a stable published release")
	}
	return release, resp.Header.Get("ETag"), false, nil
}

func findReleaseAsset(release githubRelease) (githubAsset, bool) {
	for _, asset := range release.Assets {
		if asset.Name == releaseAssetName && asset.BrowserDownloadURL != "" {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func downloadReleaseAsset(asset githubAsset) (string, error) {
	base := filepath.Join(os.TempDir(), configDirName)
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(base, "update-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".zip")

	ctx, cancel := context.WithTimeout(context.Background(), updateHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "LowBar/"+buildVersion)
	client := &http.Client{Timeout: updateHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release download returned HTTP %d", resp.StatusCode)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, maxUpdatePackageSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", closeErr
	}
	if written > maxUpdatePackageSize {
		_ = os.Remove(path)
		return "", fmt.Errorf("update package exceeds %d MiB", maxUpdatePackageSize/(1<<20))
	}
	return path, nil
}

func verifyDownloadedAsset(path, digest string) error {
	if digest == "" {
		return errors.New("GitHub did not provide a SHA-256 digest for the release asset")
	}
	const prefix = "sha256:"
	if !strings.HasPrefix(strings.ToLower(digest), prefix) {
		return fmt.Errorf("unsupported GitHub asset digest %q", digest)
	}
	expected := strings.TrimPrefix(strings.ToLower(digest), prefix)
	if len(expected) != sha256.Size*2 {
		return errors.New("invalid SHA-256 digest length")
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("invalid SHA-256 digest: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("SHA-256 mismatch expected=%s actual=%s", expected, actual)
	}
	return nil
}

func validateReleaseArchive(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	required := map[string]bool{
		"LowBar.exe":             false,
		"LowBarExplorerHook.dll": false,
		"Assets":                 false,
	}
	for _, file := range reader.File {
		name := filepath.Clean(file.Name)
		if filepath.IsAbs(name) || name == "." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
			return fmt.Errorf("archive contains an unsafe path %q", file.Name)
		}
		switch strings.ReplaceAll(name, `\`, "/") {
		case "LowBar.exe":
			required["LowBar.exe"] = true
		case "LowBarExplorerHook.dll":
			required["LowBarExplorerHook.dll"] = true
		case "Assets", "Assets/":
			required["Assets"] = true
		}
		if len(name) > 240 {
			return fmt.Errorf("archive path is too long: %q", file.Name)
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("required archive member missing: %s", name)
		}
	}
	return nil
}

func launchUpdateHelper(zipPath, releaseTag string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	helperDir := filepath.Join(os.TempDir(), configDirName)
	if err := os.MkdirAll(helperDir, 0700); err != nil {
		return err
	}
	helperPath := filepath.Join(helperDir, "LowBarUpdater.exe")
	if err := copyFile(exe, helperPath); err != nil {
		return fmt.Errorf("copy updater helper: %w", err)
	}

	pid := strconv.Itoa(os.Getpid())
	cmd := exec.Command(helperPath, "--apply-update", zipPath, exe, filepath.Join(filepath.Dir(exe), explorerHookDLLName), filepath.Join(filepath.Dir(exe), "Assets"), pid, releaseTag)
	cmd.Dir = filepath.Dir(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func runUpdateHelper(args []string) int {
	if len(args) != 7 || args[0] != "--apply-update" {
		return 2
	}
	zipPath := args[1]
	targetExe := args[2]
	targetHook := args[3]
	targetAssets := args[4]
	pid, err := strconv.ParseUint(args[5], 10, 32)
	if err != nil {
		return 2
	}
	releaseTag := args[6]

	waitForParent(uint32(pid))

	applyErr := applyUpdatePackage(zipPath, targetExe, targetHook, targetAssets, releaseTag)
	if applyErr != nil {
		openLog()
		logError("updater.helper", "automatic update failed; attempting to restart LowBar", applyErr)
		closeLog()
		startUpdatedLowBar(targetExe)
		return 1
	}

	startUpdatedLowBar(targetExe)
	return 0
}

func waitForParent(pid uint32) {
	if pid == 0 {
		return
	}
	process, _, _ := openProcess.Call(uintptr(processSynchronize), 0, uintptr(pid))
	if process == 0 {
		return
	}
	defer closeHandle.Call(process)
	_, _, _ = waitForSingleObject.Call(process, infinite)
}

func applyUpdatePackage(zipPath, targetExe, targetHook, targetAssets, releaseTag string) error {
	if _, err := os.Stat(zipPath); err != nil {
		return err
	}

	stageRoot := filepath.Join(os.TempDir(), configDirName, "stage-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	defer os.RemoveAll(stageRoot)
	if err := extractReleaseArchive(zipPath, stageRoot); err != nil {
		return err
	}

	stagedExe := filepath.Join(stageRoot, "LowBar.exe")
	stagedHook := filepath.Join(stageRoot, explorerHookDLLName)
	stagedAssets := filepath.Join(stageRoot, "Assets")

	if err := replaceExecutable(stagedExe, targetExe); err != nil {
		return err
	}
	if err := replaceAssets(stagedAssets, targetAssets); err != nil {
		return err
	}
	if err := handleHookUpdate(stagedHook, targetHook); err != nil {
		// The executable/assets update is already complete. Do not leave LowBar
		// stopped merely because Windows refused the optional hook replacement.
		logHelperEvent("Explorer hook update could not be scheduled: " + err.Error())
	}

	_ = os.Remove(zipPath)
	logHelperEvent(fmt.Sprintf("LowBar updated to %s", releaseTag))
	return nil
}

func extractReleaseArchive(zipPath, destination string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := os.MkdirAll(destination, 0700); err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		clean := filepath.Clean(file.Name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe archive path %q", file.Name)
		}
		output := filepath.Join(rootAbs, clean)
		outputAbs, err := filepath.Abs(output)
		if err != nil {
			return err
		}
		prefix := rootAbs + string(os.PathSeparator)
		if outputAbs != rootAbs && !strings.HasPrefix(strings.ToLower(outputAbs), strings.ToLower(prefix)) {
			return fmt.Errorf("archive path escapes staging directory: %q", file.Name)
		}
		info := file.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(outputAbs, 0700); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("archive contains a non-regular file %q", file.Name)
		}
		if err := os.MkdirAll(filepath.Dir(outputAbs), 0700); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(outputAbs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeOutErr := out.Close()
		closeInErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOutErr != nil {
			return closeOutErr
		}
		if closeInErr != nil {
			return closeInErr
		}
	}
	return nil
}

func replaceExecutable(staged, target string) error {
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("move old executable: %w", err)
	}
	if err := copyFile(staged, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install new executable: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

func replaceAssets(staged, target string) error {
	if _, err := os.Stat(staged); err != nil {
		return err
	}
	backup := target + ".old"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("move old assets: %w", err)
		}
	}
	if err := copyDir(staged, target); err != nil {
		_ = os.RemoveAll(target)
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, target)
		}
		return fmt.Errorf("install new assets: %w", err)
	}
	_ = os.RemoveAll(backup)
	return nil
}

func handleHookUpdate(staged, target string) error {
	stagedHash, err := hashFile(staged)
	if err != nil {
		return err
	}
	currentHash, currentErr := hashFile(target)
	if currentErr == nil && stagedHash == currentHash {
		return nil
	}

	pendingDir := filepath.Join(appDataDir(), "pending")
	if err := os.MkdirAll(pendingDir, 0700); err != nil {
		return err
	}
	pending := filepath.Join(pendingDir, explorerHookDLLName)
	if err := copyFile(staged, pending); err != nil {
		return fmt.Errorf("stage updated Explorer hook: %w", err)
	}
	if err := scheduleDelayedFileReplace(pending, target); err != nil {
		return fmt.Errorf("schedule Explorer hook replacement: %w", err)
	}
	logHelperEvent("Explorer hook changed; replacement scheduled for the next Windows restart")
	return nil
}

func startUpdatedLowBar(targetExe string) {
	cmd := exec.Command(targetExe)
	cmd.Dir = filepath.Dir(targetExe)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		openLog()
		logError("updater.helper", "unable to restart updated LowBar", err)
		closeLog()
		return
	}
}

func copyFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func copyDir(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		return copyFile(path, target)
	})
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func parseReleaseVersion(raw string) (parsedVersion, error) {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(strings.TrimPrefix(clean, "v"), "V")
	clean = strings.SplitN(clean, "+", 2)[0]
	clean = strings.SplitN(clean, "-", 2)[0]
	parts := strings.Split(clean, ".")
	if len(parts) == 2 {
		// LowBar historically used tags like "1.2" for product version "0.1.2".
		clean = "0." + clean
		parts = strings.Split(clean, ".")
	}
	if len(parts) != 3 {
		return parsedVersion{}, fmt.Errorf("unsupported version %q", raw)
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" {
			return parsedVersion{}, fmt.Errorf("unsupported version %q", raw)
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return parsedVersion{}, fmt.Errorf("unsupported version %q", raw)
		}
		values[i] = value
	}
	return parsedVersion{major: values[0], minor: values[1], patch: values[2]}, nil
}

func (v parsedVersion) greaterThan(other parsedVersion) bool {
	if v.major != other.major {
		return v.major > other.major
	}
	if v.minor != other.minor {
		return v.minor > other.minor
	}
	return v.patch > other.patch
}

func postUpdateResult(message string) {
	updateMu.Lock()
	pendingUpdateMessage = message
	updateMu.Unlock()
	if mainHwnd != 0 {
		postMessageW.Call(uintptr(mainHwnd), wmUpdateResult, 0, 0)
	}
}

func takePendingUpdateMessage() string {
	updateMu.Lock()
	defer updateMu.Unlock()
	message := pendingUpdateMessage
	pendingUpdateMessage = ""
	return message
}

func logHelperEvent(message string) {
	openLog()
	logEvent("INFO", "updater.helper", message)
	closeLog()
}
