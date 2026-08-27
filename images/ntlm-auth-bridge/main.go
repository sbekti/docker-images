package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const maxRequestBytes = 4096

type authRequest struct {
	Username     string `json:"username"`
	Domain       string `json:"domain"`
	Challenge    string `json:"challenge"`
	NTResponse   string `json:"nt_response"`
	RequestNTKey bool   `json:"request_nt_key"`
}

type authResponse struct {
	Authenticated bool   `json:"authenticated"`
	NTKey         string `json:"nt_key,omitempty"`
}

type server struct {
	ntlmAuthPath string
	timeout      time.Duration
}

func main() {
	port := flag.String("port", "9555", "port to listen on")
	ntlmAuthPath := flag.String("ntlm-auth-path", "ntlm_auth", "path to ntlm_auth")
	timeout := flag.Duration("timeout", 2*time.Second, "maximum ntlm_auth runtime")
	flag.Parse()

	s := &server{ntlmAuthPath: *ntlmAuthPath, timeout: *timeout}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", s.handleAuth)

	httpServer := &http.Server{
		Addr:              ":" + *port,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       4 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("listening on %s", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (s *server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request authRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if !validRequest(request) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	args := []string{
		"--allow-mschapv2",
		"--username=" + request.Username,
		"--challenge=" + request.Challenge,
		"--nt-response=" + request.NTResponse,
		"--request-nt-key",
	}
	if request.Domain != "" {
		args = append(args, "--domain="+request.Domain)
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, s.ntlmAuthPath, args...).CombinedOutput()
	if ctx.Err() != nil {
		log.Printf("ntlm_auth timed out")
		http.Error(w, "authentication service unavailable", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			writeJSON(w, http.StatusOK, authResponse{})
			return
		}

		log.Printf("cannot run ntlm_auth: %v", err)
		http.Error(w, "authentication service unavailable", http.StatusServiceUnavailable)
		return
	}

	ntKey := findNTKey(string(output))
	if !validHex(ntKey, 32) {
		log.Printf("ntlm_auth succeeded without a valid NT key")
		http.Error(w, "invalid authentication response", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Authenticated: true, NTKey: ntKey})
}

func validRequest(request authRequest) bool {
	return request.Username != "" && len(request.Username) <= 256 &&
		len(request.Domain) <= 256 && request.RequestNTKey &&
		validHex(request.Challenge, 16) && validHex(request.NTResponse, 48)
}

func validHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func findNTKey(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if value, found := strings.CutPrefix(line, "NT_KEY: "); found {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, response authResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("cannot write response: %v", err)
	}
}
