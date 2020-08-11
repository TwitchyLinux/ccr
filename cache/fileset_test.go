package cache

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	missingHash = sha256.Sum256([]byte{0})
	createHash  = sha256.Sum256([]byte{1})
)

func TestFileset(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	if err := ioutil.WriteFile(filepath.Join(tmp, "something.txt"), []byte("something"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := NewCache(tmp)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure misses are fine.
	if _, _, _, err := c.FileInFileset(missingHash[:], "somefile.txt"); err != ErrCacheMiss {
		t.Errorf("FileInFileset(missingHash, somefile.txt) returned %v, want %v", err, ErrCacheMiss)
	}
	// Make sure misses still create the parent dir.
	if s, err := os.Stat(filepath.Join(tmp, fmt.Sprintf("%X", missingHash[:2]))); err == nil && s.IsDir() {
		t.Errorf("stat for missingHash dir failed: %v", err)
	}

	// Make sure we can create a fileset.
	pfs, err := c.CommitFileset(createHash[:])
	if err != nil {
		t.Fatalf("CommitFileset(%X) failed: %v", createHash, err)
	}
	// Attempt to add a file.
	s, _ := os.Stat(filepath.Join(tmp, "something.txt"))
	content, err := os.Open(filepath.Join(tmp, "something.txt"))
	if err != nil {
		t.Fatal(err)
	}
	pfs.AddFile("something.txt", s, content)
	if err := pfs.Close(); err != nil {
		t.Errorf("pfs.Close() failed: %v", err)
	}

	// Make sure we can read out that file.
	r, closer, mode, err := c.FileInFileset(createHash[:], "something.txt")
	if err != nil {
		t.Fatalf("FileInFileset(%X, %q) failed: %v", createHash[:], "something.txt", err)
	}
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		t.Fatal(err)
	}
	if want := []byte("something"); !bytes.Equal(b.Bytes(), want) {
		t.Errorf("bad content: %q, want %q", b.Bytes(), want)
	}
	if mode != s.Mode() {
		t.Errorf("mode = %v, want %v", mode, s.Mode())
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	// Check the FilesetReader as well.
	fr, err := c.FilesetReader(createHash[:])
	if err != nil {
		t.Fatalf("FilesetReader(%X) failed: %v", createHash, err)
	}
	defer func() {
		if err := fr.Close(); err != nil {
			t.Errorf("Failed to close FilesetReader: %v", err)
		}
	}()

	foundFiles := map[string]int64{}
	for {
		path, _, err := fr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("fr.Next() returned unexpected error: %v", err)
		}
		var b bytes.Buffer
		var n int64
		if n, err = io.Copy(&b, fr); err != nil {
			t.Errorf("Failed full read of %q: %v", path, err)
		}
		foundFiles[path] = n
	}

	if want := (map[string]int64{"something.txt": int64(len("something"))}); !reflect.DeepEqual(foundFiles, want) {
		t.Errorf("FilesetReader found files %+v, want %+v", foundFiles, want)
	}
}

func TestSubdirFileset(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	if err := ioutil.WriteFile(filepath.Join(tmp, "something.txt"), []byte("something"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewCache(tmp)
	if err != nil {
		t.Fatal(err)
	}

	pfs, err := c.CommitFileset(createHash[:])
	if err != nil {
		t.Fatal(err)
	}
	pfs.addFile(&tar.Header{Name: "10", Typeflag: tar.TypeReg, Mode: int64(0644), Size: 2}, bytes.NewReader([]byte("1 ")))
	pfs.addFile(&tar.Header{Name: "20", Typeflag: tar.TypeReg, Mode: int64(0644), Size: 2}, bytes.NewReader([]byte("2 ")))
	pfs.addFile(&tar.Header{Name: "30", Typeflag: tar.TypeReg, Mode: int64(0644), Size: 2}, bytes.NewReader([]byte("3 ")))
	pfs.addFile(&tar.Header{Name: "40/41", Typeflag: tar.TypeReg, Mode: int64(0644), Size: 2}, bytes.NewReader([]byte("4 ")))
	pfs.addFile(&tar.Header{Name: "40/42", Typeflag: tar.TypeReg, Mode: int64(0640), Size: 2}, bytes.NewReader([]byte("5 ")))
	pfs.addFile(&tar.Header{Name: "40/430/431", Typeflag: tar.TypeReg, Mode: int64(0640), Size: 3}, bytes.NewReader([]byte("6  ")))
	pfs.addFile(&tar.Header{Name: "40/430/432", Typeflag: tar.TypeSymlink, Linkname: "abc"}, bytes.NewReader(nil))

	if err := pfs.Close(); err != nil {
		t.Fatal(err)
	}

	fss, err := c.FilesetSubdir(createHash[:], "40/")
	if err != nil {
		t.Fatalf("FilesetSubdir(%q) failed: %v", "40/", err)
	}
	defer fss.Close()

	want := map[string]struct {
		h       tar.Header
		content string
	}{
		"40/41": {
			tar.Header{
				Name:     "40/41",
				Typeflag: tar.TypeReg,
				Mode:     int64(0644),
			},
			"4 ",
		},
		"40/42": {
			tar.Header{
				Name:     "40/42",
				Typeflag: tar.TypeReg,
				Mode:     int64(0640),
			},
			"5 ",
		},
		"40/430": {
			tar.Header{
				Name:     "40/430",
				Typeflag: tar.TypeDir,
				Mode:     int64(0755),
			},
			"",
		},
		"40/430/431": {
			tar.Header{
				Name:     "40/430/431",
				Typeflag: tar.TypeReg,
				Mode:     int64(0640),
			},
			"6  ",
		},
		"40/430/432": {
			tar.Header{
				Name:     "40/430/432",
				Typeflag: tar.TypeSymlink,
				Linkname: "abc",
			},
			"",
		},
	}

	for i := 0; i < len(want); i++ {
		p, h, err := fss.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		f, ok := want[p]
		if !ok {
			t.Errorf("got file %v, not wanted", p)
			continue
		}

		if p != f.h.Name {
			t.Errorf("%s: path = %q, want %q", p, p, f.h.Name)
		}
		if f.h.Typeflag != h.Typeflag {
			t.Errorf("%s: type = %v, want %v", p, h.Typeflag, f.h.Typeflag)
		}
		if f.h.Mode != h.Mode {
			t.Errorf("%s: mode = %o, want %o", p, h.Mode, f.h.Mode)
		}

		if f.h.Typeflag == tar.TypeDir || f.h.Typeflag == tar.TypeSymlink {
			continue
		}

		d, err := ioutil.ReadAll(fss)
		if err != nil {
			t.Errorf("reading failed: %v", err)
		}
		if !bytes.Equal(d, []byte(f.content)) {
			t.Errorf("%s: content = %q, want %q", p, string(d), f.content)
		}
	}
	if _, _, err := fss.Next(); err != io.EOF {
		t.Errorf("Next() = %v, expected %v", err, io.EOF)
	}
}
