// Package sandbox runs untrusted playground code in short-lived, locked-down
// Docker containers. Containers have no direct internet: their only route out is
// the egress proxy, which allowlists catalog hosts and blocks internal ranges.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Request struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type Result struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	TimedOut   bool   `json:"timed_out"`
	Truncated  bool   `json:"truncated"`
}

type ImageSpec struct {
	Image string
	Cmd   []string
}

type Config struct {
	DockerBin string
	Network   string // internal docker network that only reaches the egress proxy
	ProxyURL  string // e.g. http://egress:3128
	Timeout   time.Duration
	MemoryMB  int
	CPUs      string
	PIDs      int
	MaxCode   int
	MaxOutput int
	MaxJobs   int
	Images    map[string]ImageSpec
}

func DefaultConfig() Config {
	return Config{
		DockerBin: "docker", Network: "apiscope-sandbox", ProxyURL: "http://egress:3128",
		Timeout: 10 * time.Second, MemoryMB: 128, CPUs: "0.5", PIDs: 64,
		MaxCode: 20 << 10, MaxOutput: 64 << 10, MaxJobs: 4,
		Images: map[string]ImageSpec{
			"javascript": {Image: "node:24-alpine", Cmd: []string{"node", "-"}},
			"python":     {Image: "python:3.12-alpine", Cmd: []string{"python", "-"}},
		},
	}
}

var (
	ErrUnsupported = errors.New("language is not supported yet")
	ErrBusy        = errors.New("runner is busy, try again shortly")
	ErrTooLarge    = errors.New("code is too large")
)

// Args builds the docker command line. It is pure so it can be unit tested
// without Docker, and it is the single place the isolation policy lives.
func (c Config) Args(name, lang string) ([]string, error) {
	spec, ok := c.Images[lang]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupported, lang)
	}
	args := []string{
		"run", "--rm", "-i", "--name", name,
		"--network", c.Network,
		"--memory", fmt.Sprintf("%dm", c.MemoryMB), "--memory-swap", fmt.Sprintf("%dm", c.MemoryMB),
		"--cpus", c.CPUs, "--pids-limit", fmt.Sprint(c.PIDs),
		"--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid,size=16m",
		"--cap-drop", "ALL", "--security-opt", "no-new-privileges",
		"--user", "65534:65534",
		"-e", "HTTP_PROXY=" + c.ProxyURL, "-e", "HTTPS_PROXY=" + c.ProxyURL,
		"-e", "http_proxy=" + c.ProxyURL, "-e", "https_proxy=" + c.ProxyURL,
		"-e", "NODE_USE_ENV_PROXY=1",
		spec.Image,
	}
	return append(args, spec.Cmd...), nil
}

type limitedBuffer struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	room := l.max - l.buf.Len()
	if room <= 0 {
		l.truncated = true
		return len(p), nil
	}
	if len(p) > room {
		l.buf.Write(p[:room])
		l.truncated = true
		return len(p), nil
	}
	return l.buf.Write(p)
}

type Runner struct {
	Config Config
	sem    chan struct{}
	mu     sync.Mutex
	seq    uint64
}

func NewRunner(c Config) *Runner {
	return &Runner{Config: c, sem: make(chan struct{}, c.MaxJobs)}
}

func (r *Runner) nextName() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return fmt.Sprintf("apiscope-run-%d-%d", time.Now().UnixNano(), r.seq)
}

func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	req.Language = strings.ToLower(req.Language)
	if len(req.Code) > r.Config.MaxCode {
		return Result{}, ErrTooLarge
	}
	name := r.nextName()
	args, err := r.Config.Args(name, req.Language)
	if err != nil {
		return Result{}, err
	}
	select {
	case r.sem <- struct{}{}:
		defer func() { <-r.sem }()
	default:
		return Result{}, ErrBusy
	}

	cctx, cancel := context.WithTimeout(ctx, r.Config.Timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, r.Config.DockerBin, args...)
	cmd.Stdin = strings.NewReader(req.Code)
	out := &limitedBuffer{max: r.Config.MaxOutput}
	errb := &limitedBuffer{max: r.Config.MaxOutput}
	cmd.Stdout, cmd.Stderr = out, errb

	start := time.Now()
	runErr := cmd.Run()
	res := Result{
		Stdout: out.buf.String(), Stderr: errb.buf.String(),
		DurationMS: time.Since(start).Milliseconds(), Truncated: out.truncated || errb.truncated,
	}
	if cctx.Err() == context.DeadlineExceeded {
		res.TimedOut, res.ExitCode = true, 124
		// Killing the docker CLI does not stop the container; kill it by name.
		kctx, kc := context.WithTimeout(context.Background(), 5*time.Second)
		defer kc()
		_ = exec.CommandContext(kctx, r.Config.DockerBin, "kill", name).Run()
		return res, nil
	}
	var ee *exec.ExitError
	switch {
	case runErr == nil:
	case errors.As(runErr, &ee):
		res.ExitCode = ee.ExitCode()
	default:
		return res, fmt.Errorf("docker unavailable: %w", runErr)
	}
	return res, nil
}
