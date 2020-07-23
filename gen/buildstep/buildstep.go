// Package buildstep implements individual build steps.
package buildstep

import "gopkg.in/src-d/go-billy.v4"

type RunningBuild interface {
	Dir() string
	BuildFS() billy.Filesystem
	SourceFS() billy.Filesystem
}
