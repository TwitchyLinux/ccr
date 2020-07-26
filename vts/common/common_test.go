package common

import (
	"strings"
	"testing"

	"github.com/twitchylinux/ccr/vts"
)

func TestTargetNameAndPathConsistent(t *testing.T) {
	for p, target := range commonTargets {
		ct := target.(vts.GlobalTarget)

		if gp := ct.GlobalPath(); p != gp {
			t.Errorf("%q: declared path does not match path in target %q", p, gp)
		}
		spl := strings.Split(p, ":")
		if want, got := ct.TargetName(), spl[len(spl)-1]; want != got {
			t.Errorf("%q: target name = %q, want %q", p, got, want)
		}
	}
}
