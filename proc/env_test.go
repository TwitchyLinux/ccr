package proc

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	version "github.com/knqyf263/go-deb-version"
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

func TestFilePersistance(t *testing.T) {
	t.Parallel()
	e, err := NewEnv(false)
	if err != nil {
		t.Fatal(err)
	}

	o, s, _, err := e.RunBlocking("/tmp", "touch", "yee")
	if err != nil {
		e.Close()
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
		t.Logf("stdout = %q\nstderr = %q", string(o), string(s))
	}

	if _, err := os.Stat(filepath.Join(e.OverlayUpperPath(), "tmp", "yee")); err != nil {
		t.Errorf("could not stat /tmp/yee: %v", err)
	}
	if err := e.Close(); err != nil {
		t.Error(err)
	}
}

func TestTmpMasked(t *testing.T) {
	o, err := exec.Command("fuse-overlayfs", "--version").Output()
	if err != nil {
		t.SkipNow()
		return
	}
	for _, line := range strings.Split(string(o), "\n") {
		if strings.Contains(line, "fuse-overlayfs: version ") {
			s2 := strings.Split(line, " ")
			vers, err := version.NewVersion(s2[len(s2)-1])
			if err != nil {
				t.SkipNow()
				return
			}
			if minVers, _ := version.NewVersion("0.7"); vers.LessThan(minVers) {
				t.Skipf("fuse-overlayfs has version %v, need at least 0.7", vers)
				return
			}
		}
	}

	t.Parallel()
	e, err := NewEnv(true)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	out, _, _, err := e.RunBlocking("/tmp", "ls")
	if err != nil {
		t.Errorf("RunBlocking(%q) failed: %v", "echo", err)
	}
	if strings.Count(string(out), "\n") > 5 {
		t.Errorf("Far too many entries being listed in /tmp for the masking to have worked (#files = %d)", strings.Count(string(out), "\n"))
		t.Logf("Output: \n%s", string(out))
	}
}
