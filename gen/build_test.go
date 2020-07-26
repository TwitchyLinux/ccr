package gen

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
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
			ToPath: "output",
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
