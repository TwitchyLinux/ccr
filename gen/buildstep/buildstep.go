// Package buildstep implements individual build steps.
package buildstep

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/twitchylinux/ccr/cache"
	"gopkg.in/src-d/go-billy.v4"
)

type RunningBuild interface {
	OverlayMountPath() string
	OverlayUpperPath() string
	RootFS() billy.Filesystem
	SourceFS() billy.Filesystem
	ExecBlocking(args []string, stdout, stderr io.Writer) error
}

// download returns a reader to the file referenced by url, downloading
// and caching it if necessary.
func download(c *cache.Cache, sha256, url string) (cache.ReadSeekCloser, error) {
	f, err := c.BySHA256(sha256)
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
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("unexpected response code '%d' (%s)", r.StatusCode, r.Status)
	}
	w, err := os.OpenFile(c.SHA256Path(sha256), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(w, r.Body); err != nil {
		w.Close()
		return nil, err
	}
	w.Seek(0, os.SEEK_SET)
	return w, nil
}
