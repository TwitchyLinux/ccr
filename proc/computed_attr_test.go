package proc

import (
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func TestComputedAttr(t *testing.T) {
	class := &vts.AttrClass{
		Name: "semantic",
		Path: "common://attrs/version:semantic",
	}
	res := &vts.Resource{
		Name: "some_file",
		Path: "//test:some_file",
	}

	tcs := []struct {
		name     string
		attr     *vts.Attr
		expected starlark.Value
	}{
		{
			"basic number",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/a.py",
					Func:     "some_number",
				},
			},
			starlark.MakeInt(42),
		},
		{
			"basic string",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/a.py",
					Func:     "some_string",
				},
			},
			starlark.String("1.2"),
		},
		{
			"read attr info",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/a.py",
					Func:     "read_attr",
				},
			},
			starlark.String("name=amd64, path=//test:amd64"),
		},
		{
			"read parent info",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/a.py",
					Func:     "parent_info",
				},
			},
			starlark.String("attr_class: name=semantic, path=common://attrs/version:semantic"),
		},
		{
			"read target info",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/a.py",
					Func:     "target_info",
				},
			},
			starlark.String("resource: name=some_file, path=//test:some_file, deps=[], details=[]"),
		},
		{
			"run basic",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/run.py",
					Func:     "run_basic",
				},
			},
			starlark.String("Linux\nye"),
		},
		{
			"no write",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/run.py",
					Func:     "check_no_write",
				},
			},
			starlark.MakeInt(1),
		},
		{
			"working dir",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					Filename: "testdata/run.py",
					Func:     "wd",
				},
			},
			starlark.String("/"),
		},
		{
			"inline",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					InlineScript: []byte("'1' + '2'"),
				},
			},
			starlark.String("12"),
		},
		{
			"inline multiline",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					InlineScript: []byte("v = 'yeet'\nreturn v"),
				},
			},
			starlark.String("yeet"),
		},
		{
			"inline multiline auto-return",
			&vts.Attr{
				Name:   "amd64",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					InlineScript: []byte("\n  v = 2*2\n  v\n  "),
				},
			},
			starlark.MakeInt(4),
		},
		{
			"inline multiline starts first line",
			&vts.Attr{
				Name:   "BROSKIE",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					InlineScript: []byte("v = attr.name\nreturn 'yeet_' + v\n\n  "),
				},
			},
			starlark.String("yeet_BROSKIE"),
		},
		{
			"go version",
			&vts.Attr{
				Name:   "BROSKIE",
				Path:   "//test:amd64",
				Parent: vts.TargetRef{Target: class},
				Val: &vts.ComputedValue{
					ReadWrite: true,
					InlineScript: []byte(`
inv = run("go", "version")
spl = inv.output.split(" ")
return spl[2]
		`),
				},
			},
			starlark.String(runtime.Version()),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out, err := EvalComputedAttribute(tc.attr, res, tc.attr.Val.(*vts.ComputedValue), &vts.RunnerEnv{})
			if err != nil {
				t.Fatalf("EvalComputedAttribute() failed: %v", err)
			}
			if diff := cmp.Diff(tc.expected, out, cmp.AllowUnexported(starlark.Int{})); diff != "" {
				t.Errorf("unexpected result (+got, -want):\n%s", diff)
			}
		})
	}
}
