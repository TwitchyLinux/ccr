package vts

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/gobwas/glob"
	"github.com/twitchylinux/ccr/vts/match"
	"go.starlark.net/starlark"
)

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	h, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return h
}

func TestRollupHash(t *testing.T) {
	tcs := []struct {
		name   string
		target ReproducibleTarget
		want   []byte
		err    string
	}{
		{
			"toolchain",
			&Toolchain{Name: "go", Path: "common://toolchains:go", BinaryMappings: map[string]string{
				"go":    "/usr/local/go/bin/go",
				"gofmt": "/usr/local/go/bin/gofmt",
			}},
			mustDecodeHex(t, "A5261B9F19737D815CC4C4C967ABF358E195627B114A38A6170EF73962735A72"),
			"",
		},
		{
			"toolchain with attr",
			&Toolchain{Name: "go", Path: "common://toolchains:go", Details: []TargetRef{
				{Target: &Attr{Name: "something", Val: starlark.String("abc"), Parent: TargetRef{Target: &AttrClass{}}}},
			}},
			mustDecodeHex(t, "9A15452BA3BB49A3E97D790051DE73A394717E0CBEDF9CCC282A6E02AD3FCE99"),
			"",
		},
		{
			"toolchain with computed attr",
			&Toolchain{Name: "go", Path: "common://toolchains:go", Details: []TargetRef{
				{Target: &Attr{Name: "something", Val: &ComputedValue{Filename: "a.py", Func: "something"}, Parent: TargetRef{Target: &AttrClass{}}}},
			}},
			mustDecodeHex(t, "DA6AEF9F00E8FF800B58BE2848FB0D16F61EC796BBD9516A0402D6524A139856"),
			"",
		},
		{
			"toolchain with resource dep",
			&Toolchain{Deps: []TargetRef{{Target: &Resource{
				Name:    "waht",
				Parent:  TargetRef{Target: &ResourceClass{Path: "abc"}},
				Details: []TargetRef{{Target: &Attr{Name: "something", Val: starlark.String("abc"), Parent: TargetRef{Target: &AttrClass{Path: "cbaz"}}}}},
			}}}},
			mustDecodeHex(t, "64B82626B08FE989546E2AEDD37C5E837741EE32C09EFD3B54FE8207A54FEECA"),
			"",
		},
		{
			"toolchain with component dep",
			&Toolchain{Deps: []TargetRef{{Target: &Component{Name: "waht"}}}},
			mustDecodeHex(t, "899810096981DD85DABD499ADA519DF0BC1EBA44FC5E45D742C4112CA84FA293"),
			"",
		},
		{
			"toolchain with component dep 2",
			&Toolchain{Deps: []TargetRef{{Target: &Component{Name: "blueberry"}}}},
			mustDecodeHex(t, "B5F40458F6CD470D0F396B591B58452BD3AA175079537275CDBBF82F3420B9DC"),
			"",
		},
		{
			"component with toolchain dep",
			&Component{Deps: []TargetRef{{Target: &Toolchain{Name: "blueberry"}}}},
			mustDecodeHex(t, "AB2998DBA8D8B099A747FEEE8A28077515A519D415505057D66B6A05CA30B210"),
			"",
		},
		{
			"generator",
			&Generator{Name: "users", Path: "//systems:users_list"},
			mustDecodeHex(t, "ADD57E80916543636E36480D73B2FA4B216618AE59442ACACCEAB6AC1B690184"),
			"",
		},
		{
			"puesdo",
			&Puesdo{Name: "users", Path: "//systems:users_list", Kind: FileRef, SHA256: "EDA1AF8391DAAE70543512FBEE98185454B26FE136479CA5CEDFA5AD13FB4F2F"},
			mustDecodeHex(t, "1D1DB3BC7625A100018B4C58803C16B528D5650878C85BCDB6BECF6BEF20E05D"),
			"",
		},
		{
			"sieve",
			&Sieve{Name: "users", TargetPath: "//systems:users_list",
				AddPrefix: "/usr/", ExcludeGlobs: []string{"*.txt", "*.ccr"},
				Inputs: []TargetRef{{Target: &Generator{Name: "users", Path: "//systems:users_list"}}},
				Renames: &match.FilenameRules{
					Rules: []match.MatchRule{
						{P: glob.MustCompile("cool.txt"), Out: match.LiteralOutputMapper("kek.txt")},
					},
				}},
			mustDecodeHex(t, "A06A624DDF7CAFF4D37CA8576246C0283B2245D2A24C47446B034350C1D8E0A0"),
			"",
		},
		{
			"puesdo with attr",
			&Puesdo{Name: "something", Path: "//waht:something", Details: []TargetRef{
				{Target: &Attr{Name: "val", Val: starlark.String("abc"), Parent: TargetRef{Target: &AttrClass{}}}},
			}},
			mustDecodeHex(t, "249BF5B2D671B0F4AFBDBB30C1A2E2B2D747651F863D94FE13825E8E4C888648"),
			"",
		},
		{
			"unhashable dep",
			&Toolchain{Deps: []TargetRef{{Target: &AttrClass{}}}},
			nil,
			"cannot compute rollup hash on non-reproducible target of type *vts.AttrClass",
		},
		{
			"build",
			&Build{Name: "users", Path: "//systems:users_list",
				Steps: []*BuildStep{
					{Kind: StepUnpackGz, Path: "go1.11.4.tar.gz", ToPath: "src"},
					{Kind: StepConfigure, Dir: "/tmp/aaa", NamedArgs: map[string]string{
						"with-prefix": "/usr",
						"something":   "else",
					}},
				},
				HostDeps: []TargetRef{{Target: &Component{Name: "mmkay"}}},
				Output: &match.FilenameRules{
					Rules: []match.MatchRule{
						{P: glob.MustCompile("cool.txt"), Out: match.LiteralOutputMapper("cool.txt")},
						{P: glob.MustCompile("*.txt"), Out: match.LiteralOutputMapper("/kek.txt")},
						{P: glob.MustCompile("*.go"), Out: &match.StripPrefixOutputMapper{Prefix: "/usr/local/go/src"}},
					},
				},
				PatchIns: map[string]TargetRef{"/cool.txt": TargetRef{Target: &Puesdo{Kind: FileRef, Path: "cool.txt"}}},
				Env: map[string]starlark.Value{
					"yeet": starlark.String("123"),
					"noot": starlark.String(":)"),
				},
			},
			mustDecodeHex(t, "A038737A2811E5418749D64ECFCD33942FDAEC80FFA7F12E712E43B1C18427EF"),
			"",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, err := tc.target.RollupHash(nil, func(attr *Attr, target Target, runInfo *ComputedValue, env *RunnerEnv) (starlark.Value, error) {
				return starlark.String("computed"), nil
			})
			if err != nil && err.Error() != tc.err {
				t.Fatalf("RollupHash() failed: %v", err)
			}

			if !bytes.Equal(h, tc.want) {
				t.Errorf("hash = %X, want %X", h, tc.want)
			}
		})
	}
}
