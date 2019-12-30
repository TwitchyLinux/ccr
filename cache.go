package ccr

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

var ErrCacheMiss = errors.New("cache miss")

// Cache manages cached files.
type Cache struct {
	dir string
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

func (c *Cache) BySHA256(hash string) (io.ReadCloser, error) {
	dir := filepath.Join(c.dir, hash[:2])
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(filepath.Join(dir, hash[2:]))
	if err != nil && os.IsNotExist(err) {
		return nil, ErrCacheMiss
	}
	return f, err
}

// NewCache initializes a new cache backed by dir. If dir is the empty string,
// a standard dotpath in the users home directory is used.
func NewCache(dir string) (*Cache, error) {
	if dir == "" {
		hd, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(hd, ".ccr", "cache")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &Cache{dir: dir}, nil
}
