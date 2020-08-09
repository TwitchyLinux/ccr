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

func TestHashedRW(t *testing.T) {
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
	if f, err := c.ByHash(ne[:]); err != ErrCacheMiss || f != nil {
		t.Errorf("ByHash(%q) returned (%v,%v), want (%v,%v)", "doesnt-exist", f, err, nil, ErrCacheMiss)
	}
	if isCached, err := c.IsHashCached(ne[:]); isCached || err != nil {
		t.Errorf("IsHashCached(%q) returned (%v,%v), want (false,nil)", ne, isCached, err)
	}

	if err := ioutil.WriteFile(c.hashPath(ne[:]), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if isCached, err := c.IsHashCached(ne[:]); !isCached || err != nil {
		t.Errorf("IsHashCached(%q) returned (%v,%v), want (true,nil)", ne, isCached, err)
	}
	f, err := c.ByHash(ne[:])
	if err != nil {
		t.Errorf("ByHash() failed: %v", err)
	}
	defer f.Close()

	os.Chtimes(c.hashPath(ne[:]), time.Now(), time.Now().Add(-7*24*time.Hour))
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
	f, err := c.HashWriter(ne[:])
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if err := os.Chtimes(c.hashPath(ne[:]), time.Now(), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	// Open the cached object.
	f2, err := c.ByHash(ne[:])
	if err != nil {
		t.Fatalf("ByHash(%q) failed: %v", hash, err)
	}
	f2.Close()

	// Expect the mtime to have been updated.
	s, err := os.Stat(c.hashPath(ne[:]))
	if err != nil {
		t.Fatal(err)
	}
	if time.Now().Sub(s.ModTime()) > time.Minute {
		t.Errorf("subtime is too recent: %v, want %v", s.ModTime(), time.Now())
	}
}
