package main

import (
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
	"runtime"
	"strconv"
	"strings"
	"time"
)

var Version = "dev"

const (
	githubRepo  = "xadomas2012/Plane-crazy-multitool"
	releasesAPI = "https://api.github.com/repos/" +
		githubRepo +
		"/releases?per_page=100"
)

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Body       string        `json:"body"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
}

type updateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseName    string
	ReleaseNotes   string
	Asset          githubAsset
}

const (
	releaseImportanceNormal    = "normal"
	releaseImportanceImportant = "important"
	releaseImportanceCritical  = "critical"
	releaseImportanceMarker    = "<!-- pc-multitool-importance:"
)

func releaseImportance(release githubRelease) string {
	for _, rawLine := range strings.Split(release.Body, "\n") {
		line := strings.TrimSpace(rawLine)

		if !strings.HasPrefix(line, releaseImportanceMarker) ||
			!strings.HasSuffix(line, "-->") {
			continue
		}

		value :=
			strings.TrimSpace(
				strings.TrimSuffix(
					strings.TrimPrefix(
						line,
						releaseImportanceMarker,
					),
					"-->",
				),
			)

		switch strings.ToLower(value) {
		case releaseImportanceNormal:
			return releaseImportanceNormal
		case releaseImportanceImportant:
			return releaseImportanceImportant
		case releaseImportanceCritical:
			return releaseImportanceCritical
		}
	}

	return releaseImportanceNormal
}

func releaseAllowedForChannel(
	release githubRelease,
	channel string,
) bool {

	switch channel {

	case updateChannelIronclad:

		importance := releaseImportance(release)

		return importance == releaseImportanceImportant ||
			importance == releaseImportanceCritical

	case updateChannelEdge,
		updateChannelWorkshop:

		return true

	default:

		return false
	}
}

func selectLatestEligibleRelease(
	releases []githubRelease,
	channel string,
	currentVersion string,
) (githubRelease, bool) {

	currentVersion =
		strings.TrimSpace(currentVersion)

	allowPrerelease :=
		currentVersion != "dev" &&
			isPrereleaseVersion(currentVersion)

	var selected githubRelease
	found := false

	for _, candidate := range releases {

		if strings.TrimSpace(candidate.TagName) == "" {
			continue
		}

		if !releaseAllowedForChannel(
			candidate,
			channel,
		) {
			continue
		}

		// Stable builds never jump to a prerelease.
		// Prerelease builds may continue following prereleases.
		if candidate.Prerelease &&
			!allowPrerelease {

			continue
		}

		if !found ||
			isNewerVersion(
				candidate.TagName,
				selected.TagName,
			) {

			selected = candidate
			found = true
		}
	}

	return selected, found
}

func selectUpdateRelease(
	releases []githubRelease,
	channel string,
	currentVersion string,
) (githubRelease, bool) {

	selected, found :=
		selectLatestEligibleRelease(
			releases,
			channel,
			currentVersion,
		)

	if !found {
		return githubRelease{}, false
	}

	if currentVersion != "dev" &&
		!isNewerVersion(
			selected.TagName,
			currentVersion,
		) {

		return githubRelease{}, false
	}

	return selected, true
}

func getChannelTargetUpdate(
	channel string,
	currentVersion string,
) (updateInfo, error) {

	releases, err := fetchReleaseList()
	if err != nil {
		return updateInfo{}, err
	}

	release, found :=
		selectLatestEligibleRelease(
			releases,
			channel,
			currentVersion,
		)

	if !found {
		return updateInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
		}, nil
	}

	info := updateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseName:    release.Name,
		ReleaseNotes:   release.Body,
	}

	if currentVersion == "dev" &&
		channel == updateChannelWorkshop {
		return info, nil
	}

	asset, err :=
		findPlatformAsset(release.Assets)
	if err != nil {
		return updateInfo{}, err
	}

	info.Asset = asset

	return info, nil
}

const updateCacheTTL = 6 * time.Hour

type releaseCache struct {
	FetchedAt time.Time       `json:"fetched_at"`
	Releases  []githubRelease `json:"releases"`
}

func releaseCachePath(name string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}

	dir := filepath.Join(cacheDir, "pc-multitool")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}

	return filepath.Join(dir, name)
}

func updateCompletionPath() string {
	return releaseCachePath("update-complete")
}

func updatePendingPath() string {
	return releaseCachePath("update-pending")
}

// normalizeUpdateVersion makes release tags (v2.3.0) comparable with
// the version embedded in the executable (2.3.0).
func normalizeUpdateVersion(version string) string {
	version = strings.TrimSpace(version)
	if strings.HasPrefix(version, "v") || strings.HasPrefix(version, "V") {
		version = version[1:]
	}
	return strings.TrimSpace(version)
}

func consumeUpdateMarker(path, currentVersion string) string {
	if path == "" {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	_ = os.Remove(path)

	version := strings.TrimSpace(string(data))
	if version == "" || normalizeUpdateVersion(version) != normalizeUpdateVersion(currentVersion) {
		return ""
	}
	return version
}

// writePendingUpdate records the expected version before handing off to the
// updater. Older bundled helpers do not write completion markers, so the new
// application validates this marker against its compiled-in version at launch.
func writePendingUpdate(version string) {
	version = strings.TrimSpace(version)
	path := updatePendingPath()
	if version == "" || path == "" {
		return
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, []byte(version), 0600); err != nil {
		return
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
	}
}

func consumeUpdateCompletion() string {
	completionPath := updateCompletionPath()
	pendingPath := updatePendingPath()
	if Version == "dev" {
		if completionPath != "" {
			_ = os.Remove(completionPath)
		}
		if pendingPath != "" {
			_ = os.Remove(pendingPath)
		}
		return ""
	}

	if version := consumeUpdateMarker(completionPath, Version); version != "" {
		return version
	}
	return consumeUpdateMarker(pendingPath, Version)
}

func loadReleaseCache(name string) ([]githubRelease, bool, []githubRelease) {
	path := releaseCachePath(name)
	if path == "" {
		return nil, false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, nil
	}

	var cache releaseCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false, nil
	}

	if len(cache.Releases) == 0 {
		return nil, false, nil
	}

	if time.Since(cache.FetchedAt) <= updateCacheTTL {
		return cache.Releases, true, cache.Releases
	}

	return nil, false, cache.Releases
}

func saveReleaseCache(name string, releases []githubRelease) {
	path := releaseCachePath(name)
	if path == "" || len(releases) == 0 {
		return
	}

	data, err := json.Marshal(releaseCache{
		FetchedAt: time.Now(),
		Releases:  releases,
	})
	if err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0600)
}

const releasesPerPage = 100

func fetchReleasePages(client *http.Client, endpoint string) ([]githubRelease, error) {
	var all []githubRelease

	for page := 1; ; page++ {
		separator := "?"
		if strings.Contains(endpoint, "?") {
			separator = "&"
		}
		pageURL := fmt.Sprintf("%s%spage=%d", endpoint, separator, page)

		req, err := http.NewRequest(http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			status := resp.StatusCode
			_ = resp.Body.Close()
			return nil, fmt.Errorf("github API returned HTTP %d", status)
		}

		var pageReleases []githubRelease
		err = json.NewDecoder(resp.Body).Decode(&pageReleases)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if len(pageReleases) == 0 {
			if page == 1 {
				return nil, errors.New("GitHub returned no releases")
			}
			break
		}

		all = append(all, pageReleases...)
		if len(pageReleases) < releasesPerPage {
			break
		}
	}

	return all, nil
}

func fetchReleaseList() ([]githubRelease, error) {
	cached, fresh, stale := loadReleaseCache("releases.json")
	if fresh {
		return cached, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	releases, err := fetchReleasePages(client, releasesAPI)
	if err != nil {
		if len(stale) > 0 {
			return stale, nil
		}
		return nil, err
	}

	saveReleaseCache("releases.json", releases)
	return releases, nil
}

func isPrereleaseVersion(version string) bool {
	version =
		strings.TrimPrefix(
			strings.TrimSpace(version),
			"v",
		)

	return strings.Contains(version, "-")
}

func getLatestUpdate() (updateInfo, error) {
	cfg := loadConfig()

	channel := cfg.UpdateChannel

	if channel != updateChannelWorkshop &&
		channel != updateChannelEdge &&
		channel != updateChannelIronclad {

		channel =
			defaultUpdateChannel()
	}

	releases, err := fetchReleaseList()

	if err != nil {
		return updateInfo{}, err
	}

	release, found :=
		selectUpdateRelease(
			releases,
			channel,
			Version,
		)

	if !found {

		return updateInfo{
			CurrentVersion: Version,
			LatestVersion:  Version,
		}, nil
	}

	info := updateInfo{
		CurrentVersion: Version,
		LatestVersion:  release.TagName,
		ReleaseName:    release.Name,
		ReleaseNotes:   release.Body,
	}

	// Workshop builds can manually inspect the latest release,
	// but they do not participate in automatic update installation.
	if Version == "dev" {
		return info, nil
	}

	asset, err :=
		findPlatformAsset(
			release.Assets,
		)

	if err != nil {
		return updateInfo{}, err
	}

	info.Asset = asset

	return info, nil
}

func findPlatformAsset(
	assets []githubAsset,
) (githubAsset, error) {

	expectedPrefix := ""

	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			expectedPrefix = "PC-Multitool-Linux-x64-"
		}

	case "windows":
		if runtime.GOARCH == "amd64" {
			expectedPrefix = "PC-Multitool-Windows-x64-"
		}

	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			expectedPrefix = "PC-Multitool-macOS-x64-"
		case "arm64":
			expectedPrefix = "PC-Multitool-macOS-arm64-"
		}
	}

	if expectedPrefix == "" {
		return githubAsset{},
			fmt.Errorf(
				"unsupported platform: %s/%s",
				runtime.GOOS,
				runtime.GOARCH,
			)
	}

	for _, asset := range assets {
		if strings.HasPrefix(asset.Name, expectedPrefix) &&
			strings.HasSuffix(asset.Name, ".zip") {
			return asset, nil
		}
	}

	return githubAsset{},
		fmt.Errorf(
			"release does not contain platform ZIP starting with %q",
			expectedPrefix,
		)
}

func downloadUpdate(
	asset githubAsset,
) (string, error) {

	if asset.BrowserDownloadURL == "" {
		return "",
			errors.New(
				"release asset has no download URL",
			)
	}

	targetPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	updateDir :=
		filepath.Join(
			filepath.Dir(targetPath),
			".update",
		)

	if err := os.MkdirAll(updateDir, 0755); err != nil {
		return "", fmt.Errorf(
			"cannot create update directory: %w",
			err,
		)
	}

	stagedPath :=
		filepath.Join(
			updateDir,
			"new-"+asset.Name,
		)

	tempPath :=
		filepath.Join(
			updateDir,
			"."+asset.Name+".tmp",
		)

	_ = os.Remove(stagedPath)
	_ = os.Remove(tempPath)

	client :=
		&http.Client{
			Timeout: 5 * time.Minute,
		}

	resp, err :=
		client.Get(
			asset.BrowserDownloadURL,
		)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		return "",
			fmt.Errorf(
				"download returned HTTP %d",
				resp.StatusCode,
			)
	}

	file, err :=
		os.Create(tempPath)

	if err != nil {
		return "", err
	}

	_, copyErr :=
		io.Copy(
			file,
			resp.Body,
		)

	closeErr :=
		file.Close()

	if copyErr != nil {
		os.Remove(tempPath)
		return "", copyErr
	}

	if closeErr != nil {
		os.Remove(tempPath)
		return "", closeErr
	}

	if asset.Size > 0 {
		info, err :=
			os.Stat(tempPath)

		if err != nil {
			os.Remove(tempPath)
			return "", err
		}

		if info.Size() != asset.Size {
			os.Remove(tempPath)

			return "",
				fmt.Errorf(
					"downloaded size mismatch: got %d, expected %d",
					info.Size(),
					asset.Size,
				)
		}
	}

	if err :=
		verifyDownloadedFile(
			tempPath,
			asset.Digest,
		); err != nil {

		os.Remove(tempPath)
		return "", err
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempPath, 0755); err != nil {
			os.Remove(tempPath)
			return "", err
		}
	}

	if err := os.Rename(tempPath, stagedPath); err != nil {
		os.Remove(tempPath)
		return "",
			fmt.Errorf(
				"cannot stage update: %w",
				err,
			)
	}

	return stagedPath, nil
}

func verifyDownloadedFile(
	path string,
	expectedDigest string,
) error {

	expectedDigest =
		strings.TrimSpace(
			expectedDigest,
		)

	expectedDigest =
		strings.TrimPrefix(
			expectedDigest,
			"sha256:",
		)

	if expectedDigest == "" {
		return errors.New(
			"release asset has no SHA-256 digest",
		)
	}

	file, err :=
		os.Open(path)

	if err != nil {
		return err
	}

	defer file.Close()

	hash :=
		sha256.New()

	if _, err :=
		io.Copy(
			hash,
			file,
		); err != nil {

		return err
	}

	actualDigest :=
		hex.EncodeToString(
			hash.Sum(nil),
		)

	if !strings.EqualFold(
		actualDigest,
		expectedDigest,
	) {
		return fmt.Errorf(
			"SHA-256 mismatch: got %s, expected %s",
			actualDigest,
			expectedDigest,
		)
	}

	return nil
}

func startUpdateHelper(
	downloadedPath string,
	version string,
) error {
	targetPath, err := os.Executable()
	if err != nil {
		return err
	}

	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	updaterName :=
		"PC-Gear-Calculator-Updater"

	if runtime.GOOS == "windows" {
		updaterName += ".exe"
	}

	updaterPath :=
		filepath.Join(
			filepath.Dir(targetPath),
			".update",
			updaterName,
		)

	if _, err := os.Stat(updaterPath); err != nil {
		return fmt.Errorf(
			"bundled updater not found: %w",
			err,
		)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(updaterPath, 0755); err != nil {
			return err
		}
	}

	pid :=
		strconv.Itoa(
			os.Getpid(),
		)

	// Keep the original three-argument helper protocol. Existing installations
	// contain the v2.1.0 updater, which rejects the newer version argument.
	// The pending marker lets the updated app show its completion notice even
	// when the older helper performs the installation.
	writePendingUpdate(version)

	cmd :=
		exec.Command(
			updaterPath,
			downloadedPath,
			targetPath,
			pid,
		)

	cmd.Dir =
		filepath.Dir(
			targetPath,
		)

	if err := cmd.Start(); err != nil {
		if path := updatePendingPath(); path != "" {
			_ = os.Remove(path)
		}
		return fmt.Errorf(
			"unable to start updater: %w",
			err,
		)
	}

	return nil
}

type parsedVersion struct {
	major      int
	minor      int
	patch      int
	preRelease bool
	preID      string
	preNumber  int
}

func isNewerVersion(latest, current string) bool {
	latestVersion, latestOK := parseVersion(latest)
	currentVersion, currentOK := parseVersion(current)

	if !latestOK || !currentOK {
		return false
	}

	if latestVersion.major != currentVersion.major {
		return latestVersion.major > currentVersion.major
	}

	if latestVersion.minor != currentVersion.minor {
		return latestVersion.minor > currentVersion.minor
	}

	if latestVersion.patch != currentVersion.patch {
		return latestVersion.patch > currentVersion.patch
	}

	if latestVersion.preRelease != currentVersion.preRelease {
		return !latestVersion.preRelease
	}

	if !latestVersion.preRelease {
		return false
	}

	preRank := func(id string) int {
		switch strings.ToLower(id) {
		case "alpha":
			return 1
		case "beta":
			return 2
		case "rc":
			return 3
		default:
			return 0
		}
	}

	latestRank := preRank(latestVersion.preID)
	currentRank := preRank(currentVersion.preID)

	if latestRank != currentRank {
		return latestRank > currentRank
	}

	return latestVersion.preNumber > currentVersion.preNumber
}

func parseVersion(version string) (parsedVersion, bool) {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")

	parts := strings.SplitN(version, "-", 2)
	core := strings.Split(parts[0], ".")

	if len(core) != 3 {
		return parsedVersion{}, false
	}

	major, err := strconv.Atoi(core[0])
	if err != nil {
		return parsedVersion{}, false
	}

	minor, err := strconv.Atoi(core[1])
	if err != nil {
		return parsedVersion{}, false
	}

	patch, err := strconv.Atoi(core[2])
	if err != nil {
		return parsedVersion{}, false
	}

	result := parsedVersion{
		major: major,
		minor: minor,
		patch: patch,
	}

	if len(parts) == 1 {
		return result, true
	}

	pre := strings.Split(parts[1], ".")

	if len(pre) != 2 || pre[0] == "" {
		return parsedVersion{}, false
	}

	preNumber, err := strconv.Atoi(pre[1])
	if err != nil || preNumber < 0 {
		return parsedVersion{}, false
	}

	result.preRelease = true
	result.preID = pre[0]
	result.preNumber = preNumber

	return result, true
}
