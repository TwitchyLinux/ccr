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

func TestPopulateResourceFromDeb(t *testing.T) {
	debSrc := &vts.Puesdo{
		Kind:         vts.DebRef,
		ContractPath: "../testdata/generators/deb.ccr",
		SHA256:       "d2e9dd92dd3f1bdbafd63b4a122415d28fecc5f6152d82fa0f76a9766d95ba17",
		Path:         "libwoff1_1.0.2-1_amd64.deb",
		Name:         "fake_deb",
	}
	tcs := []struct {
		name        string
		r           *vts.Resource
		expectFiles map[string]os.FileMode
		err         string
	}{
		{
			name: "deb_explicit_mode",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/usr/lib/x86_64-linux-gnu/libwoff2common.so.1.0.2"),
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
					Target: debSrc,
				},
			},
			expectFiles: map[string]os.FileMode{"/usr/lib/x86_64-linux-gnu/libwoff2common.so.1.0.2": os.FileMode(0600)},
		},
		{
			name: "deb_inherit_mode",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/usr/lib/x86_64-linux-gnu/libwoff2common.so.1.0.2"),
						},
					},
				},
				Source: &vts.TargetRef{
					Target: debSrc,
				},
			},
			expectFiles: map[string]os.FileMode{"/usr/lib/x86_64-linux-gnu/libwoff2common.so.1.0.2": os.FileMode(0755)},
		},
		{
			name: "deb_file_missing",
			r: &vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Target: common.FileResourceClass},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/lol/missing"),
						},
					},
				},
				Source: &vts.TargetRef{
					Target: debSrc,
				},
			},
			err: os.ErrNotExist.Error(),
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

			d := tc.r.Source.Target.(*vts.Puesdo)

			err = PopulateResource(GenerationContext{
				Cache:   c,
				Console: &log.Silent{},
				RunnerEnv: &vts.RunnerEnv{
					FS: osfs.New(outDir),
				},
			}, tc.r, d)

			switch {
			case err != nil && tc.err != err.Error():
				t.Errorf("PopulateResource(%v, %v) err = %v, want %v", tc.r, d, err, tc.err)
			case err != nil && tc.err == "":
				t.Errorf("PopulateResource(%v, %v) failed: %v", tc.r, d, err)
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
