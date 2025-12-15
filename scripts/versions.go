// scripts/versions.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type row struct {
	Name        string
	Description string
	Module      string
	URL         string

	LatestVer  string
	LatestDate string
	LatestAgo  string
}

type goDLItem struct {
	Version string `json:"version"` // e.g. "go1.25.5"
	Stable  bool   `json:"stable"`
	Files   []any  `json:"files"`
}

type goListModule struct {
	Path    string    `json:"Path"`
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

type fw struct {
	Name        string
	Description string
	Module      string
	URL         string
}

var frameworks = []fw{
	{"net/http", "Go standard library HTTP server", "", "https://github.com/golang/go"},
	{"Chi", "Router built on net/http", "github.com/go-chi/chi/v5", "https://github.com/go-chi/chi"},
	{"Gin", "API focused HTTP framework", "github.com/gin-gonic/gin", "https://github.com/gin-gonic/gin"},
	{"Echo", "HTTP framework with error returns", "github.com/labstack/echo/v4", "https://github.com/labstack/echo"},
	{"Fiber", "fasthttp based framework", "github.com/gofiber/fiber/v2", "https://github.com/gofiber/fiber"},
	{"Mizu", "net/http aligned framework", "github.com/go-mizu/mizu", "https://github.com/go-mizu/mizu"},
}

func main() {
	var (
		timeout = flag.Duration("timeout", 10*time.Second, "HTTP timeout")
		verbose = flag.Bool("v", false, "print debug warnings to stderr")
	)
	flag.Parse()

	now := time.Now()
	ctx := context.Background()
	httpc := &http.Client{Timeout: *timeout}

	var rows []row

	// net/http: best is go.dev dl for version + go.dev release history for date.
	goVer, goDate, goAgo := "N/A", "N/A", "N/A"

	if v, err := latestStableGoVersion(ctx, httpc); err == nil {
		goVer = v
		if d, err := goReleaseDateFromHistory(ctx, httpc, v); err == nil {
			goDate = d
			if t, err := time.Parse("2006-01-02", d); err == nil {
				goAgo = fmt.Sprintf("%d days", daysAgo(t, now))
			}
			if *verbose {
				fmt.Fprintf(os.Stderr, "go: version=%s date=%s (from go.dev)\n", goVer, goDate)
			}
		} else {
			if *verbose {
				fmt.Fprintf(os.Stderr, "warn: go release date not found for %s: %v\n", v, err)
			}
		}
	} else {
		if *verbose {
			fmt.Fprintf(os.Stderr, "warn: go.dev dl failed: %v\n", err)
		}
	}

	// Fallback: local Go version (date unknown)
	if goVer == "N/A" || goVer == "" {
		if lv := localGoVersion(); lv != "" {
			goVer = lv
			if *verbose {
				fmt.Fprintf(os.Stderr, "warn: using local Go version fallback: %s (date unknown)\n", lv)
			}
		}
	}

	rows = append(rows, row{
		Name:        "net/http",
		Description: "Go standard library HTTP server",
		URL:         "https://github.com/golang/go",
		LatestVer:   goVer,
		LatestDate:  goDate,
		LatestAgo:   goAgo,
	})

	// frameworks: go list -m -json <module>@latest
	for _, f := range frameworks {
		if f.Module == "" {
			continue
		}
		ver, t, ok := latestViaGoList(f.Module)
		r := row{
			Name:        f.Name,
			Description: f.Description,
			Module:      f.Module,
			URL:         f.URL,
			LatestVer:   "N/A",
			LatestDate:  "N/A",
			LatestAgo:   "N/A",
		}
		if ok {
			r.LatestVer = ver
			if !t.IsZero() {
				r.LatestDate = t.Format("2006-01-02")
				r.LatestAgo = fmt.Sprintf("%d days", daysAgo(t, now))
			}
		}
		rows = append(rows, r)
	}

	printMarkdown(rows)
}

func latestStableGoVersion(ctx context.Context, httpc *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://go.dev/dl/?mode=json", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "go-fw-versions/1.6")
	req.Header.Set("Accept", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("go.dev dl: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var items []goDLItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return "", err
	}

	for _, it := range items {
		if it.Stable && it.Version != "" {
			return it.Version, nil
		}
	}

	// Fallback: first version if stable flags are missing.
	for _, it := range items {
		if it.Version != "" {
			return it.Version, nil
		}
	}

	return "", fmt.Errorf("no versions found in go.dev dl feed")
}

func goReleaseDateFromHistory(ctx context.Context, httpc *http.Client, version string) (string, error) {
	// Release history page includes lines like: "go1.25.5 (released 2025-12-15)"
	req, err := http.NewRequestWithContext(ctx, "GET", "https://go.dev/doc/devel/release", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "go-fw-versions/1.6")
	req.Header.Set("Accept", "text/html")

	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("go.dev release history: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	s := string(b)

	ver := regexp.QuoteMeta(version)
	re := regexp.MustCompile(`\b` + ver + `\b\s*\(released\s*([0-9]{4}-[0-9]{2}-[0-9]{2})\)`)
	m := re.FindStringSubmatch(s)
	if len(m) == 2 {
		return m[1], nil
	}

	return "", fmt.Errorf("release date not found for %s", version)
}

func localGoVersion() string {
	cmd := exec.Command("go", "env", "GOVERSION")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func latestViaGoList(module string) (ver string, t time.Time, ok bool) {
	cmd := exec.Command("go", "list", "-m", "-json", module+"@latest")
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return "", time.Time{}, false
	}

	var m goListModule
	if err := json.Unmarshal(out, &m); err != nil {
		return "", time.Time{}, false
	}
	if m.Version == "" {
		return "", time.Time{}, false
	}
	return m.Version, m.Time, true
}

func daysAgo(t, now time.Time) int {
	y1, m1, d1 := now.Date()
	y2, m2, d2 := t.Date()

	n0 := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)
	t0 := time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)

	if n0.Before(t0) {
		return 0
	}
	return int(n0.Sub(t0).Hours() / 24)
}

func printMarkdown(rows []row) {
	fmt.Println("## Frameworks covered")
	fmt.Println()
	fmt.Println("| Framework | Description | Latest version | Latest date | Released ago | GitHub |")
	fmt.Println("|---|---|---|---|---|---|")
	for _, r := range rows {
		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			esc(r.Name),
			esc(r.Description),
			esc(r.LatestVer),
			esc(r.LatestDate),
			esc(r.LatestAgo),
			r.URL,
		)
	}
}

func esc(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
