package gen

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/log"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

func TestFilePopulateResource(t *testing.T) {
	tcs := []struct {
		name        string
		r           *vts.Resource
		err         string
		expectFiles map[string]os.FileMode
	}{
		{
			name: "basic",
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
							Val:    starlark.String("0645"),
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
			},
			expectFiles: map[string]os.FileMode{"/yeetfile": os.FileMode(0645)},
		},
		{
			name: "missing_attrs",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Source: &vts.TargetRef{
					Target: &vts.Puesdo{
						Kind:         vts.FileRef,
						ContractPath: "testdata/something.ccr",
						Path:         "file.txt",
					},
				},
			},
			err: "path: attr not specified",
		},
		{
			name: "file_not_present",
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
							Val:    starlark.String("0645"),
						},
					},
				},
				Source: &vts.TargetRef{
					Target: &vts.Puesdo{
						Kind:         vts.FileRef,
						ContractPath: "testdata/something.ccr",
						Path:         "lol its missing.txt",
					},
				},
			},
			err: "stat testdata/lol its missing.txt: no such file or directory",
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

			gc := GenerationContext{
				Cache:   c,
				Console: &log.Silent{},
				RunnerEnv: &vts.RunnerEnv{
					FS: osfs.New(outDir),
				},
			}
			err = PopulateResource(gc, tc.r, tc.r.Source.Target)

			switch {
			case err == nil && tc.err != "":
				t.Errorf("PopulateResource() did not error, expected %q", tc.err)
			case err != nil && tc.err == "":
				t.Errorf("PopulateResource(%v) failed: %v", tc.r, err)
			case err != nil && tc.err != err.Error():
				t.Errorf("PopulateResource().err = %v, want %v", err, tc.err)
			}

			for p, m := range tc.expectFiles {
				s, err := os.Stat(filepath.Join(outDir, p))
				if err != nil {
					t.Errorf("missing output file: %v", err)
				} else {
					if s.Mode() != m {
						t.Errorf("%q: mode = %v, want %v", p, s.Mode(), m)
					}
				}
			}
		})
	}
}
