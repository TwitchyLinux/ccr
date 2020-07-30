package gen

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
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
	outMapping := starlark.NewDict(12)
	// Leading / should have been removed by (*vts.Build).OutputMappings().
	outMapping.SetKey(starlark.String("a.txt"), starlark.String("b.txt"))

	if err := rb.WriteToCache(c, &vts.Build{Output: outMapping}, bytes.Repeat([]byte{0}, 32)); err != nil {
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

	data, err := ioutil.ReadFile(filepath.Join(rb.OverlayUpperPath(), "output/blue/slang"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if want := []byte("bluezies\n"); !bytes.Equal(data, want) {
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

func TestPatchingBuildEnv(t *testing.T) {
	tcs := []struct {
		name           string
		r              *vts.Resource
		expectFiles    map[string]os.FileMode
		backingArchive string
	}{
		{
			name: "file target",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Source: &vts.TargetRef{
					Target: &vts.Build{
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
				},
			},
			expectFiles: map[string]os.FileMode{"/somefile.txt": os.FileMode(0644)},
		},
		{
			name: "build target",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Source: &vts.TargetRef{
					Target: &vts.Build{
						Path: "//test:fake_file_build",
						Name: "fake_file_build",
						PatchIns: map[string]vts.TargetRef{
							"/p": {Target: &vts.Build{}},
						},
					},
				},
			},
			expectFiles:    map[string]os.FileMode{"/p/yeetfile": os.FileMode(0644)},
			backingArchive: "testdata/fake_file_build_cache.tar.gz",
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
				b := tc.r.Source.Target.(*vts.Build).PatchIns["/p"].Target.(*vts.Build)
				h, err := b.RollupHash(nil, nil)
				if err != nil {
					t.Fatalf("RollupHash() failed: %v", err)
				}

				p := c.SHA256Path(hex.EncodeToString(h))
				cmd := exec.Command("install", "-D", tc.backingArchive, p)
				cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to yeet backing archive into cache: %v", err)
				}
			}

			b := tc.r.Source.Target.(*vts.Build)
			env, err := proc.NewEnv(false)
			if err != nil {
				t.Fatal(err)
			}
			rb := RunningBuild{env: env, fs: osfs.New("/"), contractDir: b.ContractDir}
			defer rb.Close()
			if err := rb.Patch(GenerationContext{
				Cache: c,
				RunnerEnv: &vts.RunnerEnv{
					FS: osfs.New(outDir),
				}}, b.PatchIns); err != nil {
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

func TestPopulateBuild(t *testing.T) {
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
				p := c.SHA256Path(hex.EncodeToString(h))
				cmd := exec.Command("install", "-D", tc.backingArchive, p)
				cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to yeet backing archive into cache: %v", err)
				}
			}

			if err := populateBuild(GenerationContext{
				Cache: c,
				RunnerEnv: &vts.RunnerEnv{
					FS: osfs.New(outDir),
				}}, tc.r, b); err != nil {
				t.Errorf("populateBuild(%v, %v) failed: %v", tc.r, b, err)
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
	outTxtOnly := starlark.NewDict(3)
	outTxtOnly.SetKey(starlark.String("*.txt"), &vts.StripPrefixOutputMapper{Prefix: "/"})

	tcs := []struct {
		name        string
		b           *vts.Build
		expectFiles map[string]os.FileMode
	}{
		{
			name: "file",
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
				Output: outTxtOnly,
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
					FS: osfs.New(outDir),
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
