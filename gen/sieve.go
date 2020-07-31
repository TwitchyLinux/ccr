package gen

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"

	"github.com/gobwas/glob"
	"github.com/twitchylinux/ccr/vts"
)

// unionFileset exposes a single fileset that is the ordered union of input
// filesets.
type unionFileset struct {
	all       []fileset
	remaining []fileset
}

func (fs *unionFileset) Close() error {
	for _, f := range fs.all {
		if err := f.Close(); err != nil {
			return err
		}
	}
	fs.all, fs.remaining = nil, nil
	return nil
}

func (fs *unionFileset) Next() (path string, header *tar.Header, err error) {
	if len(fs.remaining) == 0 {
		return "", nil, io.EOF
	}

	p, h, err := fs.remaining[0].Next()
	if err != nil {
		if err == io.EOF {
			fs.remaining = fs.remaining[1:]
			return fs.Next()
		}
		return "", nil, err
	}
	return p, h, nil
}

func (fs *unionFileset) Read(b []byte) (int, error) {
	if len(fs.remaining) == 0 {
		return 0, errors.New("file not open")
	}
	return fs.remaining[0].Read(b)
}

// filterFileset exposes a fileset that filters files based on path.
type filterFileset struct {
	base            fileset
	excludePatterns []glob.Glob
}

func (fs *filterFileset) Close() error {
	return fs.base.Close()
}

func (fs *filterFileset) Next() (path string, header *tar.Header, err error) {
outer:
	for {
		path, h, err := fs.base.Next()
		if err != nil {
			return "", nil, err
		}

		for _, p := range fs.excludePatterns {
			if p.Match(path) {
				continue outer
			}
		}
		return path, h, nil
	}
}

func (fs *filterFileset) Read(b []byte) (int, error) {
	return fs.base.Read(b)
}

func filesetForSieve(gc GenerationContext, s *vts.Sieve) (fileset, error) {
	inputs := make([]fileset, 0, len(s.Inputs))
	for i, inp := range s.Inputs {
		fs, err := filesetForSource(gc, inp.Target)
		if err != nil {
			return nil, fmt.Errorf("input[%d] loading fileset: %v", i, err)
		}
		inputs = append(inputs, fs)
	}
	var out fileset = &unionFileset{all: inputs, remaining: inputs}

	if len(s.ExcludeGlobs) > 0 {
		ff := filterFileset{base: out, excludePatterns: make([]glob.Glob, len(s.ExcludeGlobs))}
		for i, p := range s.ExcludeGlobs {
			var err error
			if ff.excludePatterns[i], err = glob.Compile(p); err != nil {
				return nil, err
			}
		}
		out = &ff
	}

	if s.AddPrefix != "" {
		return nil, errors.New("add prefix not implemented")
	}

	return out, nil
}
