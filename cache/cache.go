// Package cache implements filesystem and memory caching of larger
// or expensive resources.
package cache

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/sys/unix"
)

// numCachedObjects should be fairly low, as the objects in question are
// usually massive (few Mb) such as unpacked debs.
const numCachedObjects = 16

var ErrCacheMiss = errors.New("cache miss")

// Cache manages cached files.
type Cache struct {
	dir      string
	objCache *lru.TwoQueueCache
}

type ReadSeekCloser interface {
	io.ReadCloser
	io.Seeker
}

func (c *Cache) GetObj(sha256 string) (value interface{}, ok bool) {
	return c.objCache.Get(sha256)
}

func (c *Cache) PutObj(sha256 string, v interface{}) {
	c.objCache.Add(sha256, v)
}

func (c *Cache) NamePath(name string) string {
	return filepath.Join(c.dir, name)
}

func (c *Cache) ByName(name string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(c.dir, name))
	if err != nil && os.IsNotExist(err) {
		return nil, ErrCacheMiss
	}
	return f, err
}

func (c *Cache) SHA256Path(hash string) string {
	return filepath.Join(c.dir, hash[:2], hash[2:])
}

// BySHA256 returns a ReadSeekCloser for the given hash if cached.
// Regardless of whether the hash is cached or not, any directory tree
// for storing the object is created if it does not exist.
func (c *Cache) BySHA256(hash string) (ReadSeekCloser, error) {
	dir := filepath.Join(c.dir, hash[:2])
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(filepath.Join(dir, hash[2:]))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	// If the mod time is more than 3 hours, we boop it. Doing this
	// allows us to use the mod time as a signal for recent use of this
	// cache entry, but avoid unnecessary disk writes that would happen
	// if we unconditionally updated the modtime.
	if n := time.Now(); s.ModTime().Add(3 * time.Hour).Before(n) {
		nt := unix.NsecToTimeval(n.UnixNano())
		if err := unix.Futimes(int(f.Fd()), []unix.Timeval{nt, nt}); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to update mtime: %v", err)
		}
	}

	return f, nil
}

// Clean purges old objects from the cache.
func (c *Cache) Clean() error {
	dirs, err := ioutil.ReadDir(c.dir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, d := range dirs {
		cf, err := ioutil.ReadDir(filepath.Join(c.dir, d.Name()))
		if err != nil {
			return err
		}
		for _, f := range cf {
			if f.ModTime().Add(4 * 24 * time.Hour).Before(now) {
				if err := os.Remove(filepath.Join(c.dir, d.Name(), f.Name())); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func defaultCacheDir() (string, error) {
	if cd, err := os.UserCacheDir(); err == nil {
		return filepath.Join(cd, "ccr"), nil
	}

	hd, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(hd, ".ccr", "cache"), nil
}

// NewCache initializes a new cache backed by dir. If dir is the empty string,
// a standard dotpath in the users home directory is used.
func NewCache(dir string) (*Cache, error) {
	if dir == "" {
		var err error
		if dir, err = defaultCacheDir(); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	c, err := lru.New2Q(numCachedObjects)
	if err != nil {
		return nil, err
	}

	return &Cache{dir: dir, objCache: c}, nil
}
