package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArgsEnforceIsolation(t *testing.T) {
	args, err := DefaultConfig().Args("n1", "javascript")
	if err != nil {
		t.Fatal(err)
	}
	joined := " " + strings.Join(args, " ") + " "
	for _, must := range []string{
		" --read-only ", " --cap-drop ALL ", " no-new-privileges ", " --user 65534:65534 ",
		" --network apiscope-sandbox ", " --memory 128m ", " --pids-limit 64 ", " --rm ",
	} {
		if !strings.Contains(joined, must) {
			t.Errorf("missing %q in %v", must, args)
		}
	}
	if strings.Contains(joined, "--privileged") || strings.Contains(joined, "-v ") {
		t.Error("sandbox must not be privileged or mount volumes")
	}
	if args[len(args)-2] != "node" {
		t.Errorf("unexpected command tail %v", args[len(args)-2:])
	}
	if _, err := DefaultConfig().Args("n", "cobol"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("expected ErrUnsupported, got %v", err)
	}
}

func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{max: 5}
	b.Write([]byte("abc"))
	b.Write([]byte("defgh"))
	if b.buf.String() != "abcde" || !b.truncated {
		t.Fatalf("got %q truncated=%v", b.buf.String(), b.truncated)
	}
}

// fakeDocker lets us exercise Run end to end without Docker installed.
func fakeDocker(t *testing.T, script string) string {
	p := filepath.Join(t.TempDir(), "docker")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunCapturesOutputAndExitCode(t *testing.T) {
	c := DefaultConfig()
	c.DockerBin = fakeDocker(t, `cat; echo oops >&2; exit 3`)
	res, err := NewRunner(c).Run(context.Background(), Request{Language: "python", Code: "print(1)"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "print(1)" || !strings.Contains(res.Stderr, "oops") || res.ExitCode != 3 {
		t.Fatalf("unexpected %+v", res)
	}
}

func TestRunTimeout(t *testing.T) {
	c := DefaultConfig()
	c.Timeout = 200 * time.Millisecond
	c.DockerBin = fakeDocker(t, `[ "$1" = "kill" ] && exit 0; exec sleep 5`)
	res, err := NewRunner(c).Run(context.Background(), Request{Language: "javascript", Code: "x"})
	if err != nil || !res.TimedOut || res.ExitCode != 124 {
		t.Fatalf("expected timeout, got %+v err=%v", res, err)
	}
}

func TestRunRejectsOversizeAndBusy(t *testing.T) {
	c := DefaultConfig()
	c.MaxCode = 4
	r := NewRunner(c)
	if _, err := r.Run(context.Background(), Request{Language: "python", Code: "12345"}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
	c2 := DefaultConfig()
	c2.MaxJobs = 1
	r2 := NewRunner(c2)
	r2.sem <- struct{}{}
	if _, err := r2.Run(context.Background(), Request{Language: "python", Code: "x"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("got %v", err)
	}
}
