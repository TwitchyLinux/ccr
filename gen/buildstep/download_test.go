package buildstep

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/twitchylinux/ccr/cache"
)

type staticResponseFakeServer struct {
	d *bytes.Reader
}

func (s *staticResponseFakeServer) Close() error {
	return nil
}

func (s *staticResponseFakeServer) Read(b []byte) (int, error) {
	return s.d.Read(b)
}

func (s *staticResponseFakeServer) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Body:       s,
		StatusCode: 200,
	}, nil
}

func TestDownloadFailsOnBadHash(t *testing.T) {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(d, ".__cache"), 0755); err != nil {
		t.Fatal(err)
	}
	c, err := cache.NewCache(filepath.Join(d, ".__cache"))
	if err != nil {
		t.Fatal(err)
	}

	s256 := hex.EncodeToString(bytes.Repeat([]byte{1}, sha256.Size))
	respData := []byte("some content here lol\n")
	r, err := downloadWithClient(&staticResponseFakeServer{d: bytes.NewReader(respData)}, c, s256, "https://aaa.com/somefile.txt")
	if err == nil {
		r.Close()
		t.Error("Expected non-nil error")
	}
	if want := fmt.Sprintf("incorrect hash: \"%x\" != %q", sha256.Sum256(respData), s256); err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
}

func TestDownload(t *testing.T) {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(d, ".__cache"), 0755); err != nil {
		t.Fatal(err)
	}
	c, err := cache.NewCache(filepath.Join(d, ".__cache"))
	if err != nil {
		t.Fatal(err)
	}

	respData := []byte("swiggity swooty the chonky cat is a cutie\n")
	h := sha256.Sum256(respData)
	s256 := hex.EncodeToString(h[:])
	r, err := downloadWithClient(&staticResponseFakeServer{d: bytes.NewReader(respData)}, c, s256, "https://aaa.com/cats.txt")
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	defer r.Close()

	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		t.Errorf("failed copy: %v", err)
	}
	if !bytes.Equal(b.Bytes(), respData) {
		t.Errorf("data = %q, want %q", string(b.Bytes()), string(respData))
	}
}
