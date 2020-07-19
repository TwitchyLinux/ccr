package proc

import (
	"bytes"
	"testing"
)

func TestRunEnv(t *testing.T) {
	e, err := NewEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Close(); err != nil {
		t.Error(err)
	}
}

func TestRunBlocking(t *testing.T) {
	e, err := NewEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	out, _, _, err := e.RunBlocking("echo", "mmmyay", "1")
	if err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}
	if want := []byte("mmmyay 1\n"); !bytes.Equal(out, want) {
		t.Errorf("Output = %q, want %q", string(out), string(want))
	}
	out, _, _, err = e.RunBlocking("echo", "mmmyay", "2")
	if err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}
	if want := []byte("mmmyay 2\n"); !bytes.Equal(out, want) {
		t.Errorf("Output = %q, want %q", string(out), string(want))
	}

	_, _, code, err := e.RunBlocking("bash", "-c", "exit 12")
	if err != nil && err.Error() != "exit status 12" {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "exit", err)
	}
	if code != 12 {
		t.Errorf("code = %d, want %d", code, 12)
	}

	if err := e.Close(); err != nil {
		t.Error(err)
	}
}
