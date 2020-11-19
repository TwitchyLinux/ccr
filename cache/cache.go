// Package cache implements filesystem and memory caching of larger
// or expensive resources.
package cache

import (
	"encoding/base64"
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
	io.ReaderAt
}

func (c *Cache) GetObj(sha256 string) (value interface{}, ok bool) {
	return c.objCache.Get(sha256)
}

func (c *Cache) PutObj(sha256 string, v interface{}) {
	c.objCache.Add(sha256, v)
}

func (c *Cache) NamePath(name string) string {
	return filepath.Join(c.dir, "named", name)
}

func (c *Cache) ByName(name string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(c.dir, "named", name))
	if err != nil && os.IsNotExist(err) {
		return nil, ErrCacheMiss
	}
	return f, err
}

func (c *Cache) hashString(h []byte) string {
	s := base64.RawURLEncoding.EncodeToString(h)
	if len(s) > 36 {
		return s[:36]
	}
	return s
}

func (c *Cache) hashPath(h []byte) string {
	hash := c.hashString(h)
	return filepath.Join(c.dir, "hash", hash[:1], hash[1:])
}

func (c *Cache) IsHashCached(h []byte) (bool, error) {
	_, err := os.Stat(c.hashPath(h))
	switch {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, err
	}
}

func (c *Cache) DeleteHash(h []byte) error {
	return os.Remove(c.hashPath(h))
}

func (c *Cache) HashWriter(h []byte) (*os.File, error) {
	hash := c.hashString(h)
	dir := filepath.Join(c.dir, "hash", hash[:1])

	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(filepath.Join(dir, hash[1:]), os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
}

// ByHash returns a ReadSeekCloser for the given hash if cached.
// Regardless of whether the hash is cached or not, any directory tree
// for storing the object is created if it does not exist.
func (c *Cache) ByHash(h []byte) (ReadSeekCloser, error) {
	hash := c.hashString(h)

	dir := filepath.Join(c.dir, "hash", hash[:1])
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(filepath.Join(dir, hash[1:]))
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
	dirs, err := ioutil.ReadDir(filepath.Join(c.dir, "hash"))
	if err != nil {
		return err
	}

	now := time.Now()
	for _, d := range dirs {
		cf, err := ioutil.ReadDir(filepath.Join(c.dir, "hash", d.Name()))
		if err != nil {
			return err
		}
		for _, f := range cf {
			if f.ModTime().Add(4 * 24 * time.Hour).Before(now) {
				if err := os.Remove(filepath.Join(c.dir, "hash", d.Name(), f.Name())); err != nil {
					return err
				}
			}
		}
	}

	roots, err := ioutil.ReadDir(filepath.Join(c.dir, "chroots"))
	if err != nil {
		return err
	}
	for _, root := range roots {
		if root.ModTime().Add(36 * time.Hour).Before(now) {
			if err := os.RemoveAll(filepath.Join(c.dir, "chroots", root.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

// Chroot returns the path to the base filesystem of the rootFS with that hash.
// If create is set, the directory will be created if it doesnt exist, and
// cleared of contents if it does.
func (c *Cache) Chroot(h []byte, create bool) (string, error) {
	hash := c.hashString(h)
	p := filepath.Join(c.dir, "chroots", hash)

	s, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			if create {
				return p, os.Mkdir(p, 0755)
			}
			return "", ErrCacheMiss
		}
		return "", err
	}

	if create {
		if err := os.RemoveAll(p); err != nil {
			return "", err
		}
		return p, os.Mkdir(p, 0755)
	}

	// If the mod time is more than an hour, we boop it. Doing this
	// allows us to use the mod time as a signal for recent use of this
	// cache entry, but avoid unnecessary disk writes that would happen
	// if we unconditionally updated the modtime.
	if n := time.Now(); s.ModTime().Add(time.Hour).Before(n) {
		nt := unix.NsecToTimeval(n.UnixNano())
		if err := unix.Lutimes(p, []unix.Timeval{nt, nt}); err != nil {
			return "", fmt.Errorf("failed to update mtime: %v", err)
		}
	}
	return p, nil
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
	if err := os.MkdirAll(filepath.Join(dir, "hash"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "named"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "chroots"), 0755); err != nil {
		return nil, err
	}

	c, err := lru.New2Q(numCachedObjects)
	if err != nil {
		return nil, err
	}

	return &Cache{dir: dir, objCache: c}, nil
}
