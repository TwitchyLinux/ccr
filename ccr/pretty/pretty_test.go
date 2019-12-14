package pretty

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	simpleRE = regexp.MustCompile("testdata/(.*)_in\\.ccr")
)

func TestFormatCCR(t *testing.T) {
	scripts, err := filepath.Glob("testdata/*_in.ccr")
	if err != nil {
		t.Fatal(err)
	}

	for _, fPath := range scripts {
		var (
			prefix  = simpleRE.FindAllStringSubmatch(fPath, 1)[0][1]
			inPath  = "testdata/" + prefix + "_in.ccr"
			outPath = "testdata/" + prefix + "_out.ccr"
		)

		t.Run(prefix, func(t *testing.T) {
			changed, out, err := FormatCCR(inPath)
			if err != nil {
				t.Fatalf("FormatCCR(%q) failed: %v", inPath, err)
			}

			inF, err := ioutil.ReadFile(inPath)
			if err != nil {
				t.Fatal(err)
			}
			outF, err := ioutil.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := changed, !bytes.Equal(inF, outF); got != want {
				t.Errorf("changed = %v, want %v", got, want)
			}

			if diff := cmp.Diff(string(outF), out.String()); diff != "" {
				t.Errorf("output mismatch (+got, -want): \n%s", diff)
			}
		})
	}
}
