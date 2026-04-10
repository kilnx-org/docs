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

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
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

	// Run kilnx check
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer checkCancel()

	checkCmd := exec.CommandContext(checkCtx, "kilnx", "check", appFile)
	checkOut, _ := checkCmd.CombinedOutput()
	diagnostics := strings.TrimSpace(string(checkOut))

	// Find a free port and run the app
	port, err := freePort()
	if err != nil {
		writeJSON(w, 500, PlaygroundResponse{Error: "no free port"})
		return
	}

	dbPath := filepath.Join(tmpDir, "app.db")

	runCtx, runCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer runCancel()

	runCmd := exec.CommandContext(runCtx, "kilnx", "run", appFile)
	runCmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("DATABASE_URL=sqlite://%s", dbPath),
		"SECRET_KEY=playground-secret",
	)
	runCmd.Dir = tmpDir

	if err := runCmd.Start(); err != nil {
		writeJSON(w, 200, PlaygroundResponse{
			Diagnostics: diagnostics,
			Error:       "failed to start: " + err.Error(),
		})
		return
	}
	defer func() {
		runCmd.Process.Kill()
		runCmd.Wait()
	}()

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
