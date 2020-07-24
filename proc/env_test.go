package proc

import (
	"bytes"
	"os"
	"testing"
)

func TestRunEnv(t *testing.T) {
	t.Parallel()
	e, err := NewEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Close(); err != nil {
		t.Error(err)
	}
}

func TestRunBlocking(t *testing.T) {
	t.Parallel()
	e, err := NewEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	out, _, _, err := e.RunBlocking(wd, "echo", "mmmyay", "1")
	if err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}
	if want := []byte("mmmyay 1\n"); !bytes.Equal(out, want) {
		t.Errorf("Output = %q, want %q", string(out), string(want))
	}
	out, _, _, err = e.RunBlocking(wd, "echo", "mmmyay", "2")
	if err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}
	if want := []byte("mmmyay 2\n"); !bytes.Equal(out, want) {
		t.Errorf("Output = %q, want %q", string(out), string(want))
	}

	_, _, code, err := e.RunBlocking(wd, "bash", "-c", "exit 12")
	if err != nil && err.Error() != "exit status 12" {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "exit", err)
	}
	if code != 12 {
		t.Errorf("code = %d, want %d", code, 12)
	}

	out, _, _, err = e.RunBlocking(wd, "pwd")
	if err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "pwd", err)
	}
	if want := []byte(wd + "\n"); !bytes.Equal(out, want) {
		t.Errorf("Output = %q, want %q", string(out), string(want))
	}

	if err := e.Close(); err != nil {
		t.Error(err)
	}
}

func TestRunStreaming(t *testing.T) {
	t.Parallel()
	e, err := NewEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	id, err := e.RunStreaming("/", &stdout, &stderr, "echo", "yeow")
	if err != nil {
		t.Fatalf("RunStreaming() failed: %v", err)
	}
	id2, err2 := e.RunStreaming("/", &stdout, &stderr, "bash", "-c", "sleep 0.1 && >&2 echo noot")
	if err2 != nil {
		t.Fatalf("RunStreaming() failed: %v", err2)
	}

	if err := e.WaitStreaming(id); err != nil {
		t.Errorf("WaitStreaming() failed: %v", err)
	}
	if err := e.WaitStreaming(id2); err != nil {
		t.Errorf("WaitStreaming() failed: %v", err)
	}

	if want := []byte("yeow\n"); !bytes.Equal(want, stdout.Bytes()) {
		t.Errorf("stdout = %q, want %q", string(stdout.Bytes()), string(want))
	}
	if want := []byte("noot\n"); !bytes.Equal(want, stderr.Bytes()) {
		t.Errorf("stderr = %q, want %q", string(stderr.Bytes()), string(want))
	}

	if err := e.Close(); err != nil {
		t.Error(err)
	}
}
