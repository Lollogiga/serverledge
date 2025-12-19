package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type InvocationRequest struct {
	Command      []string               `json:"Command"`
	Params       map[string]interface{} `json:"Params"`
	Handler      string                 `json:"Handler"`
	HandlerDir   string                 `json:"HandlerDir"`
	ReturnOutput bool                   `json:"ReturnOutput"`
}

type InvocationResult struct {
	Success bool   `json:"Success"`
	Result  string `json:"Result"`
	Output  string `json:"Output"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/invoke", invokeHandler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Native Executor listening on :8080")
	log.Fatal(srv.ListenAndServe())
}

func invokeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, InvocationResult{Success: false, Result: "", Output: "invalid json: " + err.Error()})
		return
	}

	// 1) Decide command to run:
	// Prefer req.Command if provided (as per spec). Otherwise fallback to Handler/HandlerDir.
	cmdSlice := buildCommand(req)
	if len(cmdSlice) == 0 {
		writeJSON(w, InvocationResult{Success: false, Result: "", Output: "empty command"})
		return
	}

	// 2) Append arguments derived from Params (generic, deterministic order).
	args := paramsToArgs(req.Params)
	full := append(cmdSlice, args...)

	// 3) Execute and capture stdout/stderr
	// (You can tune timeout if you want; keep it simple now.)
	c := exec.Command(full[0], full[1:]...)

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = &stdoutBuf
	c.Stderr = &stderrBuf

	err := c.Run()

	stdout := strings.TrimSpace(stdoutBuf.String())
	stderr := strings.TrimSpace(stderrBuf.String())

	success := (err == nil)

	// Convention:
	// - Result is stdout (trimmed)
	// - Output is stdout+stderr only if ReturnOutput is true
	out := ""
	if req.ReturnOutput {
		if stdout != "" {
			out += stdout
		}
		if stderr != "" {
			if out != "" {
				out += "\n"
			}
			out += stderr
		}
	}

	res := InvocationResult{
		Success: success,
		Result:  stdout, // main "return"
		Output:  out,
	}

	// If error, include it in Output (useful debugging)
	if !success && req.ReturnOutput {
		if res.Output != "" {
			res.Output += "\n"
		}
		res.Output += "exec error: " + err.Error()
	}

	writeJSON(w, res)
}

func buildCommand(req InvocationRequest) []string {
	if len(req.Command) > 0 {
		// Spec says Command is runtime-dependent and may be used as-is.
		return req.Command
	}

	// Fallback: HandlerDir + "/" + Handler (common pattern)
	if req.HandlerDir != "" && req.Handler != "" {
		path := strings.TrimRight(req.HandlerDir, "/") + "/" + strings.TrimLeft(req.Handler, "/")
		return []string{path}
	}

	// Last chance: use Handler alone
	if req.Handler != "" {
		return []string{req.Handler}
	}
	return nil
}

func paramsToArgs(params map[string]interface{}) []string {
	if len(params) == 0 {
		return nil
	}

	// Deterministic order for reproducibility
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys))
	for _, k := range keys {
		v := params[k]
		// Convention: pass as --key=value
		// (Works well for multiple params. For single param you can still parse.)
		args = append(args, "--"+k+"="+toString(v))
	}
	return args
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func writeJSON(w http.ResponseWriter, res InvocationResult) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
