package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/getmiroir/trading-service/internal/version"
)

type AppUpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl"`
}

const manifestTimeout = 10 * time.Second

var manifestClient = &http.Client{Timeout: manifestTimeout}

type manifestRelease struct {
	Version   string            `json:"version"`
	Platforms map[string]string `json:"platforms"`
}

// CheckAppUpdate fetches the release manifest and reports whether a newer
// desktop app build is published for this platform. Every failure is logged and
// reported as "no update available", so the GUI stays silent when the manifest
// is unreachable.
func (a *App) CheckAppUpdate() AppUpdateInfo {
	cfg := a.Settings.Settings().CopyTrading.Update
	info := AppUpdateInfo{CurrentVersion: version.Version, DownloadURL: cfg.DownloadURL}

	releases, err := fetchManifest(a.ctx, cfg.ManifestURL)
	if err != nil {
		a.logger.Warn("App update check failed", "url", cfg.ManifestURL, "error", err)
		return info
	}

	latest, found := latestRelease(releases, manifestPlatformKey())
	if !found {
		a.logger.Warn("App update check found no published release",
			"url", cfg.ManifestURL, "platform", version.Platform)
		return info
	}
	info.LatestVersion = latest

	if version.IsDevelop(info.CurrentVersion) {
		info.UpdateAvailable = true
	} else {
		cmp, err := version.Compare(latest, info.CurrentVersion)
		if err != nil {
			a.logger.Warn("App update compare failed",
				"latest", latest, "current", info.CurrentVersion, "error", err)
			return info
		}
		info.UpdateAvailable = cmp > 0
	}

	a.logger.Info("App update check",
		"current", info.CurrentVersion, "latest", info.LatestVersion, "available", info.UpdateAvailable)
	return info
}

func fetchManifest(ctx context.Context, url string) ([]manifestRelease, error) {
	if url == "" {
		return nil, fmt.Errorf("no manifest URL configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := manifestClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var releases []manifestRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// latestRelease returns the highest version in the manifest that ships a build
// for platformKey. An empty platformKey accepts every release. Entries whose
// version cannot be parsed are skipped, and manifest ordering is not trusted.
func latestRelease(releases []manifestRelease, platformKey string) (string, bool) {
	latest := ""
	for _, release := range releases {
		if platformKey != "" && release.Platforms[platformKey] == "" {
			continue
		}
		if _, _, err := version.Parse(release.Version); err != nil {
			continue
		}
		if latest != "" {
			cmp, err := version.Compare(release.Version, latest)
			if err != nil || cmp <= 0 {
				continue
			}
		}
		latest = release.Version
	}
	return latest, latest != ""
}

// manifestPlatformKey maps the build-time platform (e.g. "darwin/universal") to
// the key used in the manifest. An empty result means the platform is unknown,
// in which case releases must not be filtered on it.
func manifestPlatformKey() string {
	goos, _, _ := strings.Cut(version.Platform, "/")
	switch goos {
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return ""
	}
}
