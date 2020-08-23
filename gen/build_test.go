package gen

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gobwas/glob"
	"github.com/google/crfs/stargz"
	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"github.com/twitchylinux/ccr/vts/match"
	"go.starlark.net/starlark"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

func makeEnv(t *testing.T, copySrc ...string) (*RunningBuild, *cache.Cache, string) {
	t.Helper()
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

	for _, path := range copySrc {
		if err := exec.Command("cp", path, d).Run(); err != nil {
			t.Fatal(err)
		}
	}

	env, err := proc.NewEnv(false)
	if err != nil {
		t.Fatalf("creating new environment: %v", err)
	}
	return &RunningBuild{env: env, fs: osfs.New("/"), contractDir: d}, c, d
}

func TestBuildWriteToCache(t *testing.T) {
	rb, c, d := makeEnv(t, "testdata/cool.tar.gz")
	defer os.RemoveAll(d)
	defer rb.Close()

	// Make a fake file.
	if err := ioutil.WriteFile(filepath.Join(rb.OverlayUpperPath(), "a.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	// Make a second fake file, that shouldnt be written out.
	if err := ioutil.WriteFile(filepath.Join(rb.OverlayUpperPath(), "b.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	// Make a symlink.
	if err := os.Symlink("b.txt", filepath.Join(rb.OverlayUpperPath(), "sym.txt")); err != nil {
		t.Fatal(err)
	}

	if err := rb.WriteToCache(c, &vts.Build{Output: &match.FilenameRules{
		Rules: []match.MatchRule{
			{P: glob.MustCompile("a.txt"), Out: match.LiteralOutputMapper("b.txt")},
			{P: glob.MustCompile("sym.txt"), Out: match.LiteralOutputMapper("sym.txt")},
		},
	}}, bytes.Repeat([]byte{0}, 32)); err != nil {
		t.Errorf("rb.WriteToCache() failed: %v", err)
	}

	// Make sure the file we set was written out.
	_, closer, _, err := c.FileInFileset(bytes.Repeat([]byte{0}, 32), "b.txt")
	if err != nil {
		t.Errorf("FileInFileset(%X, %q) failed: %v", bytes.Repeat([]byte{0}, 32), "b.txt", err)
	} else {
		closer.Close()
	}
	// Make sure the other file we wrote out (but didnt map as an output) did NOT get written out.
	if _, closer, _, err = c.FileInFileset(bytes.Repeat([]byte{0}, 32), "a.txt"); err == nil {
		closer.Close()
		t.Errorf("FileInFileset(%X, %q) did not fail: %v", bytes.Repeat([]byte{0}, 32), "a.txt", err)
	}
	// Find that symbolic link and make sure its info is good.
	r, err := c.FilesetReader(bytes.Repeat([]byte{0}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for {
		path, h, err := r.Next()
		if err != nil {
			if err == io.EOF {
				t.Error("did not find sym.txt")
				break
			}
			t.Fatalf("iterating buildset: %v", err)
		}

		if path == "sym.txt" {
			if h.Typeflag != tar.TypeSymlink {
				t.Errorf("type = %v, want symlink", h.Typeflag)
			}
			if h.Linkname != "b.txt" {
				t.Errorf("target = %q, want b.txt", h.Linkname)
			}
			if m := os.FileMode(h.Mode); m&os.ModeSymlink == 0 {
				t.Errorf("target was not symlink (mode = %v)", m)
			}
			break
		}
	}
}

func TestStepUnpackGz(t *testing.T) {
	rb, c, d := makeEnv(t, "testdata/cool.tar.gz")
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind:   vts.StepUnpackGz,
			ToPath: "/output",
			Path:   "cool.tar.gz",
		},
	}

	if err := rb.Generate(c); err != nil {
		t.Errorf("Generate() failed: %v", err)
	}

	// filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Println(path)
	// 	return nil
	// })

	p := filepath.Join(rb.OverlayUpperPath(), "output/blue/slang")
	data, err := ioutil.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if want := []byte("bluezies\n"); !bytes.Equal(data, want) {
		t.Errorf("file content = %q, want %q", data, want)
	}
	s, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2020, 7, 23, 18, 7, 26, 0, time.Local); s.ModTime().Unix() != want.Unix() {
		t.Errorf("modtime = %v, want %v", s.ModTime(), want)
	}
}

func TestStepPatch(t *testing.T) {
	rb, c, d := makeEnv(t, "testdata/patch.diff")
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind: vts.StepShellCmd,
			Args: []string{"echo \"pre-patch content\" > /tmp/a"},
		},
		{
			Kind:   vts.StepPatch,
			ToPath: "/tmp",
			Path:   "patch.diff",
		},
	}

	if err := rb.Generate(c); err != nil {
		t.Errorf("Generate() failed: %v", err)
	}

	// filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Println(path)
	// 	return nil
	// })

	data, err := ioutil.ReadFile(filepath.Join(rb.OverlayUpperPath(), "tmp/a"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if want := []byte("post-patch content\n"); !bytes.Equal(data, want) {
		t.Errorf("file content = %q, want %q", data, want)
	}
}

func TestStepUnpackXz(t *testing.T) {
	rb, c, d := makeEnv(t, "testdata/archive.tar.xz")
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind:   vts.StepUnpackXz,
			ToPath: "output",
			Path:   "archive.tar.xz",
		},
	}

	if err := rb.Generate(c); err != nil {
		t.Errorf("Generate() failed: %v", err)
	}

	data, err := ioutil.ReadFile(filepath.Join(rb.OverlayUpperPath(), "output/fake.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if want := []byte("Fake contents!!\n"); !bytes.Equal(data, want) {
		t.Errorf("file content = %q, want %q", data, want)
	}
}

func TestStepShellCmd(t *testing.T) {
	rb, c, d := makeEnv(t)
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind: vts.StepShellCmd,
			Args: []string{"touch blueberries"},
		},
	}

	if err := rb.Generate(c); err != nil {
		t.Errorf("Generate() failed: %v", err)
	}

	_, err := ioutil.ReadFile(filepath.Join(rb.OverlayUpperPath(), "tmp", "blueberries"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
}

func TestStepShellCmdErrors(t *testing.T) {
	rb, c, d := makeEnv(t)
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind: vts.StepShellCmd,
			Args: []string{"exit 14"},
		},
	}

	err := rb.Generate(c)
	switch {
	case err == nil:
		t.Error("Generate() = nil, want non-nil error")
	case err != nil && err.Error() != "step 1 (bash_cmd) failed: exit status 14":
		t.Errorf("Generate() returned %q, want %q", err.Error(), "step 1 (bash_cmd) failed: exit status 14")
	}
}

func TestStepConfigure(t *testing.T) {
	rb, c, d := makeEnv(t)
	defer os.RemoveAll(d)
	defer rb.Close()
	rb.steps = []*vts.BuildStep{
		{
			Kind: vts.StepConfigure,
			Dir:  "/tmp/somedir",
		},
	}
	if err := os.Mkdir(filepath.Join(rb.OverlayUpperPath(), "tmp/somedir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(rb.OverlayUpperPath(), "tmp/somedir/configure"), []byte("#!/bin/bash\nexit 0"), 0777); err != nil {
		t.Fatal(err)
	}

	if err := rb.Generate(c); err != nil {
		t.Errorf("Generate() failed: %v", err)
	}
}

func TestPatchingBuildEnv(t *testing.T) {
	tcs := []struct {
		name           string
		b              *vts.Build
		expectFiles    map[string]os.FileMode
		backingArchive string
	}{
		{
			name: "file target",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/somefile.txt": {Target: &vts.Puesdo{
						Kind:         vts.FileRef,
						ContractPath: "testdata/something.ccr",
						Path:         "file.txt",
					}},
				},
			},
			expectFiles: map[string]os.FileMode{"/somefile.txt": os.FileMode(0644)},
		},
		{
			name: "build target",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/p": {Target: &vts.Build{}},
				},
			},
			expectFiles:    map[string]os.FileMode{"/p/yeetfile": os.FileMode(0644)},
			backingArchive: "testdata/fake_file_build_cache.tar.gz",
		},
		{
			name: "sieve target",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/usr/include": {Target: &vts.Sieve{
						Inputs: []vts.TargetRef{
							{Target: &vts.Puesdo{
								Kind:         vts.FileRef,
								ContractPath: "testdata/something.ccr",
								Path:         "file.txt",
							}},
						},
						Renames: &match.FilenameRules{Rules: []match.MatchRule{
							{P: glob.MustCompile("file.txt"), Out: match.LiteralOutputMapper("file2.txt")},
						}},
					}},
				},
			},
			expectFiles: map[string]os.FileMode{"/usr/include/file2.txt": os.FileMode(0644)},
		},
		{
			name: "resource file target",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/ext_root": {Target: &vts.Resource{
						Parent: vts.TargetRef{Target: common.FileResourceClass},
						Details: []vts.TargetRef{
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.PathClass},
									Val:    starlark.String("/usr/kek.txt"),
								},
							},
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.ModeClass},
									Val:    starlark.String("0641"),
								},
							},
						},
						Source: &vts.TargetRef{
							Target: &vts.Puesdo{
								Kind:         vts.FileRef,
								ContractPath: "testdata/something.ccr",
								Path:         "file.txt",
							},
						},
					}},
				},
			},
			expectFiles: map[string]os.FileMode{"/ext_root/usr/kek.txt": os.FileMode(0641)},
		},
		{
			name: "resource symlink target",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/ext_root": {Target: &vts.Resource{
						Parent: vts.TargetRef{Target: common.SymlinkResourceClass},
						Details: []vts.TargetRef{
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.PathClass},
									Val:    starlark.String("/usr/kek"),
								},
							},
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.TargetClass},
									Val:    starlark.String("../"),
								},
							},
						},
						Source: &vts.TargetRef{
							Target: common.SymlinkGenerator,
						},
					}},
				},
			},
			expectFiles: map[string]os.FileMode{"/ext_root/usr/kek": os.ModeDir | os.FileMode(0755)},
		},
		{
			name: "component target",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/ext_root": {Target: &vts.Component{
						Deps: []vts.TargetRef{
							{Target: &vts.Resource{
								Parent: vts.TargetRef{Target: common.FileResourceClass},
								Details: []vts.TargetRef{
									{
										Target: &vts.Attr{
											Parent: vts.TargetRef{Target: common.PathClass},
											Val:    starlark.String("/usr/swiggity.txt"),
										},
									},
								},
								Source: &vts.TargetRef{
									Target: &vts.Puesdo{
										Kind:         vts.FileRef,
										ContractPath: "testdata/something.ccr",
										Path:         "file.txt",
									},
								}}},
						}}},
				},
			},
			expectFiles: map[string]os.FileMode{"/ext_root/usr/swiggity.txt": os.FileMode(0644)},
		},
	}

	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	c, err := cache.NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(outDir)

			// This is not the correct way to write filesets into the cache (should
			// use cache.PendingFileset etc), but because the fileset format is .tar.gz,
			// this should work fine.
			if tc.backingArchive != "" {
				b := tc.b.PatchIns["/p"].Target.(*vts.Build)
				h, err := b.RollupHash(nil, nil)
				if err != nil {
					t.Fatalf("RollupHash() failed: %v", err)
				}

				w, err := c.HashWriter(h)
				if err != nil {
					t.Fatal(err)
				}
				r, err := os.Open(tc.backingArchive)
				if err != nil {
					t.Fatal(err)
				}
				defer r.Close()
				if _, err := io.Copy(w, r); err != nil {
					t.Fatal(err)
				}
				w.Close()
			}

			env, err := proc.NewEnv(false)
			if err != nil {
				t.Fatal(err)
			}
			rb := RunningBuild{env: env, fs: osfs.New("/"), contractDir: tc.b.ContractDir}
			defer rb.Close()
			if err := rb.Patch(GenerationContext{
				Cache: c,
				RunnerEnv: &vts.RunnerEnv{
					FS:  osfs.New(outDir),
					Dir: outDir,
				}}, tc.b.PatchIns); err != nil {
				t.Fatalf("Patch() failed: %v", err)
			}

			for p, m := range tc.expectFiles {
				s, err := os.Stat(filepath.Join(env.OverlayUpperPath(), p))
				if err != nil {
					t.Errorf("failed to check expected file %q: %v", p, err)
				}
				if err == nil && m != s.Mode() {
					t.Errorf("%q: mode = %v, want %v", p, s.Mode(), m)
				}
			}
		})
	}
}

func TestPopulateResourceFromBuild(t *testing.T) {
	tcs := []struct {
		name           string
		r              *vts.Resource
		expectFiles    map[string]os.FileMode
		backingArchive string
	}{
		{
			name: "file",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/yeetfile"),
						},
					},
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.ModeClass},
							Val:    starlark.String("0600"),
						},
					},
				},
				Source: &vts.TargetRef{
					Target: &vts.Build{
						Path: "//test:fake_file_build",
						Name: "fake_file_build",
					},
				},
			},
			expectFiles:    map[string]os.FileMode{"/yeetfile": os.FileMode(0600)},
			backingArchive: "testdata/fake_file_build_cache.tar.gz",
		},
		{
			name: "symlinks",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.CHeadersResourceClass},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/dir"),
						},
					},
				},
				Source: &vts.TargetRef{
					Target: &vts.Build{
						Path: "//test:fake_file_build",
						Name: "fake_file_build",
					},
				},
			},
			expectFiles:    map[string]os.FileMode{"/dir/pasta.txt": os.FileMode(0644)},
			backingArchive: "testdata/symlinks.tar.gz",
		},
		{
			name: "c headers",
			r: &vts.Resource{
				Path:   "//test:yote",
				Name:   "yote",
				Parent: vts.TargetRef{Target: common.CHeadersResourceClass},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/usr/include"),
						},
					},
				},
				Source: &vts.TargetRef{
					Target: &vts.Build{
						Path: "//test:fake_headers_build",
						Name: "fake_headers_build",
					},
				},
			},
			expectFiles: map[string]os.FileMode{
				"/usr":                       os.ModeDir | os.FileMode(0755),
				"/usr/include":               os.ModeDir | os.FileMode(0755),
				"/usr/include/asm":           os.ModeDir | os.FileMode(0755),
				"/usr/include/asm/headerz.h": os.FileMode(0644),
			},
			backingArchive: "testdata/fake_cheaders_build_cache.tar.gz",
		},
	}

	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	c, err := cache.NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(outDir)

			b := tc.r.Source.Target.(*vts.Build)
			h, err := b.RollupHash(nil, nil)
			if err != nil {
				t.Fatalf("RollupHash() failed: %v", err)
			}

			// This is not the correct way to write filesets into the cache (should
			// use cache.PendingFileset etc), but because the fileset format is .tar.gz,
			// this should work fine.
			if tc.backingArchive != "" {
				w, err := c.HashWriter(h)
				if err != nil {
					t.Fatal(err)
				}
				sgz := stargz.NewWriter(w)
				r, err := os.Open(tc.backingArchive)
				if err != nil {
					t.Fatal(err)
				}
				defer r.Close()
				if err := sgz.AppendTar(r); err != nil {
					t.Fatal(err)
				}
				if err := sgz.Close(); err != nil {
					t.Fatal(err)
				}
				w.Close()
			}
			if err := PopulateResource(GenerationContext{
				Cache: c,
				RunnerEnv: &vts.RunnerEnv{
					FS:  osfs.New(outDir),
					Dir: outDir,
				}}, tc.r, b); err != nil {
				t.Errorf("PopulateResource(%v, %v) failed: %v", tc.r, b, err)
			}

			for p, m := range tc.expectFiles {
				s, err := os.Stat(filepath.Join(outDir, p))
				if err != nil {
					t.Errorf("failed to check expected file %q: %v", p, err)
				}
				if err == nil && m != s.Mode() {
					t.Errorf("%q: mode = %v, want %v", p, s.Mode(), m)
				}
			}
		})
	}
}

func TestGenerateBuild(t *testing.T) {
	tcs := []struct {
		name        string
		b           *vts.Build
		expectFiles map[string]os.FileMode
	}{
		{
			name: "patches in output",
			b: &vts.Build{
				Path: "//test:fake_file_build",
				Name: "fake_file_build",
				PatchIns: map[string]vts.TargetRef{
					"/somefile.txt": {Target: &vts.Puesdo{
						Kind:         vts.FileRef,
						ContractPath: "testdata/something.ccr",
						Path:         "file.txt",
					}},
					"/should_get_output.o": {Target: &vts.Puesdo{
						Kind:         vts.FileRef,
						ContractPath: "testdata/something.ccr",
						Path:         "file.txt",
					}},
				},
				Output: &match.FilenameRules{
					Rules: []match.MatchRule{
						{P: glob.MustCompile("*.txt"), Out: &match.StripPrefixOutputMapper{Prefix: "/"}},
					},
				},
			},
			expectFiles: map[string]os.FileMode{"somefile.txt": os.FileMode(0644)},
		},
		{
			name: "injections not in output",
			b: &vts.Build{
				Path: "//test:f",
				Name: "f",
				PatchIns: map[string]vts.TargetRef{
					"/somefile.txt": {Target: &vts.Puesdo{
						Kind:         vts.FileRef,
						ContractPath: "testdata/something.ccr",
						Path:         "file.txt",
					}},
				},
				Injections: []vts.TargetRef{
					{Target: &vts.Resource{
						Path:   "//test:yote",
						Name:   "yote",
						Parent: vts.TargetRef{Target: common.FileResourceClass},
						Details: []vts.TargetRef{
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.PathClass},
									Val:    starlark.String("/usr/kek.txt"),
								},
							},
						},
						Source: &vts.TargetRef{
							Target: &vts.Puesdo{
								Kind:         vts.FileRef,
								ContractPath: "testdata/something.ccr",
								Path:         "file.txt",
							},
						},
					}},
				},
				Output: &match.FilenameRules{
					Rules: []match.MatchRule{
						{P: glob.MustCompile("**.txt"), Out: &match.StripPrefixOutputMapper{Prefix: ""}},
					},
				},
			},
			expectFiles: map[string]os.FileMode{"somefile.txt": os.FileMode(0644)},
		},
	}

	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	c, err := cache.NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(outDir)

			h, err := tc.b.RollupHash(nil, nil)
			if err != nil {
				t.Fatalf("RollupHash() failed: %v", err)
			}

			if err := Generate(GenerationContext{
				Cache: c,
				RunnerEnv: &vts.RunnerEnv{
					FS:  osfs.New(outDir),
					Dir: outDir,
				}}, tc.b); err != nil {
				t.Errorf("Generate(%v) failed: %v", tc.b, err)
			}

			fr, err := c.FilesetReader(h)
			defer fr.Close()

			for {
				path, h, err := fr.Next()
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatalf("iterating buildset: %v", err)
				}

				if m, ok := tc.expectFiles[path]; ok {
					if got := os.FileMode(h.Mode); m != got {
						t.Errorf("%q: mode = %v, want %v", path, os.FileMode(h.Mode), m)
					}
				} else {
					t.Errorf("Unexpected file in buildset: %q", path)
				}
				delete(tc.expectFiles, path)
			}

			for p, _ := range tc.expectFiles {
				t.Errorf("file %q missing from buildset", p)
			}
		})
	}
}

func TestBuildInjections(t *testing.T) {
	tcs := []struct {
		name        string
		b           *vts.Build
		expectFiles map[string]os.FileMode

		buildSource    *vts.Build
		backingArchive string
	}{
		{
			name: "unary file resource",
			b: &vts.Build{
				Injections: []vts.TargetRef{
					{Target: &vts.Resource{
						Path:   "//test:yote",
						Name:   "yote",
						Parent: vts.TargetRef{Target: common.FileResourceClass},
						Details: []vts.TargetRef{
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.PathClass},
									Val:    starlark.String("/usr/kek.txt"),
								},
							},
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.ModeClass},
									Val:    starlark.String("0641"),
								},
							},
						},
						Source: &vts.TargetRef{
							Target: &vts.Puesdo{
								Kind:         vts.FileRef,
								ContractPath: "testdata/something.ccr",
								Path:         "file.txt",
							},
						},
					}},
				},
			},
			expectFiles: map[string]os.FileMode{"usr/kek.txt": os.FileMode(0641)},
		},
		{
			name: "multi file resource",
			b: &vts.Build{
				Path: "//test:f",
				Name: "f",
				Injections: []vts.TargetRef{
					{Target: &vts.Resource{
						Parent: vts.TargetRef{Target: common.CHeadersResourceClass},
						Details: []vts.TargetRef{
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.PathClass},
									Val:    starlark.String("/usr/inc"),
								},
							},
						},
						Source: &vts.TargetRef{
							Target: &vts.Sieve{
								Inputs: []vts.TargetRef{
									{Target: &vts.Puesdo{
										Kind:         vts.FileRef,
										ContractPath: "testdata/something.ccr",
										Path:         "file.txt",
									}},
								},
								Renames: &match.FilenameRules{Rules: []match.MatchRule{
									{P: glob.MustCompile("file.txt"), Out: match.LiteralOutputMapper("kek.h")},
								}},
							},
						},
					}},
				},
			},
			expectFiles: map[string]os.FileMode{"usr/inc/kek.h": os.FileMode(0644)},
		},
		{
			name: "build",
			b: &vts.Build{
				Injections: []vts.TargetRef{
					{Target: &vts.Build{
						Path: "//test:fake_headers_build",
						Name: "fake_headers_build",
					}},
				},
			},
			buildSource: &vts.Build{
				Path: "//test:fake_headers_build",
				Name: "fake_headers_build",
			},
			backingArchive: "testdata/fake_cheaders_build_cache.tar.gz",
			expectFiles: map[string]os.FileMode{
				"asm":           os.ModeDir | os.FileMode(0755),
				"asm/headerz.h": os.FileMode(0644),
			},
		},
		{
			name: "file_from_build",
			b: &vts.Build{
				Path: "//test:f",
				Name: "f",
				Injections: []vts.TargetRef{
					{Target: &vts.Resource{
						Parent: vts.TargetRef{Target: common.CHeadersResourceClass},
						Details: []vts.TargetRef{
							{
								Target: &vts.Attr{
									Parent: vts.TargetRef{Target: common.PathClass},
									Val:    starlark.String("/usr/inc2"),
								},
							},
						},
						Source: &vts.TargetRef{Target: &vts.Build{
							Path: "//test:fake_headers_build2",
							Name: "fake_headers_build2",
						}},
					}},
				},
			},
			buildSource: &vts.Build{
				Path: "//test:fake_headers_build2",
				Name: "fake_headers_build2",
			},
			backingArchive: "testdata/fake_cheaders_build_cache.tar.gz",
			expectFiles: map[string]os.FileMode{
				"/usr/inc2/asm":           os.ModeDir | os.FileMode(0755),
				"/usr/inc2/asm/headerz.h": os.FileMode(0644),
			},
		},
		{
			name: "sieve",
			b: &vts.Build{
				Injections: []vts.TargetRef{
					{Target: &vts.Sieve{
						Inputs: []vts.TargetRef{
							{Target: &vts.Build{
								Path: "//test:fake_headers_build3",
								Name: "fake_headers_build3",
							}},
							{Target: &vts.Puesdo{
								Kind:         vts.FileRef,
								ContractPath: "testdata/something.ccr",
								Path:         "file.txt",
							}},
						},
						Renames: &match.FilenameRules{Rules: []match.MatchRule{
							{P: glob.MustCompile("**.h"), Out: &match.StripPrefixOutputMapper{Prefix: "asm/"}},
							{P: glob.MustCompile("file.txt"), Out: match.LiteralOutputMapper("kek.h")},
						}},
					}},
				},
			},
			buildSource: &vts.Build{
				Path: "//test:fake_headers_build3",
				Name: "fake_headers_build3",
			},
			backingArchive: "testdata/fake_cheaders_build_cache.tar.gz",
			expectFiles: map[string]os.FileMode{
				"headerz.h": os.FileMode(0644),
				"kek.h":     os.FileMode(0644),
			},
		},
		{
			name: "simple component",
			b: &vts.Build{
				Injections: []vts.TargetRef{
					{Target: &vts.Component{Deps: []vts.TargetRef{
						{Target: &vts.Build{
							Path: "//test:fake_headers_build4",
							Name: "fake_headers_build4",
						}}}}},
				},
			},
			buildSource: &vts.Build{
				Path: "//test:fake_headers_build4",
				Name: "fake_headers_build4",
			},
			backingArchive: "testdata/fake_cheaders_build_cache.tar.gz",
			expectFiles: map[string]os.FileMode{
				"asm":           os.ModeDir | os.FileMode(0755),
				"asm/headerz.h": os.FileMode(0644),
			},
		},
		{
			name: "component containing generated files",
			b: &vts.Build{
				Injections: []vts.TargetRef{
					{Target: &vts.Component{Deps: []vts.TargetRef{
						{Target: &vts.Resource{
							Parent: vts.TargetRef{Target: common.CHeadersResourceClass},
							Details: []vts.TargetRef{
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.PathClass},
										Val:    starlark.String("/usr/inc3"),
									},
								},
							},
							Source: &vts.TargetRef{Target: &vts.Build{
								Path: "//test:fake_headers_build5",
								Name: "fake_headers_build5",
							}},
						}},
						{Target: &vts.Resource{
							Parent: vts.TargetRef{Target: common.FileResourceClass},
							Details: []vts.TargetRef{
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.PathClass},
										Val:    starlark.String("/usr/share/blueberries.txt"),
									},
								},
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.ModeClass},
										Val:    starlark.String("0745"),
									},
								},
							},
							Source: &vts.TargetRef{
								Target: &vts.Puesdo{
									Kind:         vts.FileRef,
									ContractPath: "testdata/something.ccr",
									Path:         "file.txt",
								},
							},
						}},
						{Target: &vts.Resource{
							Parent: vts.TargetRef{Target: common.SymlinkResourceClass},
							Details: []vts.TargetRef{
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.PathClass},
										Val:    starlark.String("/usr/share/fruit"),
									},
								},
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.ModeClass},
										Val:    starlark.String("0745"),
									},
								},
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.TargetClass},
										Val:    starlark.String("blueberries.txt"),
									},
								},
							},
							Source: &vts.TargetRef{Target: common.SymlinkGenerator},
						}},
						{Target: &vts.Resource{
							Parent: vts.TargetRef{Target: common.DirResourceClass},
							Details: []vts.TargetRef{
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.PathClass},
										Val:    starlark.String("/test_dir"),
									},
								},
								{
									Target: &vts.Attr{
										Parent: vts.TargetRef{Target: common.ModeClass},
										Val:    starlark.String("0755"),
									},
								},
							},
							Source: &vts.TargetRef{Target: common.DirGenerator},
						}},
					}}},
				},
			},
			buildSource: &vts.Build{
				Path: "//test:fake_headers_build5",
				Name: "fake_headers_build5",
			},
			backingArchive: "testdata/fake_cheaders_build_cache.tar.gz",
			expectFiles: map[string]os.FileMode{
				"/test_dir":                  os.ModeDir | os.FileMode(0755),
				"/usr/inc3/asm":              os.ModeDir | os.FileMode(0755),
				"/usr/inc3/asm/headerz.h":    os.FileMode(0644),
				"/usr/share/blueberries.txt": os.FileMode(0745),
				"/usr/share/fruit":           os.FileMode(0745),
			},
		},
	}

	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	c, err := cache.NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(outDir)

			// This is not the correct way to write filesets into the cache (should
			// use cache.PendingFileset etc), but because the fileset format is .tar.gz,
			// this should work fine.
			if tc.backingArchive != "" {
				h, err := tc.buildSource.RollupHash(nil, nil)
				if err != nil {
					t.Fatalf("RollupHash() failed: %v", err)
				}

				w, err := c.HashWriter(h)
				if err != nil {
					t.Fatal(err)
				}
				sgz := stargz.NewWriter(w)
				r, err := os.Open(tc.backingArchive)
				if err != nil {
					t.Fatal(err)
				}
				defer r.Close()
				if err := sgz.AppendTar(r); err != nil {
					t.Fatal(err)
				}
				if err := sgz.Close(); err != nil {
					t.Fatal(err)
				}
				w.Close()
			}

			env, err := proc.NewEnv(false)
			if err != nil {
				t.Fatal(err)
			}
			rb := RunningBuild{env: env, fs: osfs.New("/"), contractDir: tc.b.ContractDir}
			defer rb.Close()

			gc := GenerationContext{
				Cache: c,
				RunnerEnv: &vts.RunnerEnv{
					FS:  osfs.New(outDir),
					Dir: outDir,
				}}
			if err := rb.Inject(gc, tc.b.Injections); err != nil {
				t.Fatalf("rb.Inject() failed: %v", err)
			}

			for p, m := range tc.expectFiles {
				fp := filepath.Join(rb.OverlayPatchPath(), p)
				s, err := os.Stat(fp)
				if err != nil {
					t.Errorf("%s: %v", p, err)
					continue
				}
				if s.Mode() != m {
					t.Errorf("mode = %v, want %v", s.Mode(), m)
				}
			}
		})
	}
}
