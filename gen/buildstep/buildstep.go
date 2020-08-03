// Package buildstep implements individual build steps.
package buildstep

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/twitchylinux/ccr/cache"
	"gopkg.in/src-d/go-billy.v4"
)

type RunningBuild interface {
	OverlayMountPath() string
	OverlayUpperPath() string
	RootFS() billy.Filesystem
	SourceFS() billy.Filesystem
	ExecBlocking(wd string, args []string, stdout, stderr io.Writer) (int, error)
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func downloadWithClient(client httpClient, c *cache.Cache, s256, url string) (cache.ReadSeekCloser, error) {
	f, err := c.BySHA256(s256)
	switch {
	case err == cache.ErrCacheMiss:
	case err == nil:
		return f, nil
	default:
		return nil, err
	}

	// Download.
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("unexpected response code '%d' (%s)", r.StatusCode, r.Status)
	}
	w, err := os.OpenFile(c.SHA256Path(s256), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(w, r.Body); err != nil {
		w.Close()
		return nil, err
	}

	// Check hash is good.
	if _, err := w.Seek(0, os.SEEK_SET); err != nil {
		w.Close()
		return nil, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, w); err != nil {
		w.Close()
		return nil, err
	}
	if got, want := fmt.Sprintf("%x", h.Sum(nil)), strings.ToLower(s256); got != want {
		w.Close()
		os.Remove(c.SHA256Path(s256))
		return nil, fmt.Errorf("incorrect hash: %q != %q", got, want)
	}

	if _, err := w.Seek(0, os.SEEK_SET); err != nil {
		w.Close()
		return nil, err
	}
	return w, nil
}

// download returns a reader to the file referenced by url, downloading
// and caching it if necessary.
func download(c *cache.Cache, s256, url string) (cache.ReadSeekCloser, error) {
	return downloadWithClient(http.DefaultClient, c, s256, url)
}
