package proc

import (
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

	if err := e.RunBlocking("echo", "mmmyay", "1"); err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}
	if err := e.RunBlocking("echo", "mmmyay", "2"); err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}

	if err := e.Close(); err != nil {
		t.Error(err)
	}
}
