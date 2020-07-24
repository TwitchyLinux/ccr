package ccr

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
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
				{Target: &vts.Attr{
					Parent: vts.TargetRef{Path: "common://attrs:path"},
					Val:    starlark.String("/other_thing"),
				}},
			},
			Source: &vts.TargetRef{Path: "//root:some_build"},
		}, nil
	case "//root:some_build":
		return &vts.Build{
			Path: "//root:some_build",
			Name: "some_build",
			HostDeps: []vts.TargetRef{
				{
					Path: "common://toolchains:go",
					Constraints: []vts.RefConstraint{
						{
							Meta:   vts.TargetRef{Target: common.SemverClass},
							Params: []starlark.Value{starlark.String(">>"), starlark.String("1.2.3")},
							Eval:   &ccbuild.RefComparisonConstraint{},
						},
					},
				},
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

	if err := uv.Build([]vts.TargetRef{{Path: "//yeet:floop"}}, &findOpts, ""); err != nil {
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
		pathTargets:    map[string]vts.Target{},
	}
	findOpts := FindOptions{
		FallbackResolvers: []CCRResolver{testResolver},
		PrefixResolvers: map[string]CCRResolver{
			"common": common.Resolve,
		},
	}

	if err := uv.Build([]vts.TargetRef{{Path: "//root:root_thing"}}, &findOpts, ""); err != nil {
		t.Errorf("universe.Build(%q) failed: %v", "//root:root_thing", err)
	}

	// Confirm all the targets we expected were loaded.
	for _, path := range []string{"//root:root_thing", "common://attrs/arch:amd64", "common://attrs:arch", "//distant/yolo:reee", "//root:other_thing"} {
		if _, exists := uv.fqTargets[path]; !exists {
			t.Errorf("target %q not present", path)
		}
	}

	// Confirm a few targets with paths were inserted into runtimeFinder.PathTargets.
	for _, path := range []string{"/other_thing"} {
		if _, ok := uv.pathTargets[path]; !ok {
			t.Errorf("target with path attribute was not tracked in PathTargets[%q]", path)
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
		if hostDeps, hasHostDeps := target.(vts.HostDepTarget); hasHostDeps {
			for i, d := range hostDeps.HostDependencies() {
				if d.Target == nil {
					t.Errorf("%s: host_deps[%d].Target = nil, want non-nil", target.GlobalPath(), i)
				}
				for x, c := range d.Constraints {
					if c.Meta.Target == nil {
						t.Errorf("%s: host_deps[%d].Constraint[%d].Meta.Target = nil, want non-nil", target.GlobalPath(), i, x)
					}
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
			name:    "filelist_good",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//file_resource:filelist_good"}},
		},
		{
			name:    "filelist_missing_files",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//file_resource:filelist_missing_files"}},
			err:     "referencing file at line 3: stat testdata/checkers/base/waht: no such file or directory",
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
			name:    "deb_info_bad_tyruntimeFinderpe",
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
		{
			name:    "symlink_missing",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//symlink:bad_missing"}},
			err:     "lstat testdata/checkers/base/missing_link: no such file or directory",
		},
		{
			name:    "semver_good_1",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//semver_attr:simple"}},
		},
		{
			name:    "semver_good_2",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//semver_attr:normal"}},
		},
		{
			name:    "semver_empty",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//semver_attr:empty"}},
			err:     "invalid version \"\": strconv.ParseUint: parsing \"\": invalid syntax",
		},
		{
			name:    "semver_invalid_deb",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//semver_attr:bad_semver_1"}},
			err:     "invalid version \"1.2:3\": semvers cannot contain a trailing epoch",
		},
		{
			name:    "semver_invalid_letters",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//semver_attr:bad_semver_2"}},
			err:     "invalid version \"a\": Invalid character(s) found in major number \"a\"",
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

			if err := uv.Build(tc.targets, &findOpts, tc.base); err != nil {
				t.Fatalf("universe.Build(%q) failed: %v", tc.targets, err)
			}

			err := uv.Check(tc.targets, tc.base)
			switch {
			case err == nil && tc.err != "":
				t.Errorf("universe.Check(%q) returned no error, want %q", tc.targets, tc.err)
			case err != nil && tc.err != err.Error():
				t.Errorf("universe.Check(%q) returned %q, want %q", tc.targets, err, tc.err)
			}
		})
	}
}

func TestBuildValidation(t *testing.T) {
	tcs := []struct {
		name    string
		base    string
		targets []vts.TargetRef
		err     string
	}{
		{
			name:    "multiple_exclusive_attrs",
			base:    "testdata/checkers/base",
			targets: []vts.TargetRef{{Path: "//multiple_exclusive_attrs:path"}},
			err:     "duplicate attributes with non-repeatable class \"common://attrs:path\"",
		},
		{
			name:    "resource with source",
			base:    "testdata/basic/nested",
			targets: []vts.TargetRef{{Path: "//build:cat"}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			uv := NewUniverse(&silentOpTrack{}, nil)
			dr := NewDirResolver(filepath.Dir(tc.base))
			findOpts := FindOptions{
				FallbackResolvers: []CCRResolver{dr.Resolve},
				PrefixResolvers: map[string]CCRResolver{
					"common": common.Resolve,
				},
			}

			err := uv.Build(tc.targets, &findOpts, tc.base)
			switch {
			case err == nil && tc.err != "":
				t.Errorf("universe.Build(%q) returned no error, want %q", tc.targets, tc.err)
			case err != nil && tc.err != err.Error():
				t.Errorf("universe.Build(%q) returned %q, want %q", tc.targets, err, tc.err)
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
		{
			name:   "symlink_no_target",
			target: "//symlink:bad_missing_target",
			config: GenerateConfig{},
			err:    "cannot generate symlink when no target was specified",
		},
		{
			name:   "symlink_good",
			target: "//symlink:good",
			config: GenerateConfig{},
			hasFiles: map[string]os.FileMode{
				"/good_link": os.FileMode(0644),
			},
		},
		{
			name:   "toolchain_check_missing",
			target: "//toolchain_checkers:missing",
			err:    "toolchain component missing: ls",
			config: GenerateConfig{},
		},
		{
			name:   "basic_build",
			target: "//basic_build:output",
			config: GenerateConfig{},
		},
		{
			name:   "exec_build",
			target: "//basic_build:exec_output",
			config: GenerateConfig{},
			hasFiles: map[string]os.FileMode{
				"/out.txt": os.FileMode(0600),
			},
		},
		{
			name:   "artifact_missing_in_build",
			target: "//basic_build:output_missing_in_build",
			err:    "file missing from build output",
			config: GenerateConfig{},
		},
		{
			name:   "host_dep_missing",
			target: "//basic_build:output_missing_host_dep",
			err:    "toolchain component missing: something_missing",
			config: GenerateConfig{},
		},
		{
			name:   "should_exist",
			target: "//toolchain_checkers:should_exist",
			config: GenerateConfig{},
		},
		{
			name:   "hostdep_constraint_gt",
			target: "//build_hostdep_constraint:passing_gt_constraint",
			config: GenerateConfig{},
		},
		{
			name:   "hostdep_constraint_lt",
			target: "//build_hostdep_constraint:passing_lt_constraint",
			config: GenerateConfig{},
		},
		{
			name:   "hostdep_constraint_semver_fail",
			target: "//build_hostdep_constraint:failing_constraint",
			err:    "semver constraint was not met",
			config: GenerateConfig{},
		},
	}

	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	cache, err := cache.NewCache(cd)
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

			if err := uv.Build([]vts.TargetRef{{Path: tc.target}}, &findOpts, td); err != nil {
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

func TestSystemLibraryStuff(t *testing.T) {
	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	cache, err := cache.NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	uv := NewUniverse(&silentOpTrack{}, cache)
	dr := NewDirResolver("testdata/syslibs")
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

	if err := uv.Build([]vts.TargetRef{{Path: "//core:core"}}, &findOpts, td); err != nil {
		t.Fatalf("universe.Build(%q) failed: %v", "//core:core", err)
	}

	err = uv.Generate(GenerateConfig{}, vts.TargetRef{Path: "//core:core"}, td)
	if err != nil {
		t.Errorf("universe.Generate(\"//core:core\") returned %v, want nil", err)
	}
}

func TestBuildComputedAttrValues(t *testing.T) {
	wd, _ := os.Getwd()
	tcs := []struct {
		name   string
		base   string
		target vts.TargetRef
		err    string
		attrs  map[string]starlark.Value
	}{
		{
			name:   "basic",
			base:   "testdata/basic",
			target: vts.TargetRef{Path: "//computed_attr:computed"},
			attrs: map[string]starlark.Value{
				common.PathClass.Path: starlark.String("computed_dir"),
			},
		},
		{
			name:   "missing_compute_file",
			base:   "testdata/basic",
			target: vts.TargetRef{Path: "//computed_attr:missing_macro_file"},
			err:    "open testdata/compute/missing.py: no such file or directory",
		},
		{
			name:   "inline",
			base:   "testdata/basic",
			target: vts.TargetRef{Path: "//computed_attr:inline"},
			attrs: map[string]starlark.Value{
				common.PathClass.Path: starlark.String("value_inline"),
			},
		},
		{
			name:   "wd",
			base:   "testdata/basic",
			target: vts.TargetRef{Path: "//computed_attr:wd"},
			attrs: map[string]starlark.Value{
				common.PathClass.Path: starlark.String(filepath.Join(wd, "testdata/compute")),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			uv := NewUniverse(&silentOpTrack{}, nil)
			dr := NewDirResolver("testdata/compute")
			findOpts := FindOptions{
				FallbackResolvers: []CCRResolver{dr.Resolve},
				PrefixResolvers: map[string]CCRResolver{
					"common": common.Resolve,
				},
			}

			err := uv.Build([]vts.TargetRef{tc.target}, &findOpts, tc.base)
			switch {
			case err == nil && tc.err != "":
				t.Errorf("universe.Build(%q) returned no error, want %q", tc.target, tc.err)
			case err != nil && tc.err != err.Error():
				t.Errorf("universe.Build(%q) returned %q, want %q", tc.target, err, tc.err)
			}

			for p, val := range tc.attrs {
				v, err := uv.QueryByClass(tc.base, tc.target.Path, p)
				if err != nil {
					t.Errorf("failed querying for attr %q: %v", p, err)
				}
				if !reflect.DeepEqual(v, val) {
					t.Errorf("attr %q: got value %v, want %v", p, v, val)
				}
			}
		})
	}
}

func TestBuildFailsDuplicatePaths(t *testing.T) {
	cd, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cd)
	cache, err := cache.NewCache(cd)
	if err != nil {
		t.Fatal(err)
	}

	uv := NewUniverse(&silentOpTrack{}, cache)
	dr := NewDirResolver("testdata/basic")
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

	f := "//dupe_paths"
	out := uv.Build([]vts.TargetRef{{Path: "//dupe_paths:fail"}}, &findOpts, td).(vts.WrappedErr)
	var want = vts.WrappedErr{
		Path: "/some/path",
		Err:  errors.New("multiple targets declared the same path"),
		Target: &vts.Resource{
			Path: "//dupe_paths:thing1",
			Name: "thing1",
			Pos: &vts.DefPosition{
				Path:  "testdata/basic/dupe_paths.ccr",
				Frame: starlark.CallFrame{Name: "<toplevel>", Pos: syntax.MakePosition(&f, 9, 9)},
			},
		},
		ActionTarget: &vts.Resource{
			Path: "//dupe_paths:thing2",
			Name: "thing2",
			Pos: &vts.DefPosition{
				Path:  "testdata/basic/dupe_paths.ccr",
				Frame: starlark.CallFrame{Name: "<toplevel>", Pos: syntax.MakePosition(&f, 15, 9)},
			},
		},
	}
	if out.Err.Error() != want.Err.Error() {
		t.Errorf("Incorrect error string: got %q, want %q", out.Err.Error(), want.Err.Error())
	}

	if diff := cmp.Diff(out, want, cmpopts.IgnoreTypes(vts.TargetRef{}),
		cmpopts.IgnoreFields(vts.Resource{}, "Pos", "Details"),
		cmpopts.IgnoreFields(vts.WrappedErr{}, "Err")); diff != "" {
		t.Fatalf("universe.Build(%q) failed: \n%s", "//dupe_paths:fail", diff)
	}
}
