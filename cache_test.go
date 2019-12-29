package ccr

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
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
}
