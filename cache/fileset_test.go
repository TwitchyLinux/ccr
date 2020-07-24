package cache

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var (
	missingHash = sha256.Sum256([]byte{0})
	createHash  = sha256.Sum256([]byte{1})
)

func TestFileset(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	if err := ioutil.WriteFile(filepath.Join(tmp, "something.txt"), []byte("something"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := NewCache(tmp)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure misses are fine.
	if _, _, _, err := c.FileInFileset(missingHash[:], "somefile.txt"); err != ErrCacheMiss {
		t.Errorf("FileInFileset(missingHash, somefile.txt) returned %v, want %v", err, ErrCacheMiss)
	}
	// Make sure misses still create the parent dir.
	if s, err := os.Stat(filepath.Join(tmp, fmt.Sprintf("%X", missingHash[:2]))); err == nil && s.IsDir() {
		t.Errorf("stat for missingHash dir failed: %v", err)
	}

	// Make sure we can create a fileset.
	pfs, err := c.CommitFileset(createHash[:])
	if err != nil {
		t.Fatalf("CommitFileset(%X) failed: %v", createHash, err)
	}
	// Attempt to add a file.
	s, _ := os.Stat(filepath.Join(tmp, "something.txt"))
	content, err := os.Open(filepath.Join(tmp, "something.txt"))
	if err != nil {
		t.Fatal(err)
	}
	pfs.AddFile("something.txt", s, content)
	if err := pfs.Close(); err != nil {
		t.Errorf("pfs.Close() failed: %v", err)
	}

	// Make sure we can read out that file.
	r, closer, mode, err := c.FileInFileset(createHash[:], "something.txt")
	if err != nil {
		t.Fatalf("FileInFileset(%X, %q) failed: %v", createHash[:], "something.txt", err)
	}
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		t.Fatal(err)
	}
	if want := []byte("something"); !bytes.Equal(b.Bytes(), want) {
		t.Errorf("bad content: %q, want %q", b.Bytes(), want)
	}
	if mode != s.Mode() {
		t.Errorf("mode = %v, want %v", mode, s.Mode())
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
}
