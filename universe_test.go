package ccr

import (
	"os"
	"testing"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
)

func testResolver(path string) (vts.Target, error) {
	switch path {
	case "//root:root_thing":
		return &vts.Component{
			Path: path,
			Name: "root_thing",
			Deps: []vts.TargetRef{
				{Path: "//root:other_thing"},
			},
		}, nil
	case "//root:other_thing":
		return &vts.Resource{
			Path:   path,
			Parent: vts.TargetRef{Path: "//distant/yolo:reee"},
			Name:   "other_thing",
			Details: []vts.TargetRef{
				{Path: "common://attrs/arch:amd64"},
			},
		}, nil
	case "//distant/yolo:reee":
		return &vts.ResourceClass{
			Path: path,
			Name: "reee",
		}, nil
	}
	return nil, os.ErrNotExist
}

func TestDirResolver(t *testing.T) {
	uv := Universe{
		fqTargets: map[string]vts.GlobalTarget{},
		logger:    &silentOpTrack{},
	}
	dr := DirResolver{
		dir:     "testdata/basic",
		targets: map[string]vts.GlobalTarget{},
	}
	findOpts := FindOptions{
		FallbackResolvers: []CCRResolver{dr.Resolve},
		PrefixResolvers: map[string]CCRResolver{
			"common": common.Resolve,
		},
	}

	if err := uv.Build([]vts.TargetRef{{Path: "//yeet:floop"}}, &findOpts); err != nil {
		t.Errorf("universe.Build(%q) failed: %v", "//yeet:floop", err)
	}
}

func TestUniverseBuild(t *testing.T) {
	uv := Universe{
		fqTargets: map[string]vts.GlobalTarget{},
		logger:    &silentOpTrack{},
	}
	findOpts := FindOptions{
		FallbackResolvers: []CCRResolver{testResolver},
		PrefixResolvers: map[string]CCRResolver{
			"common": common.Resolve,
		},
	}

	if err := uv.Build([]vts.TargetRef{{Path: "//root:root_thing"}}, &findOpts); err != nil {
		t.Errorf("universe.Build(%q) failed: %v", "//root:root_thing", err)
	}

	// Confirm all the targets we expected were loaded.
	for _, path := range []string{"//root:root_thing", "common://attrs/arch:amd64", "common://attrs:arch", "//distant/yolo:reee", "//root:other_thing"} {
		if _, exists := uv.fqTargets[path]; !exists {
			t.Errorf("target %q not present", path)
		}
	}

	for p, target := range uv.fqTargets {
		if p != target.GlobalPath() {
			t.Errorf("target.Path = %q, want %q", target.GlobalPath(), p)
		}

		// Confirm all targets reference other targets by path.
		if deps, hasDeps := target.(vts.DepTarget); hasDeps {
			for i, dep := range deps.Dependencies() {
				if dep.Target == nil {
					t.Errorf("%s: dep[%d].Target = nil, want non-nil", target.GlobalPath(), i)
				}
			}
		}
		if chks, hasChecks := target.(vts.CheckedTarget); hasChecks {
			for i, chk := range chks.Checkers() {
				if chk.Target == nil {
					t.Errorf("%s: chk[%d].Target = nil, want non-nil", target.GlobalPath(), i)
				}
			}
		}
		if attrs, hasAttrs := target.(vts.DetailedTarget); hasAttrs {
			for i, attr := range attrs.Attributes() {
				if attr.Target == nil {
					t.Errorf("%s: attr[%d].Target = nil, want non-nil", target.GlobalPath(), i)
				}
			}
		}
		if class, hasClass := target.(vts.ClassedTarget); hasClass {
			if class.Class().Target == nil {
				t.Errorf("%s: class.Target = nil, want non-nil", target.GlobalPath())
			}
		}
	}
}

func TestUniverseCheck(t *testing.T) {
	tcs := []struct {
		name    string
		base    string
		targets []vts.TargetRef
		err     string
	}{
		{
			name:    "resource check json good",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//json:good_json"}},
		},
		{
			name:    "resource check json bad",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//json:bad_json"}},
			err:     "invalid character 'd' in literal false (expecting 'a')",
		},
		{
			name:    "resource check json missing",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//json:missing_json"}},
			err:     "open testdata/checkers/base/missing.json: no such file or directory",
		},
		{
			name:    "bad_path",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//path:bad_path"}},
			err:     "path contains illegal characters",
		},
		{
			name:    "empty_path",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//path:empty_path"}},
			err:     "path was empty",
		},
		{
			name:    "good_path",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//path:good_path"}},
		},
		{
			name:    "good_enum",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//enum:good_enum"}},
		},
		{
			name:    "bad_enum",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//enum:bad_enum"}},
			err:     "invalid value: \"swiggity\"",
		},
		{
			name:    "component_checker",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//component:ls"}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			uv := NewUniverse(&silentOpTrack{})
			dr := NewDirResolver("testdata/checkers")
			findOpts := FindOptions{
				FallbackResolvers: []CCRResolver{dr.Resolve},
				PrefixResolvers: map[string]CCRResolver{
					"common": common.Resolve,
				},
			}

			if err := uv.Build(tc.targets, &findOpts); err != nil {
				t.Fatalf("universe.Build(%q) failed: %v", tc.targets, err)
			}

			err := uv.Check(tc.targets, tc.base)
			switch {
			case err == nil && tc.err != "":
				t.Errorf("universe.Check(%q) returned no error, want %q", tc.targets, tc.err)
			case err != nil && tc.err != err.Error():
				t.Errorf("universe.Check(%q) returned %v, want %q", tc.targets, err, tc.err)
			}
		})
	}
}
