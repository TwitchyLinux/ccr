package buildstep

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/twitchylinux/ccr/vts"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

type runningBuildMock struct {
	dir            string
	srcFS, buildFS billy.Filesystem
}

func (rb *runningBuildMock) Dir() string {
	return rb.dir
}
func (rb *runningBuildMock) RootFS() billy.Filesystem {
	return rb.buildFS
}
func (rb *runningBuildMock) SourceFS() billy.Filesystem {
	return rb.srcFS
}

func makeEnv(t *testing.T, copySrc ...string) (*runningBuildMock, string) {
	t.Helper()
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(d, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(d, "build"), 0755); err != nil {
		t.Fatal(err)
	}

	for _, path := range copySrc {
		if err := exec.Command("cp", path, filepath.Join(d, "src")).Run(); err != nil {
			t.Fatal(err)
		}
	}

	return &runningBuildMock{
		dir:     filepath.Join(d, "build"),
		srcFS:   osfs.New(filepath.Join(d, "src")),
		buildFS: osfs.New("/"),
	}, d
}

func TestUnpackGz(t *testing.T) {
	rb, d := makeEnv(t, "testdata/cool.tar.gz")
	defer os.RemoveAll(d)

	if err := RunUnpackGz(rb, &vts.BuildStep{
		ToPath: "output",
		Path:   "cool.tar.gz",
	}); err != nil {
		t.Errorf("RunUnpackGz() failed: %v", err)
	}

	// filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Println(path)
	// 	return nil
	// })

	data, err := ioutil.ReadFile(filepath.Join(rb.Dir(), "output/blue/slang"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if want := []byte("bluezies\n"); !bytes.Equal(data, want) {
		t.Errorf("file content = %q, want %q", data, want)
	}
}
