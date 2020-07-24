// Package buildstep implements individual build steps.
package buildstep

import (
	"io"

	"gopkg.in/src-d/go-billy.v4"
)

type RunningBuild interface {
	OverlayMountPath() string
	OverlayUpperPath() string
	RootFS() billy.Filesystem
	SourceFS() billy.Filesystem
	ExecBlocking(args []string, stdout, stderr io.Writer) error
}
