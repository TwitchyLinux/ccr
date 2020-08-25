package gen

import (
	"io"
	"os"
	"testing"

	"github.com/gobwas/glob"
	"github.com/twitchylinux/ccr/log"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/match"
)

func TestSievePrefixFastpath(t *testing.T) {
	rb, c, d := makeEnv(t)
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind: vts.StepShellCmd,
			Args: []string{"mkdir /tmp/aa && mkdir /tmp/bb"},
		},
		{
			Kind: vts.StepShellCmd,
			Args: []string{"touch /tmp/blueberries && touch /tmp/aa/beh && touch /tmp/bb/a && touch /tmp/aa/yeow"},
		},
	}

	if err := rb.Generate(c, os.Stdout, os.Stderr); err != nil {
		t.Errorf("Generate() failed: %v", err)
	}

	b := &vts.Build{
		Output: &match.FilenameRules{
			Rules: []match.MatchRule{
				{P: glob.MustCompile("**"), Out: &match.StripPrefixOutputMapper{Prefix: ""}},
			},
		},
	}
	h, _ := b.RollupHash(nil, nil)
	if err := rb.WriteToCache(c, b, h); err != nil {
		t.Fatal(err)
	}

	s := &vts.Sieve{
		Inputs: []vts.TargetRef{
			{Target: b},
		},
		Renames: &match.FilenameRules{Rules: []match.MatchRule{
			{P: glob.MustCompile("tmp/aa/**"), Out: &match.StripPrefixOutputMapper{Prefix: "tmp/aa/"}},
		}},
		IncludeGlobs: []string{"tmp/aa/**"},
	}
	if !s.IsDirPrefixSieve() {
		t.Fatalf("sieve is not a directory prefix")
	}

	fs, err := filesetForSieve(GenerationContext{Cache: c, Console: &log.Silent{}}, s)
	if err != nil {
		t.Fatalf("filesetForSieve() failed: %v", err)
	}
	defer fs.Close()

	wantFiles := map[string]bool{
		"beh":  true,
		"yeow": true,
	}

	for {
		path, _, err := fs.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("iterating buildset: %v", err)
		}

		found, ok := wantFiles[path]
		switch {
		case found:
			delete(wantFiles, path)
		case !ok:
			t.Errorf("unexpected file %q", path)
		}
	}

	for path, _ := range wantFiles {
		t.Errorf("expected file %q", path)
	}
}

func TestSieveFilesets(t *testing.T) {
	s := &vts.Sieve{
		Inputs: []vts.TargetRef{
			{Target: &vts.Puesdo{
				Kind:         vts.FileRef,
				ContractPath: "testdata/something.ccr",
				Path:         "file.txt",
			}},
			{Target: &vts.Puesdo{
				Kind:         vts.FileRef,
				ContractPath: "testdata/something.ccr",
				Path:         "cool.tar.gz",
			}},
			{Target: &vts.Puesdo{
				Kind:         vts.FileRef,
				ContractPath: "testdata/something.ccr",
				Path:         "symlinks.tar.gz",
			}},
		},
		ExcludeGlobs: []string{"cool.tar.gz"},
		IncludeGlobs: []string{"*.gz", "*.txt"},
		Renames: &match.FilenameRules{Rules: []match.MatchRule{
			{P: glob.MustCompile("*.txt"), Out: match.LiteralOutputMapper("b.txt")},
		}},
	}
	if s.IsDirPrefixSieve() {
		t.Fatal("sieve should not be a directory prefix")
	}

	fs, err := filesetForSieve(GenerationContext{Console: &log.Silent{}}, s)
	if err != nil {
		t.Fatalf("filesetForSieve() failed: %v", err)
	}
	defer fs.Close()

	wantFiles := map[string]bool{
		"symlinks.tar.gz": true,
		"b.txt":           true,
	}

	for {
		path, _, err := fs.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("iterating buildset: %v", err)
		}

		found, ok := wantFiles[path]
		switch {
		case found:
			delete(wantFiles, path)
		case !ok:
			t.Errorf("unexpected file %q", path)
		}
	}

	for path, _ := range wantFiles {
		t.Errorf("expected file %q", path)
	}
}
