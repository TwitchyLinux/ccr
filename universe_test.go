package ccr

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		fqTargets:      map[string]vts.GlobalTarget{},
		logger:         &silentOpTrack{},
		classedTargets: map[vts.Target][]vts.GlobalTarget{},
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

func TestUniverseMustBuildFirst(t *testing.T) {
	uv := Universe{
		fqTargets:      map[string]vts.GlobalTarget{},
		logger:         &silentOpTrack{},
		classedTargets: map[vts.Target][]vts.GlobalTarget{},
	}

	t.Run("check", func(t *testing.T) {
		if err := uv.Check([]vts.TargetRef{{Path: "//root:root_thing"}}, "wut"); err != ErrNotBuilt {
			t.Errorf("universe.Check(%q) failed: %v", "//root:root_thing", err)
		}
	})
	t.Run("generate", func(t *testing.T) {
		if err := uv.Generate(GenerateConfig{}, vts.TargetRef{Path: "//root:root_thing"}, "wut"); err != ErrNotBuilt {
			t.Errorf("universe.Generate(%q) failed: %v", "//root:root_thing", err)
		}
	})
}

func TestUniverseBuild(t *testing.T) {
	uv := Universe{
		fqTargets:      map[string]vts.GlobalTarget{},
		logger:         &silentOpTrack{},
		classedTargets: map[vts.Target][]vts.GlobalTarget{},
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
		if src, hasSrc := target.(vts.SourcedTarget); hasSrc {
			if src.Src() != nil && src.Src().Target == nil {
				t.Errorf("%s: src.Target = nil, want non-nil", target.GlobalPath())
			}
		}
		if inputs, hasInputs := target.(vts.InputTarget); hasInputs {
			for i, inp := range inputs.NeedInputs() {
				if inp.Target == nil {
					t.Errorf("%s: inputs[%d].Target = nil, want non-nil", target.GlobalPath(), i)
				}
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
			targets: []vts.TargetRef{{Path: "//file_resource:empty_path"}},
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
		{
			name:    "component_checker_fail",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//component:should_fail"}},
			err:     "debug: returning error",
		},
		{
			name:    "file_source_no_path",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//path_attr:no_path"}},
			err:     "no path specified",
		},
		{
			name:    "file_source_empty_path",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//path_attr:empty_path"}},
			err:     "path was empty",
		},
		{
			name:    "file_source",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//component:goody"}},
		},
		{
			name:    "file_not_exist",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//file_resource:not_exist"}},
			err:     "stat testdata/checkers/base/missing.json: no such file or directory",
		},
		{
			name:    "good_octal",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//octal:good_octal"}},
		},
		{
			name:    "good_octal_with_helper",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//octal:good_octal_with_helper"}},
		},
		{
			name:    "invalid_octal",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//octal:invalid_octal"}},
			err:     "invalid mode: char '8' is not a valid octal character",
		},
		{
			name:    "invalid_octal_with_helper",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//octal:invalid_octal_with_helper"}},
			err:     "invalid mode: char '8' is not a valid octal character",
		},
		{
			name:    "bool_false",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//bool:false"}},
		},
		{
			name:    "bool_true",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//bool:true"}},
		},
		{
			name:    "bool_invalid",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//bool:invalid"}},
			err:     "attr is not a boolean: got type starlark.String",
		},
		{
			name:    "deb_info_bad_type",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//deb:bad_type"}},
			err:     "expected list, got starlark.String",
		},
		{
			name:    "deb_info_good",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//deb:good"}},
		},
		{
			name:    "deb_info_unexpected_key",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//deb:unexpected_key"}},
			err:     "unexpected key: waht",
		},
		{
			name:    "deb_info_dep_unexpected_key",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//deb:deb_info_dep_unexpected_key"}},
			err:     "depends-on entry has unexpected sub-key \"waht\"",
		},
		{
			name:    "deb_info_bad_deps",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//deb:bad_dep_list"}},
			err:     "depends-on entry is not a dictionary",
		},
		{
			name:    "dir_good",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//dir:good"}},
		},
		{
			name:    "dir_good_nested",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//dir:good_nested"}},
		},
		{
			name:    "dir_bad_mode",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//dir:bad_perms"}},
			err:     "permissions mismatch: 0600 was specified but directory is 0755",
		},
		{
			name:    "dir_bad_missing",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//dir:bad_missing_dir"}},
			err:     "stat testdata/checkers/base/missing_dir: no such file or directory",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			uv := NewUniverse(&silentOpTrack{}, nil)
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

func TestUniverseGenerate(t *testing.T) {
	tcs := []struct {
		name         string
		target       string
		config       GenerateConfig
		testManifest string
		hasFiles     map[string]os.FileMode
		err          string
	}{
		{
			name:   "basic",
			target: "//basic:collect_resources",
			config: GenerateConfig{},
			testManifest: `Generator: //basic:test_manifest_generator
Resource: //basic:manifest
Direct: *vts.Resource @//basic:part_of_it_by_dep
Class: //basic:whelp
-//basic:yeet
-//basic:yolo
-//basic:swaggins

`,
		},
		{
			name:   "circular_component",
			target: "//circular:circ_component",
			config: GenerateConfig{},
			err:    "circular dependency: //circular:circ_component -> //circular:c1 -> //circular:gen",
		},
		{
			name:   "circular_resource",
			target: "//circular:circ_resource",
			config: GenerateConfig{},
			err:    "circular dependency: //circular:r2 -> //circular:c3 -> //circular:gen2",
		},
		{
			name:         "file",
			target:       "//mk_file:test_file_puesdo",
			config:       GenerateConfig{},
			testManifest: "Fake contents!!\n",
		},
		{
			name:   "file_with_mode",
			target: "//mk_file:fake_file_with_mode",
			config: GenerateConfig{},
			hasFiles: map[string]os.FileMode{
				"dir/dat_file.txt": os.FileMode(0600),
			},
		},
		{
			name:   "dir_good",
			target: "//mk_dir:good",
			config: GenerateConfig{},
			hasFiles: map[string]os.FileMode{
				"newdir": os.FileMode(os.ModeDir | 0755),
			},
		},
		{
			name:   "dir_no_mode",
			target: "//mk_dir:breaks_no_mode",
			config: GenerateConfig{},
			err:    "cannot generate dir when no mode was specified",
		},
		{
			name:   "deb_src",
			target: "//deb-libwoff1:libwoff1",
			config: GenerateConfig{},
			hasFiles: map[string]os.FileMode{
				"/usr/lib/x86_64-linux-gnu":                         os.FileMode(os.ModeDir | 0755),
				"/usr/lib/x86_64-linux-gnu/libwoff2common.so.1.0.2": os.FileMode(0644),
				"/usr/lib/x86_64-linux-gnu/libwoff2dec.so.1.0.2":    os.FileMode(0644),
				"/usr/lib/x86_64-linux-gnu/libwoff2enc.so.1.0.2":    os.FileMode(0644),
			},
		},
		{
			name:   "deb_bad_hash",
			target: "//deb:deb_has_bad_hash",
			config: GenerateConfig{},
			err:    "sha256 mismatch: got d2e9dd92dd3f1bdbafd63b4a122415d28fecc5f6152d82fa0f76a9766d95ba17 but expected aabbccddeeffggwahhhhhht",
		},
		{
			name:   "deb_invalid",
			target: "//deb:deb_invalid",
			config: GenerateConfig{},
			err:    "failed decoding deb: unexpected EOF",
		},
	}

	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	cache, err := NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			uv := NewUniverse(&silentOpTrack{}, cache)
			dr := NewDirResolver("testdata/generators")
			findOpts := FindOptions{
				FallbackResolvers: []CCRResolver{dr.Resolve},
				PrefixResolvers: map[string]CCRResolver{
					"common": common.Resolve,
				},
			}

			td, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(td)

			if err := uv.Build([]vts.TargetRef{{Path: tc.target}}, &findOpts); err != nil {
				t.Fatalf("universe.Build(%q) failed: %v", tc.target, err)
			}

			err = uv.Generate(tc.config, vts.TargetRef{Path: tc.target}, td)
			switch {
			case err == nil && tc.err != "":
				t.Errorf("universe.Generate(%q) returned no error, want %q", tc.target, tc.err)
			case err != nil && tc.err != err.Error():
				t.Errorf("universe.Generate(%q) returned %v, want %q", tc.target, err, tc.err)
			}

			if mfPath := filepath.Join(td, "test_manifest.txt"); tc.testManifest != "" {
				st, err := os.Stat(mfPath)
				if err != nil {
					t.Errorf("Failed to stat test manifest: %v", err)
				}
				man, err := ioutil.ReadFile(mfPath)
				if err != nil {
					t.Errorf("Failed to read test manifest: %v", err)
				}
				if diff := cmp.Diff(strings.Split(tc.testManifest, "\n"), strings.Split(string(man), "\n")); diff != "" {
					t.Errorf("Manifests contents do not match test (+got, -want): \n%s", diff)
				}

				if got, want := st.Mode()&os.ModePerm, os.FileMode(0644); got != want {
					t.Errorf("test manifest mode is %#o, want %#o", got, want)
				}
			}

			for p, m := range tc.hasFiles {
				st, err := os.Stat(filepath.Join(td, p))
				if err != nil {
					t.Error(err)
					continue
				}
				if st.Mode() != m {
					t.Errorf("file %q has mode %#o, want %#o", p, st.Mode(), m)
				}
			}
		})
	}
}
