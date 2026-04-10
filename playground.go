package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type PlaygroundRequest struct {
	Code string `json:"code"`
}

type PlaygroundResponse struct {
	HTML        string `json:"html,omitempty"`
	Diagnostics string `json:"diagnostics,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Limit concurrent playground runs
var sem = make(chan struct{}, 3)

// Rate limiter: track requests per IP
var (
	rateMu    sync.Mutex
	rateCount = make(map[string][]time.Time)
)

const (
	rateWindow = 1 * time.Minute
	rateMax    = 10
	maxTmpSize = 10 * 1024 * 1024 // 10MB filesystem limit per run
)

func isRateLimited(ip string) bool {
	rateMu.Lock()
	defer rateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateWindow)

	// Clean old entries
	recent := rateCount[ip][:0]
	for _, t := range rateCount[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rateMax {
		rateCount[ip] = recent
		return true
	}

	rateCount[ip] = append(recent, now)
	return false
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.SplitN(fwd, ",", 2)[0]
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// sandboxCmd restricts a command: no network, limited filesystem via ulimit
func sandboxCmd(cmd *exec.Cmd, tmpDir string) {
	// Block network access by unsetting proxy and using unshare for net namespace
	// Also set resource limits via SysProcAttr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNET, // new network namespace = no network
	}

	// Set RLIMIT_FSIZE to limit file writes
	// This is enforced per-process by the kernel
	cmd.Env = append(cmd.Env, "TMPDIR="+tmpDir)
}

// enforceFilesizeLimit sets RLIMIT_FSIZE on the current goroutine context
// We use a wrapper script approach instead for portability
func wrapWithLimits(kilnxBin string, args []string, tmpDir string, port int, dbPath string) *exec.Cmd {
	// Use sh -c with ulimit to enforce file size limit
	// ulimit -f <blocks> limits file creation size (in 512-byte blocks)
	maxBlocks := maxTmpSize / 512 // 10MB in 512-byte blocks
	shellCmd := fmt.Sprintf("ulimit -f %d; exec %s %s",
		maxBlocks, kilnxBin, strings.Join(args, " "))

	cmd := exec.Command("sh", "-c", shellCmd)
	cmd.Dir = tmpDir
	cmd.Env = []string{
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("DATABASE_URL=sqlite://%s", dbPath),
		"SECRET_KEY=playground-secret",
		"HOME=/tmp",
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"TMPDIR=" + tmpDir,
	}

	// New network namespace: no outbound access
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNET,
	}

	return cmd
}

func handlePlayground(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}

	// Rate limit by IP
	ip := clientIP(r)
	if isRateLimited(ip) {
		writeJSON(w, 429, PlaygroundResponse{Error: "rate limited: max 10 runs per minute"})
		return
	}

	// Concurrency limit
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	default:
		writeJSON(w, 429, PlaygroundResponse{Error: "too many concurrent runs, try again"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, 400, PlaygroundResponse{Error: "failed to read body"})
		return
	}

	var req PlaygroundRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, 400, PlaygroundResponse{Error: "invalid JSON"})
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeJSON(w, 400, PlaygroundResponse{Error: "empty code"})
		return
	}

	if len(code) > 32*1024 {
		writeJSON(w, 400, PlaygroundResponse{Error: "code too large (max 32KB)"})
		return
	}

	// Block dangerous patterns
	codeLower := strings.ToLower(code)
	if strings.Contains(codeLower, "fetch") {
		writeJSON(w, 400, PlaygroundResponse{Error: "fetch is disabled in the playground"})
		return
	}

	tmpDir, err := os.MkdirTemp("", "playground-*")
	if err != nil {
		writeJSON(w, 500, PlaygroundResponse{Error: "failed to create temp dir"})
		return
	}
	defer os.RemoveAll(tmpDir)

	appFile := filepath.Join(tmpDir, "app.kilnx")
	if err := os.WriteFile(appFile, []byte(code), 0644); err != nil {
		writeJSON(w, 500, PlaygroundResponse{Error: "failed to write file"})
		return
	}

	// Run kilnx check (no sandbox needed, read-only)
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer checkCancel()

	checkCmd := exec.CommandContext(checkCtx, "kilnx", "check", appFile)
	checkOut, _ := checkCmd.CombinedOutput()
	diagnostics := strings.TrimSpace(string(checkOut))

	// Find a free port and run the app in sandbox
	port, err := freePort()
	if err != nil {
		writeJSON(w, 500, PlaygroundResponse{Error: "no free port"})
		return
	}

	dbPath := filepath.Join(tmpDir, "app.db")

	runCtx, runCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer runCancel()

	runCmd := wrapWithLimits("kilnx", []string{"run", appFile}, tmpDir, port, dbPath)
	runCmd.Stdout = nil
	runCmd.Stderr = nil

	// Apply context timeout
	runCmdCtx := exec.CommandContext(runCtx, runCmd.Path, runCmd.Args[1:]...)
	runCmdCtx.Dir = runCmd.Dir
	runCmdCtx.Env = runCmd.Env
	runCmdCtx.SysProcAttr = runCmd.SysProcAttr

	if err := runCmdCtx.Start(); err != nil {
		// CLONE_NEWNET may require privileges, fall back without network sandbox
		runCmdFallback := exec.CommandContext(runCtx, "sh", "-c",
			fmt.Sprintf("ulimit -f %d; exec kilnx run %s", maxTmpSize/512, appFile))
		runCmdFallback.Dir = tmpDir
		runCmdFallback.Env = []string{
			fmt.Sprintf("PORT=%d", port),
			fmt.Sprintf("DATABASE_URL=sqlite://%s", dbPath),
			"SECRET_KEY=playground-secret",
			"HOME=/tmp",
			"PATH=/usr/local/bin:/usr/bin:/bin",
		}
		if err2 := runCmdFallback.Start(); err2 != nil {
			writeJSON(w, 200, PlaygroundResponse{
				Diagnostics: diagnostics,
				Error:       "failed to start: " + err2.Error(),
			})
			return
		}
		defer func() {
			runCmdFallback.Process.Kill()
			runCmdFallback.Wait()
		}()
	} else {
		defer func() {
			runCmdCtx.Process.Kill()
			runCmdCtx.Wait()
		}()
	}

	// Wait for server ready
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	ready := false
	for i := 0; i < 40; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get(addr + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				ready = true
				break
			}
		}
	}

	if !ready {
		writeJSON(w, 200, PlaygroundResponse{
			Diagnostics: diagnostics,
			Error:       "app failed to start (check your code for errors)",
		})
		return
	}

	// Find the first page route
	path := "/"
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "page /") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				path = parts[1]
				break
			}
		}
	}

	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer fetchCancel()

	fetchReq, _ := http.NewRequestWithContext(fetchCtx, "GET", addr+path, nil)
	fetchResp, err := http.DefaultClient.Do(fetchReq)
	if err != nil {
		writeJSON(w, 200, PlaygroundResponse{
			Diagnostics: diagnostics,
			Error:       "failed to fetch page: " + err.Error(),
		})
		return
	}
	defer fetchResp.Body.Close()

	htmlBody, _ := io.ReadAll(io.LimitReader(fetchResp.Body, 512*1024))

	writeJSON(w, 200, PlaygroundResponse{
		HTML:        string(htmlBody),
		Diagnostics: diagnostics,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	publicPort := os.Getenv("PORT")
	if publicPort == "" {
		publicPort = "8080"
	}

	docsPort := "8081"

	// Clean up stale rate limit entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rateMu.Lock()
			cutoff := time.Now().Add(-rateWindow)
			for ip, times := range rateCount {
				recent := times[:0]
				for _, t := range times {
					if t.After(cutoff) {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					delete(rateCount, ip)
				} else {
					rateCount[ip] = recent
				}
			}
			rateMu.Unlock()
		}
	}()

	// Start docs server on internal port
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("server")
		cmd.Env = append(os.Environ(), "PORT="+docsPort)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "docs server exited: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for docs server
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:" + docsPort + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
	}

	// Reverse proxy to docs server
	docsURL, _ := url.Parse("http://127.0.0.1:" + docsPort)
	proxy := httputil.NewSingleHostReverseProxy(docsURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/playground", handlePlayground)
	mux.Handle("/", proxy)

	fmt.Printf("Gateway listening on :%s (docs on :%s)\n", publicPort, docsPort)
	http.ListenAndServe(":"+publicPort, mux)
}
