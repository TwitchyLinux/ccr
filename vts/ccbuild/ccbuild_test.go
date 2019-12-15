package ccbuild

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

var (
	ttre = regexp.MustCompile("testdata/make_(.*)\\.ccr")
)

func TestLoad(t *testing.T) {
	s, err := NewScript(nil, "test", nil, nil)
	if err != nil {
		t.Errorf("NewScript() failed: %v", err)
	}
	t.Log(s)
}

var newScriptTestcases = []struct {
	name     string
	filename string
	err      error
	want     []vts.Target
}{
	{
		name:     "attr",
		filename: "testdata/make_attr.ccr",
		want: []vts.Target{
			&vts.Attr{
				Path:   "//test:amd64",
				Name:   "amd64",
				Parent: vts.TargetRef{Path: "//test/arch"},
				Value:  starlark.String("amd64"),
			},
		},
	},
	{
		name:     "resource with embedded detail",
		filename: "testdata/make_resource.ccr",
		want: []vts.Target{
			&vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Path: "common://resource/file"},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Path: "common://attrs:arch"},
							Value:  starlark.String("yeetos"),
						},
					},
				},
				Deps: []vts.TargetRef{
					{Path: "common://targets/libc"},
				},
			},
		},
	},
	{
		name:     "resource class",
		filename: "testdata/make_resourceclass.ccr",
		want: []vts.Target{
			&vts.ResourceClass{
				Path: "//test:shared_library",
				Name: "shared_library",
				Deps: []vts.TargetRef{
					{Path: "common://targets/ldd"},
				},
				Checks: []vts.TargetRef{
					{Path: "common://elf/samearch"},
					{Path: "common://elf/ldd-satisfiable"},
				},
			},
		},
	},
}

func TestNewScript(t *testing.T) {
	for _, tc := range newScriptTestcases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ioutil.ReadFile(tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			s, err := NewScript(d, "//test", nil, func(msg string) {
				t.Logf("script msg: %q", msg)
			})
			if err != nil {
				t.Fatalf("NewScript() failed: %v", err)
			}

			if diff := cmp.Diff(tc.want, s.targets); diff != "" {
				t.Errorf("unexpected targets result (+got, -want): \n%s", diff)
			}
		})
	}
}

func TestMakeTarget(t *testing.T) {
	scripts, err := filepath.Glob("testdata/make_*.ccr")
	if err != nil {
		t.Fatal(err)
	}

	for _, fPath := range scripts {
		targetType := ttre.FindAllStringSubmatch(fPath, 1)[0][1]

		t.Run(targetType, func(t *testing.T) {
			d, err := ioutil.ReadFile(fPath)
			if err != nil {
				t.Fatal(err)
			}
			s, err := NewScript(d, "//test/"+targetType, nil, func(msg string) {
				t.Logf("script msg: %q", msg)
			})
			if err != nil {
				t.Fatalf("NewScript() failed: %v", err)
			}

			if got, want := len(s.targets), 1; got != want {
				t.Fatalf("len(s.targets) = %d, want %d", got, want)
			}
			tt := strings.Replace(s.targets[0].TargetType().String(), "_", "", -1)
			if got, want := tt, targetType; got != want {
				t.Errorf("target.type = %v, want %v", got, want)
			}
			if gt, ok := s.targets[0].(vts.GlobalTarget); ok {
				if gt.GlobalPath() != "" && gt.GlobalPath() != "//test/"+targetType+":"+gt.TargetName() {
					t.Errorf("target.path = %v, want %v", gt.GlobalPath(), "//test/"+targetType+":"+gt.TargetName())
				}
			}
		})
	}
}
