package ccbuild

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
)

var (
	ttre = regexp.MustCompile("testdata/make_(.*)\\.ccr")

	posType   = reflect.TypeOf(&vts.DefPosition{})
	filterPos = cmp.FilterPath(func(p cmp.Path) bool {
		for _, p := range p {
			if p.Type() == posType {
				return true
			}
		}
		return false
	}, cmp.Ignore())
)

func TestLoad(t *testing.T) {
	s, err := NewScript(nil, "test", "", nil, nil)
	if err != nil {
		t.Errorf("NewScript() failed: %v", err)
	}
	t.Log(s)
}

var newScriptTestcases = []struct {
	name     string
	filename string
	err      string
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
				Val:    starlark.String("amd64"),
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
							Val:    starlark.String("yeetos"),
						},
					},
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.ModeClass},
							Val:    starlark.String("0755"),
						},
					},
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.TargetClass},
							Val:    starlark.String("/doesnt-make/sense"),
						},
					},
				},
				Deps: []vts.TargetRef{
					{Path: "common://targets/libc"},
				},
				Source: &vts.TargetRef{Target: &vts.Puesdo{
					Kind:         vts.FileRef,
					Path:         "/usr/share/boots.txt",
					ContractPath: "testdata/make_resource.ccr",
					Host:         true,
				},
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
	{
		name:     "resource with helper attrs",
		filename: "testdata/resource_with_helpers.ccr",
		want: []vts.Target{
			&vts.Resource{
				Path:   "//test:somefile",
				Name:   "somefile",
				Parent: vts.TargetRef{Path: "common://resource/file"},
				Details: []vts.TargetRef{
					{
						Target: &vts.Attr{
							Parent: vts.TargetRef{Target: common.PathClass},
							Val:    starlark.String("/etc/yeet"),
						},
					},
				},
			},
		},
	},
	{
		name:     "resource with file source",
		filename: "testdata/resource_with_file_source.ccr",
		want: []vts.Target{
			&vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Path: "common://resource/file"},
				Source: &vts.TargetRef{Target: &vts.Puesdo{
					Kind:         vts.FileRef,
					Path:         "./boots.txt",
					ContractPath: "testdata/resource_with_file_source.ccr",
				},
				},
			},
		},
	},
	{
		name:     "resource with deb source",
		filename: "testdata/resource_with_deb_source.ccr",
		want: []vts.Target{
			&vts.Resource{
				Path:   "//test:yeet",
				Name:   "yeet",
				Parent: vts.TargetRef{Path: "common://resources:file"},
				Source: &vts.TargetRef{Target: &vts.Puesdo{
					Kind:         vts.DebRef,
					URL:          "https://example.com/somedeb.deb",
					SHA256:       "1234",
					ContractPath: "testdata/resource_with_deb_source.ccr",
				},
				},
			},
		},
	},
	{
		name:     "attr with computed value",
		filename: "testdata/attr_with_computed_value.ccr",
		want: []vts.Target{
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Path: "//test/arch"},
				Val: &vts.ComputedValue{
					ContractPath: "testdata/attr_with_computed_value.ccr",
					Filename:     "testdata/a.py",
					Func:         "some_func",
					InlineScript: []byte(""),
				},
			},
		},
	},
	{
		name:     "attr with inline computed value",
		filename: "testdata/attr_with_inline_computed_value.ccr",
		want: []vts.Target{
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Path: "//test/arch"},
				Val: &vts.ComputedValue{
					ContractPath: "testdata/attr_with_inline_computed_value.ccr",
					InlineScript: []byte("\n  v = 2**2\n  return v\n  "),
				},
			},
		},
	},
	{
		name:     "build",
		filename: "testdata/make_build.ccr",
		want: []vts.Target{
			&vts.Build{
				Path:         "//test:thingy",
				ContractPath: "testdata/make_build.ccr",
				Name:         "thingy",
				HostDeps: []vts.TargetRef{
					{
						Path: "//test:meow",
						Constraints: []vts.RefConstraint{
							{
								Meta:   vts.TargetRef{Target: common.SemverClass},
								Params: []starlark.Value{starlark.String(">>"), starlark.String("1.2.3")},
								Eval:   &RefComparisonConstraint{},
							},
						},
					},
				},
				Steps: []*vts.BuildStep{
					{Kind: vts.StepUnpackGz, Path: "go1.11.4.tar.gz", ToPath: "src"},
					{Kind: vts.StepShellCmd, Args: []string{"echo mate"}},
				},
			},
		},
	},
	{
		name:     "build_invalid_output",
		filename: "testdata/invalid_build_output.ccr",
		err:      "invalid build output key: cannot use type starlark.Int",
	},
}

func TestNewScript(t *testing.T) {
	for _, tc := range newScriptTestcases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ioutil.ReadFile(tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			s, err := NewScript(d, "//test", tc.filename, nil, func(msg string) {
				t.Logf("script msg: %q", msg)
			})
			if err != nil && tc.err != err.Error() {
				t.Fatalf("NewScript() failed: %v", err)
			}

			if tc.err == "" {
				// for _, target := range s.targets {
				// 	if err := target.Validate(); err != nil {
				// 		t.Errorf("failed to validate: %v", err)
				// 	}
				// }

				if diff := cmp.Diff(tc.want, s.targets, filterPos,
					cmpopts.IgnoreUnexported(vts.Attr{}),
					cmpopts.IgnoreFields(vts.ComputedValue{}, "ContractDir"),
					cmpopts.IgnoreFields(vts.Build{}, "ContractDir", "Output"),
					cmpopts.IgnoreFields(vts.RefConstraint{}, "Eval")); diff != "" {
					// fmt.Println(string(s.targets[0].(*vts.Attr).Val.(*vts.ComputedValue).InlineScript))
					t.Errorf("unexpected targets result (+got, -want): \n%s", diff)
				}
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
		spl := strings.Split(ttre.FindAllStringSubmatch(fPath, 1)[0][1], "_")
		targetType := spl[len(spl)-1]

		t.Run(targetType, func(t *testing.T) {
			d, err := ioutil.ReadFile(fPath)
			if err != nil {
				t.Fatal(err)
			}
			s, err := NewScript(d, "//test/"+targetType, fPath, nil, func(msg string) {
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

			for _, tgt := range s.targets {
				if tgt.DefinedAt() == nil || tgt.DefinedAt().Path == "" {
					t.Errorf("target.DefinedAt() not specified: %+v", tgt.DefinedAt())
				}
			}
		})
	}
}
