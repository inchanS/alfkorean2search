// Package update provides self-updating via GitHub releases, replacing
// alfred-pyworkflow's update_settings + `workflow:update` magic.
//
// Flow:
//   - MaybeShow: if a newer release is already known, add a notice item; and at
//     most once per checkFrequency, spawn a detached background check.
//   - RunCheck (subcommand "_update-check"): query the latest release, record
//     whether it is newer and the .alfredworkflow asset URL.
//   - Install (magic query "workflow:update"): download that asset and open it
//     so Alfred installs the new version.
package update

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/inchanS/AlfKorean2Search/internal/alfred"
	"github.com/inchanS/AlfKorean2Search/internal/cache"
	"github.com/inchanS/AlfKorean2Search/internal/httpx"
)

const (
	// Magic is the query that triggers download & install, kept identical to the
	// previous library behaviour so the UX is unchanged.
	Magic = "workflow:update"

	githubSlug     = "inchanS/AlfKorean2Search"
	checkFrequency = 7 * 24 * time.Hour // weekly

	infoKey  = "__update_info"
	stampKey = "__update_check"
)

type info struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	URL       string `json:"url"`
}

func currentVersion() string { return os.Getenv("alfred_workflow_version") }

func readInfo() (info, bool) {
	data, ok := cache.Read(infoKey, 0)
	if !ok {
		return info{}, false
	}
	var inf info
	if err := json.Unmarshal(data, &inf); err != nil {
		return info{}, false
	}
	return inf, true
}

func writeInfo(inf info) {
	if data, err := json.Marshal(inf); err == nil {
		_ = cache.Write(infoKey, data)
	}
}

// MaybeShow adds the update notice (when a newer version is known) and triggers
// a weekly background check.
func MaybeShow(fb *alfred.Feedback) {
	if inf, ok := readInfo(); ok && shouldNotify(inf, currentVersion()) {
		fb.Add(alfred.ItemOpts{
			Title:        "AlfKorean2Search의 새 버전이 있습니다!",
			Subtitle:     "Enter를 눌러 업데이트를 설치합니다",
			Autocomplete: Magic,
			Valid:        false,
		})
	}
	maybeBackgroundCheck()
}

// shouldNotify reports whether a cached release should surface an update notice
// for the currently installed version. It recomputes from inf.Version against
// current rather than trusting the stored Available flag: the cache directory
// is keyed by bundleid and survives workflow updates, so it can hold a stale
// Available:true written while an older version was installed. Recomputing here
// makes the notice clear immediately after the user updates, without waiting
// for the next (weekly) background check to refresh the cache.
func shouldNotify(inf info, current string) bool {
	return inf.Version != "" && isNewer(inf.Version, current)
}

// maybeBackgroundCheck spawns a detached "_update-check" process at most once
// per checkFrequency, tracked by the mtime of a stamp file.
func maybeBackgroundCheck() {
	if _, fresh := cache.Read(stampKey, checkFrequency); fresh {
		return
	}
	_ = cache.Write(stampKey, []byte(time.Now().Format(time.RFC3339)))

	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "_update-check")
	// Detach: own process group, no inherited stdio, so Alfred does not wait on
	// it after the foreground feedback is written.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	cmd.Env = os.Environ()
	_ = cmd.Start()
	// Intentionally not waited on; the process outlives this one.
}

// RunCheck queries the latest release and records the result. Invoked by the
// detached background process.
func RunCheck() {
	inf, err := fetchLatest()
	if err != nil {
		return
	}
	writeInfo(inf)
}

// fetchLatest asks the GitHub API for the latest release and locates the
// .alfredworkflow asset.
func fetchLatest() (info, error) {
	body, err := httpx.Get(
		"https://api.github.com/repos/"+githubSlug+"/releases/latest",
		nil,
		map[string]string{"Accept": "application/vnd.github+json"},
	)
	if err != nil {
		return info{}, err
	}

	var rel struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &rel); err != nil {
		return info{}, err
	}

	var assetURL string
	for _, a := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".alfredworkflow") {
			assetURL = a.URL
			break
		}
	}

	return info{
		Available: isNewer(rel.TagName, currentVersion()),
		Version:   rel.TagName,
		URL:       assetURL,
	}, nil
}

// Install downloads the known .alfredworkflow asset and opens it so Alfred
// installs the update. Invoked when the query equals Magic.
func Install(fb *alfred.Feedback) {
	inf, ok := readInfo()
	if !ok || inf.URL == "" {
		if fresh, err := fetchLatest(); err == nil {
			inf = fresh
			writeInfo(inf)
		}
	}
	if inf.URL == "" {
		fb.Add(alfred.ItemOpts{Title: "설치할 업데이트가 없습니다", Valid: false})
		fb.Send()
		return
	}

	path, err := download(inf.URL)
	if err != nil {
		fb.Add(alfred.ItemOpts{
			Title:    "업데이트 다운로드에 실패했습니다",
			Subtitle: err.Error(),
			Valid:    false,
		})
		fb.Send()
		return
	}

	_ = exec.Command("/usr/bin/open", path).Run()
	fb.Add(alfred.ItemOpts{
		Title:    "업데이트 " + inf.Version + " 설치 중 …",
		Subtitle: "Alfred가 새 버전 가져오기를 안내합니다",
		Valid:    false,
	})
	fb.Send()
}

// download saves the asset to a temp file and returns its path.
func download(url string) (string, error) {
	body, err := httpx.Get(url, nil, nil)
	if err != nil {
		return "", err
	}
	dst := filepath.Join(os.TempDir(), "AlfKorean2Search-update.alfredworkflow")
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		return "", err
	}
	return dst, nil
}

// isNewer reports whether the remote version tag is greater than the current
// one, using a simple dotted-numeric comparison after stripping a leading 'v'.
func isNewer(remote, current string) bool {
	rp := parseVersion(remote)
	cp := parseVersion(current)
	n := len(rp)
	if len(cp) > n {
		n = len(cp)
	}
	for i := 0; i < n; i++ {
		var r, c int
		if i < len(rp) {
			r = rp[i]
		}
		if i < len(cp) {
			c = cp[i]
		}
		if r != c {
			return r > c
		}
	}
	return false
}

func parseVersion(v string) []int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		// Stop at any non-numeric suffix (e.g. "1.2.0-beta").
		num := p
		if idx := strings.IndexFunc(p, func(r rune) bool { return r < '0' || r > '9' }); idx >= 0 {
			num = p[:idx]
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
