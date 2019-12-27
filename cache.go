package ccr

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrCacheMiss = errors.New("cache miss")

// Cache manages cached files.
type Cache struct {
	dir string
}

func (c *Cache) GetDebPkgsPath() string {
	return filepath.Join(c.dir, "debpkgs")
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
