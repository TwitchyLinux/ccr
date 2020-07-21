package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestCacheByName(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	c, err := NewCache(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if f, err := c.ByName("doesnt-exist"); err != ErrCacheMiss || f != nil {
		t.Errorf("ByName(%q) returned (%v,%v), want (%v,%v)", "doesnt-exist", f, err, nil, ErrCacheMiss)
	}
	if err := ioutil.WriteFile(c.NamePath("new-thing"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := c.ByName("new-thing")
	if err != nil {
		t.Errorf("ByName() failed: %v", err)
	}
	defer f.Close()
}

func TestCacheBySHA256(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	c, err := NewCache(tmp)
	if err != nil {
		t.Fatal(err)
	}

	ne := sha256.Sum256([]byte("doesnt-exist"))
	if f, err := c.BySHA256(hex.EncodeToString(ne[:])); err != ErrCacheMiss || f != nil {
		t.Errorf("BySHA256(%q) returned (%v,%v), want (%v,%v)", "doesnt-exist", f, err, nil, ErrCacheMiss)
	}
	if err := ioutil.WriteFile(c.SHA256Path(hex.EncodeToString(ne[:])), nil, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := c.BySHA256(hex.EncodeToString(ne[:]))
	if err != nil {
		t.Errorf("BySHA256() failed: %v", err)
	}
	defer f.Close()

	os.Chtimes(c.SHA256Path(hex.EncodeToString(ne[:])), time.Now(), time.Now().Add(-7*24*time.Hour))
	if err := c.Clean(); err != nil {
		t.Errorf("Clean() failed: %v", err)
	}
}

func TestCacheUpdatesModtime(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	ne := sha256.Sum256([]byte(t.Name()))
	hash := hex.EncodeToString(ne[:])

	c, err := NewCache(tmp)
	if err != nil {
		t.Fatal(err)
	}
	c.BySHA256(hash) // Side effect of setting up directory structure.

	if err := ioutil.WriteFile(c.SHA256Path(hash), []byte("swiggity swooty"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(c.SHA256Path(hash), time.Now(), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	// Open the cached object.
	f, err := c.BySHA256(hash)
	if err != nil {
		t.Fatalf("BySHA256(%q) failed: %v", hash, err)
	}
	f.Close()

	// Expect the mtime to have been updated.
	s, err := os.Stat(c.SHA256Path(hash))
	if err != nil {
		t.Fatal(err)
	}
	if time.Now().Sub(s.ModTime()) > time.Minute {
		t.Errorf("subtime is too recent: %v, want %v", s.ModTime(), time.Now())
	}
}
