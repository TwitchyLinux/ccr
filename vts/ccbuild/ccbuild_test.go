package ccbuild

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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
			tt := strings.Replace(s.targets[0].Type().String(), "_", "", -1)
			if got, want := tt, targetType; got != want {
				t.Errorf("target.type = %v, want %v", got, want)
			}
		})
	}
}
