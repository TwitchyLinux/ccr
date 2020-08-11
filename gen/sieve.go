package gen

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/gobwas/glob"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/match"
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
	base     fileset
	patterns []glob.Glob
	invert   bool
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

		for _, p := range fs.patterns {
			if p.Match(path) {
				if fs.invert {
					return path, h, nil
				}
				continue outer
			}
		}
		if fs.invert {
			continue
		}
		return path, h, nil
	}
}

func (fs *filterFileset) Read(b []byte) (int, error) {
	return fs.base.Read(b)
}

// prefixFileset exposes a fileset that adds a prefix onto all files.
type prefixFileset struct {
	base   fileset
	prefix string
}

func (fs *prefixFileset) Close() error {
	return fs.base.Close()
}

func (fs *prefixFileset) Next() (path string, header *tar.Header, err error) {
	path, h, err := fs.base.Next()
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(fs.prefix, path), h, nil
}

func (fs *prefixFileset) Read(b []byte) (int, error) {
	return fs.base.Read(b)
}

// renameFileset exposes a fileset that renames files based on match rules.
type renameFileset struct {
	base  fileset
	rules *match.FilenameRules
}

func (fs *renameFileset) Close() error {
	return fs.base.Close()
}

func (fs *renameFileset) Next() (path string, header *tar.Header, err error) {
	path, h, err := fs.base.Next()
	if err != nil {
		return "", nil, err
	}

	if newPath := fs.rules.Match(path); newPath != "" {
		return newPath, h, nil
	}
	return path, h, nil
}

func (fs *renameFileset) Read(b []byte) (int, error) {
	return fs.base.Read(b)
}

func filesetForSieve(gc GenerationContext, s *vts.Sieve) (fileset, error) {
	// Fast path: Sieve's that only select a subpath + remove a prefix from a
	// path, from build output.
	if s.IsDirPrefixSieve() && len(s.Inputs) == 1 {
		if b, isBuild := s.Inputs[0].Target.(*vts.Build); isBuild {
			h, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
			if err != nil {
				return nil, err
			}

			fs, err := gc.Cache.FilesetSubdir(h, s.DirPrefix())
			if err != nil {
				return nil, err
			}
			// We still need the rename rules to trim the prefix off the filenames.
			return &renameFileset{base: fs, rules: s.Renames}, nil
		}
	}

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
		ff := filterFileset{base: out, patterns: make([]glob.Glob, len(s.ExcludeGlobs))}
		for i, p := range s.ExcludeGlobs {
			var err error
			if ff.patterns[i], err = glob.Compile(p); err != nil {
				return nil, err
			}
		}
		out = &ff
	}
	if len(s.IncludeGlobs) > 0 {
		ff := filterFileset{base: out, patterns: make([]glob.Glob, len(s.IncludeGlobs)), invert: true}
		for i, p := range s.IncludeGlobs {
			var err error
			if ff.patterns[i], err = glob.Compile(p); err != nil {
				return nil, err
			}
		}
		out = &ff
	}

	if s.AddPrefix != "" {
		out = &prefixFileset{base: out, prefix: s.AddPrefix}
	}
	if s.Renames != nil {
		out = &renameFileset{base: out, rules: s.Renames}
	}

	return out, nil
}
